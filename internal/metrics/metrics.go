package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	BarsBackfilled = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apex_bars_backfilled_total",
			Help: "Total number of historical bars fetched and stored via backfill, labeled by symbol.",
		},
		[]string{"symbol"},
	)

	BarsStreamed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apex_bars_streamed_total",
			Help: "Total number of real-time bars received and stored via the subscribe stream, labeled by symbol.",
		},
		[]string{"symbol"},
	)
)
