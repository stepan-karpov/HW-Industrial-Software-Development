package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"hw/app/handlers"
)

func main() {
	paths := ResolvePaths()
	logger := log.New(os.Stdout, "", log.LstdFlags)

	cfg, err := LoadConfig(paths.ConfigPath)
	if err != nil {
		logger.Fatalf("failed to load config: %v", err)
	}

	if err := EnsureLogFileExists(paths.LogPath); err != nil {
		logger.Fatalf("failed to prepare log file: %v", err)
	}

	var logMu sync.Mutex
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handlers.Root(cfg.Greeting))
	mux.HandleFunc("GET /status", handlers.Status())
	mux.HandleFunc("POST /log", handlers.LogPost(paths.LogPath, &logMu, logger))
	mux.HandleFunc("GET /logs", handlers.LogsGet(paths.LogPath))

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Printf("starting server on %s (log_level=%s)", srv.Addr, cfg.LogLevel)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("server error: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	logger.Printf("shutdown complete")
}
