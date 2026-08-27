package main

import (
	dbconnection "bank/db_connection"
	"bank/intern/metrics"
	"bank/router"
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	pool, err := dbconnection.CreateConnection(ctx)
	if err != nil {
		log.Fatal("DB:", err)
	}
	defer pool.Close()
	log.Println("DB connected")

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			stat := pool.Stat()
			metrics.DbPoolConnections.WithLabelValues("active").Set(float64(stat.AcquiredConns()))
			metrics.DbPoolConnections.WithLabelValues("idle").Set(float64(stat.IdleConns()))
		}
	}()

	r := router.NewRouter(router.Config{
		Pool:       pool,
		JWTSecret:  "secret",
		ExpireTime: 24 * time.Hour,
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		log.Println("🚀 Server starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server error:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down gracefully...")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatal("Forced shutdown:", err)
	}

	log.Println(" Server stopped gracefully")
}
