package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"apex/marketdata"

	"github.com/joho/godotenv"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("loading .env: %v", err)
	}

	apcaKeyID := os.Getenv("APCA_API_KEY_ID")
	apcaSecretKey := os.Getenv("APCA_API_SECRET_KEY")
	if apcaKeyID == "" || apcaSecretKey == "" {
		log.Fatal("APCA_API_KEY_ID and APCA_API_SECRET_KEY must be set (see .env.example)")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dsn := fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_DB"))

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	module, err := marketdata.New(ctx, db, apcaKeyID, apcaSecretKey)
	if err != nil {
		log.Fatalf("init marketdata module: %v", err)
	}

	const symbol, tf = "AAPL", "1Min"
	if err := module.Subscribe(ctx, symbol, tf); err != nil {
		log.Fatalf("subscribe to %s: %v", symbol, err)
	}

	log.Printf("subscribed to %s %s bars, writing to the bars table — ctrl+c to stop", symbol, tf)
	<-ctx.Done()
	log.Println("shutting down")
}
