// Package httpapi is the service's HTTP layer: the API routes, the frontend and the request log.
package httpapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

// Config is what the handler needs from the rest of the service.
type Config struct {
	Version string
	Logger  *slog.Logger
	DB      *sql.DB
	Static  fs.FS // the built frontend: index.html and its assets
}

// New returns the service's root handler.
func New(cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health(cfg))
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"version": cfg.Version})
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	})
	mux.Handle("/", frontend(cfg.Static)) // "GET /" would conflict with "/api/": neither is more specific
	return logRequests(cfg.Logger, recoverPanics(cfg.Logger, mux))
}

// health answers 200 with the version while the database answers, and 503 when it does not.
func health(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := cfg.DB.PingContext(r.Context()); err != nil {
			cfg.Logger.Error("health check failed", "error", err.Error())
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable", "version": cfg.Version})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": cfg.Version})
	}
}

// frontend serves the built files, and index.html for every other path, so the app's own routes survive a reload.
func frontend(static fs.FS) http.Handler {
	files := http.FileServerFS(static)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		if info, err := fs.Stat(static, name); name != "" && err == nil && !info.IsDir() {
			files.ServeHTTP(w, r)
			return
		}
		http.ServeFileFS(w, r, static, "index.html")
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// logRequests writes one line per request: the method, the route, the status and the time it took. A 5xx is an
// error line, so a release's errors can be counted from the log.
func logRequests(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		level := slog.LevelInfo
		if rec.status >= http.StatusInternalServerError {
			level = slog.LevelError
		}
		logger.Log(r.Context(), level, "http request",
			"method", r.Method, "route", r.Pattern, "path", r.URL.Path,
			"status", rec.status, "duration_ms", time.Since(start).Milliseconds())
	})
}

// recoverPanics turns a panic in a handler into a 500 and an error line with the stack.
func recoverPanics(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler {
				panic(v)
			}
			logger.Error("panic", "error", fmt.Sprint(v), "path", r.URL.Path, "stack", string(debug.Stack()))
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}()
		next.ServeHTTP(w, r)
	})
}
