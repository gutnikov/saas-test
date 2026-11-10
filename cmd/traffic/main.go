// Command traffic sends synthetic user requests to a running server, so that a release has traffic to be judged on.
package main

import (
	"flag"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// scenarios are the requests a synthetic user makes. A feature that ships adds its own.
var scenarios = []string{
	"/",
	"/api/version",
	"/healthz",
}

func main() {
	base := flag.String("base", "http://127.0.0.1:18080", "the server to send traffic to")
	every := flag.Duration("every", 2*time.Second, "the average pause between two requests")
	flag.Parse()

	client := &http.Client{Timeout: 5 * time.Second}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	for {
		path := scenarios[rand.IntN(len(scenarios))]
		res, err := client.Get(*base + path)
		if err != nil {
			log.Printf("GET %s: %v", path, err)
		} else {
			_ = res.Body.Close()
		}
		select {
		case <-stop:
			return
		case <-time.After(time.Duration(rand.Int64N(int64(2 * *every)))):
		}
	}
}
