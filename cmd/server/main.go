package main

import (
	"log/slog"

	"apex/logging"
)

func main() {
	logger, closeLogger := logging.New("logs")
	defer closeLogger()
	slog.SetDefault(logger)

	logger.Info("apex starting")
}
