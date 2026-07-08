package web

import (
	"apex/internal/domain"
	"apex/internal/httputil"
	"apex/marketdata"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// errAlreadySubscribed is returned by getOrCreate when symbol/tf is already
// active. It is not a failure - handleSubscribe turns it into a 409 so the
// client's fetch/htmx swap is a no-op and no duplicate panel is added.
var errAlreadySubscribed = errors.New("already subscribed")

var validTimeframes = map[string]bool{"1Min": true, "5Min": true}

type subKey struct {
	symbol string
	tf     string
}

type sseEvent struct {
	name string
	data []byte
}

type backfillEvent struct {
	Bars int    `json:"bars"`
	Err  string `json:"error,omitempty"`
}

// liveSub tracks one active symbol/timeframe subscription and fans its live
// bars and backfill-completion event out to any number of SSE listeners.
type liveSub struct {
	key    subKey
	cancel context.CancelFunc

	mu               sync.Mutex
	listeners        map[int]chan sseEvent
	nextID           int
	backfilled       bool
	lastBackfillBars int
	lastBackfillErr  string
}

func (s *liveSub) addListener() (int, chan sseEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID
	s.nextID++
	ch := make(chan sseEvent, 16)
	s.listeners[id] = ch
	return id, ch
}

func (s *liveSub) removeListener(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.listeners, id)
}

func (s *liveSub) snapshotBackfill() (bool, int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backfilled, s.lastBackfillBars, s.lastBackfillErr
}

func (s *liveSub) markBackfilled(bars int, errStr string) {
	s.mu.Lock()
	s.backfilled = true
	s.lastBackfillBars = bars
	s.lastBackfillErr = errStr
	s.mu.Unlock()
}

func (s *liveSub) broadcast(name string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.listeners {
		select {
		case ch <- sseEvent{name: name, data: data}:
		default:
		}
	}
}

// dashboard holds the set of active symbol/timeframe subscriptions backing
// the web dashboard. Subscriptions are keyed by (symbol, timeframe), persisted
// to the database, and restored on startup (subscribing to an already-active
// key is rejected with errAlreadySubscribed instead of creating a duplicate).
type dashboard struct {
	mkdata *marketdata.Module
	log    *slog.Logger
	ctx    context.Context // server-lifetime base context, not any single request's

	mu   sync.Mutex
	subs map[subKey]*liveSub
}

func Mount(mux *http.ServeMux, log *slog.Logger, mkdata *marketdata.Module, baseCtx context.Context) {
	mux.Handle("GET /static/", http.FileServerFS(staticFS))

	d := &dashboard{mkdata: mkdata, log: log, ctx: baseCtx, subs: make(map[subKey]*liveSub)}
	d.restoreSubscriptions()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		active := make([]subKey, 0, len(d.subs))
		for k := range d.subs {
			active = append(active, k)
		}
		d.mu.Unlock()
		sort.Slice(active, func(i, j int) bool {
			if active[i].symbol != active[j].symbol {
				return active[i].symbol < active[j].symbol
			}
			return active[i].tf < active[j].tf
		})
		if err := Index(active).Render(r.Context(), w); err != nil {
			log.Error("render index", slog.Any("error", err))
		}
	})

	mux.HandleFunc("POST /web/subscribe", d.handleSubscribe)
	mux.HandleFunc("GET /web/events", d.handleEvents)
	mux.HandleFunc("GET /web/bars", d.handleBars)
}

// startLiveSubscription starts live streaming for symbol/tf and registers it
// in d.subs. It does not touch backfill or persistence beyond what
// Module.Subscribe already does - callers decide whether a backfill is
// needed (a brand-new subscription) or not (restoring one from the DB).
func (d *dashboard) startLiveSubscription(symbol, tf string) (*liveSub, context.Context, error) {
	key := subKey{symbol: symbol, tf: tf}
	subCtx, cancel := context.WithCancel(d.ctx)
	sub := &liveSub{key: key, cancel: cancel, listeners: make(map[int]chan sseEvent)}

	if err := d.mkdata.Subscribe(subCtx, symbol, tf, func(bar domain.Bar) {
		data, err := json.Marshal(toChartBar(bar))
		if err != nil {
			return
		}
		sub.broadcast("bar", data)
	}); err != nil {
		cancel()
		return nil, nil, err
	}

	d.mu.Lock()
	d.subs[key] = sub
	d.mu.Unlock()

	return sub, subCtx, nil
}

// teardown removes symbol/tf from d.subs and unsubscribes it - used when a
// subscription turns out to be for a symbol with no data (bad ticker or
// genuinely empty range), so it doesn't linger as a dead live subscription or
// get restored again on the next restart. Unsubscribe runs on d.ctx, not
// sub's own subCtx, since sub.cancel() (called last) would otherwise make
// that context dead before Unsubscribe's DB/provider calls complete.
func (d *dashboard) teardown(symbol, tf string, sub *liveSub) {
	key := subKey{symbol: symbol, tf: tf}
	d.mu.Lock()
	delete(d.subs, key)
	d.mu.Unlock()

	if err := d.mkdata.Unsubscribe(d.ctx, symbol, tf); err != nil {
		d.log.Error("teardown unsubscribe", slog.String("symbol", symbol), slog.String("tf", tf), slog.Any("error", err))
	}
	sub.cancel()
}

// restoreSubscriptions re-establishes live streaming for every subscription
// persisted in the database, without re-running a backfill (historical data
// was already seeded the first time each symbol/tf was subscribed). It runs
// once, synchronously, before Mount registers any HTTP handlers, so d.subs
// is fully populated before the server can receive traffic.
//
// Persisted subscriptions with no stored bars (a stale row from a bad symbol
// subscribed before dead subscriptions were torn down) are skipped and
// cleaned up here instead of being restored, so they never reappear.
func (d *dashboard) restoreSubscriptions() {
	subs, err := d.mkdata.ListSubscriptions(d.ctx)
	if err != nil {
		d.log.Error("list persisted subscriptions", slog.Any("error", err))
		return
	}

	for _, s := range subs {
		bars, err := d.mkdata.ListBars(d.ctx, s.Symbol, s.Timeframe)
		if err != nil {
			d.log.Error("check stored bars for restore", slog.String("symbol", s.Symbol), slog.String("tf", s.Timeframe), slog.Any("error", err))
			continue
		}
		if len(bars) == 0 {
			d.log.Warn("skipping restore, no stored bars", slog.String("symbol", s.Symbol), slog.String("tf", s.Timeframe))
			if err := d.mkdata.Unsubscribe(d.ctx, s.Symbol, s.Timeframe); err != nil {
				d.log.Error("cleanup stale subscription", slog.String("symbol", s.Symbol), slog.String("tf", s.Timeframe), slog.Any("error", err))
			}
			continue
		}

		sub, _, err := d.startLiveSubscription(s.Symbol, s.Timeframe)
		if err != nil {
			d.log.Error("restore subscription", slog.String("symbol", s.Symbol), slog.String("tf", s.Timeframe), slog.Any("error", err))
			continue
		}
		sub.markBackfilled(len(bars), "")
		d.log.Info("restored subscription", slog.String("symbol", s.Symbol), slog.String("tf", s.Timeframe), slog.Int("n_bars", len(bars)))
	}
}

// getOrCreate starts a new subscription for symbol/tf: live streaming
// immediately, plus an async one-year backfill in the background. If
// symbol/tf is already subscribed, it returns errAlreadySubscribed instead of
// touching the existing subscription.
func (d *dashboard) getOrCreate(symbol, tf string) (*liveSub, error) {
	key := subKey{symbol: symbol, tf: tf}

	d.mu.Lock()
	if _, ok := d.subs[key]; ok {
		d.mu.Unlock()
		return nil, errAlreadySubscribed
	}
	d.mu.Unlock()

	sub, subCtx, err := d.startLiveSubscription(symbol, tf)
	if err != nil {
		return nil, err
	}

	end := time.Now()
	start := end.AddDate(-1, 0, 0)
	d.mkdata.Backfill(subCtx, symbol, tf, start, end, func(res marketdata.BackfillResult) {
		errStr := ""
		if res.Err != nil {
			errStr = res.Err.Error()
		}
		sub.markBackfilled(res.NumBars, errStr)
		if data, err := json.Marshal(backfillEvent{Bars: res.NumBars, Err: errStr}); err == nil {
			sub.broadcast("backfill_complete", data)
		}
		if res.Err != nil {
			d.teardown(symbol, tf, sub)
		}
	})

	return sub, nil
}

func (d *dashboard) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		if renderErr := SubscribeError("invalid form data").Render(r.Context(), w); renderErr != nil {
			d.log.Error("render subscribe error", slog.Any("error", renderErr))
		}
		return
	}

	symbol := strings.ToUpper(strings.TrimSpace(r.FormValue("symbol")))
	tf := r.FormValue("timeframe")

	if symbol == "" {
		if err := SubscribeError("symbol is required").Render(r.Context(), w); err != nil {
			d.log.Error("render subscribe error", slog.Any("error", err))
		}
		return
	}
	if !validTimeframes[tf] {
		if err := SubscribeError("timeframe must be 1Min or 5Min").Render(r.Context(), w); err != nil {
			d.log.Error("render subscribe error", slog.Any("error", err))
		}
		return
	}

	if _, err := d.getOrCreate(symbol, tf); err != nil {
		if errors.Is(err, errAlreadySubscribed) {
			// Non-2xx: htmx's default swap only applies to successful
			// responses, so this is a no-op on the client - no duplicate
			// panel is added.
			http.Error(w, "already subscribed", http.StatusConflict)
			return
		}
		d.log.Error("subscribe", slog.String("symbol", symbol), slog.String("tf", tf), slog.Any("error", err))
		if renderErr := SubscribeError(fmt.Sprintf("failed to subscribe to %s: %v", symbol, err)).Render(r.Context(), w); renderErr != nil {
			d.log.Error("render subscribe error", slog.Any("error", renderErr))
		}
		return
	}

	if err := ChartPanel(symbol, tf).Render(r.Context(), w); err != nil {
		d.log.Error("render chart panel", slog.Any("error", err))
	}
}

func (d *dashboard) handleEvents(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol")))
	tf := r.URL.Query().Get("timeframe")

	d.mu.Lock()
	sub, ok := d.subs[subKey{symbol: symbol, tf: tf}]
	d.mu.Unlock()
	if !ok {
		http.Error(w, "not subscribed", http.StatusNotFound)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	id, ch := sub.addListener()
	defer sub.removeListener(id)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if backfilled, bars, errStr := sub.snapshotBackfill(); backfilled {
		if data, err := json.Marshal(backfillEvent{Bars: bars, Err: errStr}); err == nil {
			fmt.Fprintf(w, "event: backfill_complete\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-ch:
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.name, ev.data)
			flusher.Flush()
		}
	}
}

func (d *dashboard) handleBars(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol")))
	tf := r.URL.Query().Get("timeframe")

	bars, err := d.mkdata.ListBars(r.Context(), symbol, tf)
	if err != nil {
		d.log.Error("list bars", slog.String("symbol", symbol), slog.String("tf", tf), slog.Any("error", err))
		http.Error(w, "failed to load bars", http.StatusInternalServerError)
		return
	}

	out := make([]chartBar, 0, len(bars))
	for _, b := range bars {
		out = append(out, toChartBar(b))
	}
	if err := httputil.WriteJSON(w, out, http.StatusOK); err != nil {
		d.log.Error("write bars response", slog.Any("error", err))
	}
}
