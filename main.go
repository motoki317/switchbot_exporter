package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{},
	))
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}
