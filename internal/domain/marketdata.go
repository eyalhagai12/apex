package domain

import (
	"time"
)

type Bar struct {
	Time      time.Time `json:"time"`
	Symbol    string    `json:"symbol"`
	Timeframe string    `json:"timeframe"`
	High      float64   `json:"high"`
	Open      float64   `json:"open"`
	Close     float64   `json:"close"`
	Low       float64   `json:"low"`
	Volume    uint64    `json:"volume"`
}

func NewBar(time time.Time, symbol, tf string, h, o, l, c float64, v uint64) Bar {
	return Bar{
		Time:      time,
		Symbol:    symbol,
		Timeframe: tf,
		High:      h,
		Open:      o,
		Low:       l,
		Close:     c,
		Volume:    v,
	}
}

type Subscription struct {
	Symbol    string
	Timeframe string
}

type Stream struct {
	tf     string
	symbol string

	bars chan Bar
}

func (s *Stream) Listen() <-chan Bar {
	return s.bars
}

func (s *Stream) Close() {
	close(s.bars)
}

func NewStream(symbol, tf string) *Stream {
	return &Stream{
		symbol: symbol,
		tf:     tf,

		bars: make(chan Bar),
	}
}
