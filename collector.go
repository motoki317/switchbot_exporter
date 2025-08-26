package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/nasa9084/go-switchbot/v3"
)

const (
	scrapeInterval = 60 * time.Second
)

type switchBotCollector struct {
	client *switchbot.Client
	meters []switchbot.Device

	temperature map[string]*metrics.Gauge
	humidity    map[string]*metrics.Gauge
}

func newSwitchBotCollector(token, secret string) *switchBotCollector {
	return &switchBotCollector{
		client: switchbot.New(token, secret),

		temperature: make(map[string]*metrics.Gauge),
		humidity:    make(map[string]*metrics.Gauge),
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

			c.temperature[d.ID] = metrics.NewGauge(fmt.Sprintf(`switchbot_temperature{device_id="%s", device_name="%s"}`, d.ID, d.Name), nil)
			c.humidity[d.ID] = metrics.NewGauge(fmt.Sprintf(`switchbot_humidity{device_id="%s", device_name="%s"}`, d.ID, d.Name), nil)
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
			log.Printf("failed to update status (device id: %s, name: %s): %v\n", meter.ID, meter.Name, err)
			continue
		}
		if status.Temperature == 0 && status.Humidity == 0 {
			// API sometimes returns zero values for some reason
			log.Printf("[warn] zero values for device id: %s, name: %s, skipping update\n", meter.ID, meter.Name)
			continue
		}

		c.temperature[meter.ID].Set(status.Temperature)
		c.humidity[meter.ID].Set(float64(status.Humidity))
	}
}
