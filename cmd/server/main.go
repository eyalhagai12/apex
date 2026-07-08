package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"apex/internal/httputil"
	"apex/internal/logging"
	"apex/internal/web"
	"apex/marketdata"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("loading .env: %v", err)
	}

	baseLogger, cleanup, err := logging.New()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer cleanup()

	// logger is used everywhere except the request-logging middleware, so
	// every business-level log line is automatically labeled type=domain,
	// distinguishing it from type=request lines emitted per HTTP call.
	logger := baseLogger.With(slog.String("type", "domain"))
	requestLogger := baseLogger.With(slog.String("type", "request"))

	apcaKeyID := os.Getenv("APCA_API_KEY_ID")
	apcaSecretKey := os.Getenv("APCA_API_SECRET_KEY")
	if apcaKeyID == "" || apcaSecretKey == "" {
		logger.Error("APCA_API_KEY_ID and APCA_API_SECRET_KEY must be set (see .env.example)")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dsn := fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_DB"))

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		logger.Error("open db", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		logger.Error("ping db", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("database connected")

	mkdata, err := marketdata.New(ctx, db, logger, apcaKeyID, apcaSecretKey)
	if err != nil {
		logger.Error("init marketdata module", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("marketdata module ready")

	mux := http.NewServeMux()
	web.Mount(mux, logger, mkdata, ctx)
	mux.Handle("/metrics", promhttp.Handler())

	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	srv := &http.Server{Addr: addr, Handler: httputil.LogRoutes(requestLogger, mux)}

	go func() {
		logger.Info("server starting", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.Any("error", err))
			cancel()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", slog.Any("error", err))
	}
}
