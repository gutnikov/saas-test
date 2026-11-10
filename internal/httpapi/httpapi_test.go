package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gutnikov/saas-test/internal/store"
)

func TestHealthAnswersWithTheVersion(t *testing.T) {
	srv, _ := newServer(t)
	res := get(t, srv.URL+"/healthz")
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK || body["status"] != "ok" || body["version"] != "test" {
		t.Fatalf("GET /healthz: %d %v, want 200 with status ok and version test", res.StatusCode, body)
	}
}

func TestUnknownAPIPathIsAJSON404(t *testing.T) {
	srv, _ := newServer(t)
	res := get(t, srv.URL+"/api/nothing-here")
	if res.StatusCode != http.StatusNotFound || res.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("GET /api/nothing-here: %d %s, want a JSON 404", res.StatusCode, res.Header.Get("Content-Type"))
	}
}

func TestFrontendServesAssetsAndFallsBackToIndex(t *testing.T) {
	srv, _ := newServer(t)
	for path, want := range map[string]string{
		"/":              "<title>app</title>",
		"/teams/42":      "<title>app</title>",
		"/assets/app.js": "console.log(1)",
	} {
		res := get(t, srv.URL+path)
		body, _ := io.ReadAll(res.Body)
		if res.StatusCode != http.StatusOK || !strings.Contains(string(body), want) {
			t.Errorf("GET %s: %d %q, want 200 with %q", path, res.StatusCode, body, want)
		}
	}
}

func TestPanicBecomesA500AndAnErrorLine(t *testing.T) {
	var log bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&log, nil))
	h := logRequests(logger, recoverPanics(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/boom", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
	if !strings.Contains(log.String(), `"level":"ERROR","msg":"panic","error":"boom"`) {
		t.Fatalf("no error line for the panic in the log:\n%s", log.String())
	}
}

func newServer(t *testing.T) (*httptest.Server, *bytes.Buffer) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	var log bytes.Buffer
	static := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><title>app</title>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}
	srv := httptest.NewServer(New(Config{
		Version: "test",
		Logger:  slog.New(slog.NewJSONHandler(&log, nil)),
		DB:      db,
		Static:  static,
	}))
	t.Cleanup(srv.Close)
	return srv, &log
}

func get(t *testing.T, url string) *http.Response {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	return res
}
