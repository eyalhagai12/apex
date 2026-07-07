package web

import "apex/internal/domain"

// chartBar is the wire shape the Lightweight Charts library expects: a
// UNIX-seconds integer time, not the RFC3339 string domain.Bar.Time
// serializes to via its json tag.
type chartBar struct {
	Time  int64   `json:"time"`
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}

func toChartBar(b domain.Bar) chartBar {
	return chartBar{
		Time:  b.Time.Unix(),
		Open:  b.Open,
		High:  b.High,
		Low:   b.Low,
		Close: b.Close,
	}
}
