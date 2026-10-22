// Command api serves the Bandung Sport catalog over HTTP.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jakskal/bandung-sport-be/data"
	"github.com/jakskal/bandung-sport-be/internal/catalog"
	"github.com/jakskal/bandung-sport-be/internal/httpapi"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(logger); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cat, err := catalog.Load(data.FS)
	if err != nil {
		return fmt.Errorf("load catalog: %w", err)
	}
	for _, warning := range cat.Warnings() {
		logger.Warn("catalog", "warning", warning)
	}
	if errs := cat.Validate(); len(errs) > 0 {
		for _, err := range errs {
			logger.Error("catalog", "error", err)
		}
		return fmt.Errorf("catalog has %d validation error(s)", len(errs))
	}

	origins := splitList(os.Getenv("ALLOWED_ORIGINS"))
	if len(origins) == 0 {
		logger.Warn("ALLOWED_ORIGINS is empty: browsers from the web app will be blocked by CORS")
	}

	server := &http.Server{
		Addr:              ":" + env("PORT", "8080"),
		Handler:           httpapi.New(cat, origins, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", server.Addr, "origins", origins)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func splitList(value string) []string {
	var items []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			items = append(items, part)
		}
	}
	return items
}
