package providers

import (
	"apex/internal/domain"
	"context"
	"time"
)

type barHandler func(domain.Bar) error

type Provider interface {
	Name() string
	Subscribe(context.Context, string, barHandler) error
	Unsubscribe(context.Context, string) error
	GetBackfillBars(context.Context, string, time.Time, time.Time) ([]domain.Bar, error)
}
