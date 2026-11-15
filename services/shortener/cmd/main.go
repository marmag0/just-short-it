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

	"github.com/marmag0/just-short-it/services/shortener/link"
)

func main() {
	port := os.Getenv("SHORTENER_PORT")
	if port == "" {
		port = envFileValue(".env", "SHORTENER_PORT")
	}
	if port == "" {
		slog.Error("SHORTENER_PORT is not set in the environment or .env")
		os.Exit(1)
	}
	base := "cyberwav.ee"
	addr := fmt.Sprintf(":%s", port)

	srv := &link.Server{
		Base:  base,
		Links: map[string]string{"test": "https://akamai.com"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{link}", srv.HandleLink)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server is starting", "addr", addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped cleanly")
}

// envFileValue returns the value of key from a KEY=value file such as the root .env.
func envFileValue(path, key string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for line := range strings.Lines(string(data)) {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), key+"="); ok {
			return value
		}
	}
	return ""
}
