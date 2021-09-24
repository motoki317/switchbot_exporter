package main

import (
	"context"
	"log"
	"time"

	"github.com/nasa9084/go-switchbot"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	promNamespace = "switchbot"
)

var (
	temperature = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "temperature",
			Help:      "Temperature of the meters.",
		},
		[]string{"device_id"},
	)
	humidity = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "humidity",
			Help:      "Humidity of the meters.",
		},
		[]string{"device_id"},
	)
)

func init() {
	prometheus.MustRegister(temperature, humidity)
}

const (
	scrapeInterval = 60 * time.Second
)

type switchBotCollector struct {
	client   *switchbot.Client
	meterIDs []string
}

func newSwitchBotCollector(token string) *switchBotCollector {
	return &switchBotCollector{
		client: switchbot.New(token),
	}
}

// init initializes device list.
func (c *switchBotCollector) init() error {
	devices, _, err := c.client.Device().List(context.Background())
	if err != nil {
		return err
	}

	for _, d := range devices {
		switch d.Type {
		case switchbot.Meter:
			c.meterIDs = append(c.meterIDs, d.ID)
			log.Printf("adding meter with device id: %s\n", d.ID)
		}
	}

	return nil
}

// updateLoop periodically updates the metrics.
func (c *switchBotCollector) updateLoop() {
	ticker := time.NewTicker(scrapeInterval)

	log.Println("start collecting...")
	select {
	case <-ticker.C:
		for _, meterID := range c.meterIDs {
			status, err := c.client.Device().Status(context.Background(), meterID)
			if err != nil {
				log.Printf("failed to update status for device: %s\n", meterID)
				continue
			}
			temperature.WithLabelValues(status.ID).Set(status.Temperature)
			humidity.WithLabelValues(status.ID).Set(float64(status.Humidity))
		}
	}
}
