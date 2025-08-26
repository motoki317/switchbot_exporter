package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"

	"github.com/VictoriaMetrics/metrics"
)

var (
	port   = flag.Int("port", 9257, "Port to bind.")
	token  = flag.String("token", "", "The SwitchBot open token.")
	secret = flag.String("secret", "", "The SwitchBot secret key.")
)

func main() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Lmicroseconds)

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
		metrics.WritePrometheus(w, true)
	})
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}
