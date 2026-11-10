package acceptance

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/gutnikov/saas-test/internal/httpapi"
	"github.com/gutnikov/saas-test/internal/store"
)

// The seam of README.md: how a test builds a state, sends the command CreateTeam, reads the read model Teams
// and finds the event TeamCreated in the app's log.

// runningApp is a fresh app behind a test HTTP server, with its JSON log kept in memory.
type runningApp struct {
	t      *testing.T
	url    string
	client *http.Client
	log    *logBuffer
}

// noTeams builds the state noTeams: a fresh app.
func noTeams(t *testing.T) *runningApp {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	log := &logBuffer{}
	srv := httptest.NewServer(httpapi.New(httpapi.Config{
		Version: "acceptance",
		Logger:  slog.New(slog.NewJSONHandler(log, nil)),
		DB:      db,
		Static:  fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>app</title>")}},
	}))
	t.Cleanup(srv.Close)
	return &runningApp{t: t, url: srv.URL, client: srv.Client(), log: log}
}

// teamAcme builds the state teamAcme: a fresh app in which POST /api/teams created a team named Acme.
func teamAcme(t *testing.T) *runningApp {
	t.Helper()
	app := noTeams(t)
	if res := app.createTeam("Acme"); res.status != http.StatusCreated {
		t.Fatalf("state teamAcme: %s Acme: status %d, want 201; body %s", res.request, res.status, res.body)
	}
	return app
}

// createTeam sends the command CreateTeam: POST /api/teams with the JSON body {"name": name}.
func (app *runningApp) createTeam(name string) response {
	app.t.Helper()
	body, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		app.t.Fatal(err)
	}
	return app.send(http.MethodPost, "/api/teams", body)
}

// send sends one request and reads the whole response.
func (app *runningApp) send(method, path string, body []byte) response {
	app.t.Helper()
	request := method + " " + path
	logFrom := app.log.size()
	req, err := http.NewRequest(method, app.url+path, bytes.NewReader(body))
	if err != nil {
		app.t.Fatalf("%s: %v", request, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	r, err := app.client.Do(req)
	if err != nil {
		app.t.Fatalf("%s: %v", request, err)
	}
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		app.t.Fatalf("%s: reading the body: %v", request, err)
	}
	return response{request: request, status: r.StatusCode, body: bytes.TrimSpace(data), log: app.log, logFrom: logFrom}
}

// response is what the app answered to one request.
type response struct {
	request string // the method and the path, such as "POST /api/teams"
	status  int
	body    []byte // trimmed, so that a failure message shows it on one line
	log     *logBuffer
	logFrom int // the size of the log when the request was sent
}

// wantCreated checks that the command created a team: status 201 and {"id": "<id>", "name": name}.
func wantCreated(t *testing.T, res response, name string) {
	t.Helper()
	if res.status != http.StatusCreated {
		t.Fatalf("%s: status %d, want 201; body %s", res.request, res.status, res.body)
	}
	var team struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(res.body, &team); err != nil {
		t.Fatalf("%s: the body %s is not a team: %v", res.request, res.body, err)
	}
	if team.Name != name || team.ID == "" {
		t.Fatalf("%s: the body %s, want an id and the name %q", res.request, res.body, name)
	}
}

// wantRefused checks that the command was refused: the status and {"error": errName, "message": message}.
func wantRefused(t *testing.T, res response, status int, errName, message string) {
	t.Helper()
	if res.status != status {
		t.Fatalf("%s: status %d, want %d; body %s", res.request, res.status, status, res.body)
	}
	var refusal struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(res.body, &refusal); err != nil {
		t.Fatalf("%s: the body %s is not an error: %v", res.request, res.body, err)
	}
	if refusal.Error != errName || refusal.Message != message {
		t.Fatalf("%s: error %q and message %q, want %q and %q",
			res.request, refusal.Error, refusal.Message, errName, message)
	}
}

// wantTeamNames checks the read model Teams: GET /api/teams answers 200 with these names, in this order.
func wantTeamNames(t *testing.T, app *runningApp, names ...string) {
	t.Helper()
	res := app.send(http.MethodGet, "/api/teams", nil)
	if res.status != http.StatusOK {
		t.Fatalf("%s: status %d, want 200; body %s", res.request, res.status, res.body)
	}
	var teams []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(res.body, &teams); err != nil {
		t.Fatalf("%s: the body %s is not a list of teams: %v", res.request, res.body, err)
	}
	got := make([]string, len(teams))
	for i, team := range teams {
		got[i] = team.Name
	}
	if !slices.Equal(got, names) {
		t.Fatalf("%s: the names %q, want %q", res.request, got, names)
	}
}

// wantTeamCreated checks that the request logged one domain event, TeamCreated, with a teamId and the name.
func wantTeamCreated(t *testing.T, res response, name string) {
	t.Helper()
	events := res.domainEvents(t)
	if len(events) != 1 || events[0]["event"] != "TeamCreated" {
		t.Fatalf("%s logged the domain events %v, want one TeamCreated", res.request, events)
	}
	if e := events[0]; e["name"] != name || e["teamId"] == nil || e["teamId"] == "" {
		t.Fatalf(`%s logged %v, want "name" %q and a "teamId"`, res.request, e, name)
	}
}

// wantNoEvents checks that the request logged no domain event.
func wantNoEvents(t *testing.T, res response) {
	t.Helper()
	if events := res.domainEvents(t); len(events) > 0 {
		t.Fatalf("%s logged the domain events %v, want none", res.request, events)
	}
}

// domainEvents returns the log lines with "msg":"domain event" written since the request was sent.
func (res response) domainEvents(t *testing.T) []map[string]any {
	t.Helper()
	var events []map[string]any
	for line := range strings.Lines(res.log.since(res.logFrom)) {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("the log line %q is not JSON: %v", line, err)
		}
		if entry["msg"] == "domain event" {
			events = append(events, entry)
		}
	}
	return events
}

// logBuffer is the app's log: the server writes it from its own goroutines while the test reads it.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *logBuffer) size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Len()
}

// since returns what was logged after the first n bytes.
func (b *logBuffer) since(n int) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()[n:]
}
