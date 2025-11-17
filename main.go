package main

import (
	"flag"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/VictoriaMetrics/metrics"
)

var (
	port   = flag.Int("port", 9257, "Port to bind")
	token  = flag.String("token", "", "The SwitchBot open token")
	secret = flag.String("secret", "", "The SwitchBot secret key")

	scrapeIntervalSeconds = flag.Int("scrape-interval-seconds", 5*60, "Scrape interval in seconds")
)

func main() {
	flag.Parse()
	if *token == "" {
		panic("open token is required")
	}
	if *secret == "" {
		panic("secret token is required")
	}

	c := newSwitchBotCollector(*token, *secret)
	if err := c.init(); err != nil {
		panic(err)
	}

	go c.updateLoop()

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.WritePrometheus(w, false)
	})

	slog.Info("start listening...", "version", GetFormattedVersion(), "port", *port)
	err := http.ListenAndServe(":"+strconv.Itoa(*port), nil)
	if err != nil {
		panic(err)
	}
}
