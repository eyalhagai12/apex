package providers

import (
	"apex/internal/domain"
	"context"
)

type barHandler func(domain.Bar) error

type Provider interface {
	Name() string
	Subscribe(context.Context, string, string, barHandler) error
}
