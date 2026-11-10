// Package httpapi is the service's HTTP layer: the API routes, the frontend and the request log.
package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gutnikov/saas-test/internal/store"
	"github.com/gutnikov/saas-test/internal/teams"
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
	mux.HandleFunc("POST /api/teams", createTeam(cfg))
	mux.HandleFunc("GET /api/teams", listTeams(cfg))
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

type teamJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// createTeam is the command CreateTeam (spec TEAMS-1, slice S01). It answers 201 with the team, or the refusal as
// {"error", "message"}: 400 for a name that is empty or too long, 409 for a taken one. A created team is logged as
// the domain event TeamCreated before the answer: that line is the production signal of TEAMS-1#1.
func createTeam(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "BadRequest",
				"message": "the body must be a JSON object with a name"})
			return
		}
		name, err := teams.Name(body.Name)
		if err == nil {
			var t teams.Team
			if t, err = store.CreateTeam(r.Context(), cfg.DB, name); err == nil {
				cfg.Logger.Info("domain event", "event", "TeamCreated", "teamId", t.ID, "name", t.Name)
				writeJSON(w, http.StatusCreated, teamJSON{ID: t.ID, Name: t.Name})
				return
			}
		}
		var refused *teams.Error
		if errors.As(err, &refused) {
			status := http.StatusBadRequest
			if refused == teams.ErrNameTaken {
				status = http.StatusConflict
			}
			writeJSON(w, status, map[string]string{"error": refused.Code, "message": refused.Message})
			return
		}
		cfg.Logger.Error("create team", "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

// listTeams is the read model Teams (slice V01): every team, newest first.
func listTeams(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := store.ListTeams(r.Context(), cfg.DB)
		if err != nil {
			cfg.Logger.Error("list teams", "error", err.Error())
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		out := make([]teamJSON, 0, len(list))
		for _, t := range list {
			out = append(out, teamJSON{ID: t.ID, Name: t.Name})
		}
		writeJSON(w, http.StatusOK, out)
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
