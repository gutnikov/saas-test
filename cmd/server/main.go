// Command server runs saas-test: the HTTP API and the built frontend, in one binary.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gutnikov/saas-test/internal/httpapi"
	"github.com/gutnikov/saas-test/internal/store"
	"github.com/gutnikov/saas-test/web"
)

// version is the build's version, set by the linker: -ldflags "-X main.version=<git sha>".
var version = "dev"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("version", version)
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err.Error())
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	addr := env("ADDR", "127.0.0.1:8080")
	dbPath := env("DB_PATH", "saas-test.db")

	db, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.New(httpapi.Config{Version: version, Logger: logger, DB: db, Static: web.Dist()}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	logger.Info("server started", "addr", addr, "db", dbPath)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	logger.Info("server stopping")
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdown)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
