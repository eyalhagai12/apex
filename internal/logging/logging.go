package logging

import (
	"io"
	"log/slog"
	"os"
)

func New() (*slog.Logger, func(), error) {
	if err := os.MkdirAll("logs", 0o755); err != nil {
		return nil, nil, err
	}

	f, err := os.OpenFile("logs/apex.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}

	logger := slog.New(slog.NewJSONHandler(io.MultiWriter(os.Stdout, f), nil))
	slog.SetDefault(logger)

	return logger, func() { f.Close() }, nil
}
