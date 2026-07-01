// Package logging builds the app's slog.Logger. Logs are written as JSON to
// stdout and to a file under logDir; Promtail (see docker-compose.yml) tails
// that file and ships the entries to Loki, so the app itself never talks to
// Loki directly.
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// New returns a JSON slog.Logger writing to stdout and logDir/apex.log,
// along with a close func that must be called before the process exits.
// If the log file can't be opened, it falls back to stdout only.
func New(logDir string) (*slog.Logger, func() error) {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil)), func() error { return nil }
	}

	f, err := os.OpenFile(filepath.Join(logDir, "apex.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil)), func() error { return nil }
	}

	logger := slog.New(slog.NewJSONHandler(io.MultiWriter(os.Stdout, f), nil))
	return logger, f.Close
}
