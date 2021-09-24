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
		[]string{"device_id", "device_name"},
	)
	humidity = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "humidity",
			Help:      "Humidity of the meters.",
		},
		[]string{"device_id", "device_name"},
	)
)

func init() {
	prometheus.MustRegister(temperature, humidity)
}

const (
	scrapeInterval = 60 * time.Second
)

type switchBotCollector struct {
	client *switchbot.Client
	meters []switchbot.Device
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
			c.meters = append(c.meters, d)
			log.Printf("adding meter with device id: %s, name: %s\n", d.ID, d.Name)
		}
	}

	return nil
}

// updateLoop periodically updates the metrics.
func (c *switchBotCollector) updateLoop() {
	ticker := time.NewTicker(scrapeInterval)

	log.Println("start collecting...")
	c.update()
	for {
		select {
		case <-ticker.C:
			c.update()
		}
	}
}

func (c *switchBotCollector) update() {
	for _, meter := range c.meters {
		status, err := c.client.Device().Status(context.Background(), meter.ID)
		if err != nil {
			log.Printf("failed to update status for device id: %s, name: %s\n", meter.ID, meter.Name)
			continue
		}
		temperature.WithLabelValues(meter.ID, meter.Name).Set(status.Temperature)
		humidity.WithLabelValues(meter.ID, meter.Name).Set(float64(status.Humidity))
	}
}
