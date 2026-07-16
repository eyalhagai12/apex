package domain

import (
	"time"
)

type Bar struct {
	Time   time.Time `json:"time"`
	Symbol string    `json:"symbol"`
	High   float64   `json:"high"`
	Open   float64   `json:"open"`
	Close  float64   `json:"close"`
	Low    float64   `json:"low"`
	Volume uint64    `json:"volume"`
}

func NewBar(time time.Time, symbol string, h, o, l, c float64, v uint64) Bar {
	return Bar{
		Time:   time,
		Symbol: symbol,
		High:   h,
		Open:   o,
		Low:    l,
		Close:  c,
		Volume: v,
	}
}

type Subscription struct {
	Symbol string
}
