package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/rss-aggregator/internal/api"
	"github.com/yourusername/rss-aggregator/internal/fetcher"
	"github.com/yourusername/rss-aggregator/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// --- Configuration ---
	port := envOrDefault("PORT", "8080")
	fetchInterval := 5 * time.Minute

	// --- Dependencies ---
	st := store.New()
	fetch := fetcher.New(st, fetchInterval, logger)
	srv := api.New(st, logger)

	// --- Seed some default feeds (optional, remove for production) ---
	seedFeeds(st)

	// --- Background fetcher ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go fetch.Start(ctx)

	// --- HTTP server ---
	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      srv,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server started", "port", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")

	cancel() // stop the fetcher

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}

	logger.Info("server stopped")
}

func seedFeeds(s *store.Store) {
	defaults := []struct{ name, url string }{
		{"Go Blog", "https://go.dev/blog/feed.atom"},
		{"Hacker News", "https://hnrss.org/frontpage"},
		{"Lobsters", "https://lobste.rs/rss"},
	}
	for _, d := range defaults {
		s.AddFeed(d.name, d.url)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func init() {
	fmt.Println(`
  ____  ____ ____     _                                _
 |  _ \/ ___/ ___|   / \   __ _  __ _ _ __ ___  __ _  | |_ ___  _ __
 | |_) \___ \___ \  / _ \ / _' |/ _' | '__/ _ \/ _' | | __/ _ \| '__|
 |  _ < ___) |__) |/ ___ \ (_| | (_| | | |  __/ (_| | | || (_) | |
 |_| \_\____/____/_/_/  \_\__, |\__, |_|  \___|\__, |  \__\___/|_|
                           |___/ |___/          |___/
	`)
}
