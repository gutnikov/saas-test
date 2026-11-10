// Command traffic sends synthetic user requests to a running server, so that a release has traffic to be judged on.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// A scenario is one thing a synthetic user does. A feature that ships adds its own, so that its production signal
// has traffic to appear in.
type scenario func(c *http.Client, base string) (*http.Response, error)

func get(path string) scenario {
	return func(c *http.Client, base string) (*http.Response, error) { return c.Get(base + path) }
}

// createTeam is TEAMS-1#1: a visitor creates a team under a fresh name.
func createTeam(c *http.Client, base string) (*http.Response, error) {
	body := fmt.Sprintf(`{"name": "team %d"}`, rand.Int64())
	return c.Post(base+"/api/teams", "application/json", strings.NewReader(body))
}

// scenarios, as often as they come: the list of teams (TEAMS-1#2) and new teams (TEAMS-1#1) more than the rest.
var scenarios = []scenario{
	get("/"), get("/api/version"), get("/healthz"),
	get("/api/teams"), get("/api/teams"),
	createTeam, createTeam,
}

func main() {
	base := flag.String("base", "http://127.0.0.1:18080", "the server to send traffic to")
	every := flag.Duration("every", 2*time.Second, "the average pause between two requests")
	flag.Parse()

	client := &http.Client{Timeout: 5 * time.Second}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	for {
		res, err := scenarios[rand.IntN(len(scenarios))](client, *base)
		if err != nil {
			log.Print(err)
		} else {
			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()
		}
		select {
		case <-stop:
			return
		case <-time.After(time.Duration(rand.Int64N(int64(2 * *every)))):
		}
	}
}
