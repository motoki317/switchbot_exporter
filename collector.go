package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/nasa9084/go-switchbot/v3"
)

type switchBotCollector struct {
	client *switchbot.Client

	devicesMeter     []switchbot.Device
	devicesMeterPro  []switchbot.Device
	meterTemperature map[string]*metrics.Gauge
	meterHumidity    map[string]*metrics.Gauge
	meterCO2         map[string]*metrics.Gauge

	devicesPlugMiniJP []switchbot.Device
	plugVoltage       map[string]*metrics.Gauge
	plugCurrent       map[string]*metrics.Gauge
	plugPower         map[string]*metrics.Gauge
	plugMinutesOfDay  map[string]*metrics.Gauge
}

func newSwitchBotCollector(token, secret string) *switchBotCollector {
	return &switchBotCollector{
		client: switchbot.New(token, secret),

		meterTemperature: make(map[string]*metrics.Gauge),
		meterHumidity:    make(map[string]*metrics.Gauge),
		meterCO2:         make(map[string]*metrics.Gauge),

		plugVoltage:      make(map[string]*metrics.Gauge),
		plugCurrent:      make(map[string]*metrics.Gauge),
		plugPower:        make(map[string]*metrics.Gauge),
		plugMinutesOfDay: make(map[string]*metrics.Gauge),
	}
}

// init initializes device list.
func (c *switchBotCollector) init() error {
	devices, _, err := c.client.Device().List(context.Background())
	if err != nil {
		return err
	}

	for _, d := range devices {
		slog.Info("device found", "id", d.ID, "name", d.Name, "type", d.Type)

		switch d.Type {
		case switchbot.Meter:
			c.devicesMeter = append(c.devicesMeter, d)
			c.meterTemperature[d.ID] = metrics.NewGauge(fmt.Sprintf(`switchbot_temperature{device_id="%s", device_name="%s"}`, d.ID, d.Name), nil)
			c.meterHumidity[d.ID] = metrics.NewGauge(fmt.Sprintf(`switchbot_humidity{device_id="%s", device_name="%s"}`, d.ID, d.Name), nil)

		case switchbot.MeterPro, "MeterPro(CO2)":
			c.devicesMeterPro = append(c.devicesMeterPro, d)
			c.meterTemperature[d.ID] = metrics.NewGauge(fmt.Sprintf(`switchbot_temperature{device_id="%s", device_name="%s"}`, d.ID, d.Name), nil)
			c.meterHumidity[d.ID] = metrics.NewGauge(fmt.Sprintf(`switchbot_humidity{device_id="%s", device_name="%s"}`, d.ID, d.Name), nil)
			c.meterCO2[d.ID] = metrics.NewGauge(fmt.Sprintf(`switchbot_co2{device_id="%s", device_name="%s"}`, d.ID, d.Name), nil)

		case switchbot.PlugMiniJP:
			c.devicesPlugMiniJP = append(c.devicesPlugMiniJP, d)
			c.plugVoltage[d.ID] = metrics.NewGauge(fmt.Sprintf(`switchbot_plug_voltage{device_id="%s", device_name="%s"}`, d.ID, d.Name), nil)
			c.plugCurrent[d.ID] = metrics.NewGauge(fmt.Sprintf(`switchbot_plug_current{device_id="%s", device_name="%s"}`, d.ID, d.Name), nil)
			c.plugPower[d.ID] = metrics.NewGauge(fmt.Sprintf(`switchbot_plug_power{device_id="%s", device_name="%s"}`, d.ID, d.Name), nil)
			c.plugMinutesOfDay[d.ID] = metrics.NewGauge(fmt.Sprintf(`switchbot_plug_minutes_of_day{device_id="%s", device_name="%s"}`, d.ID, d.Name), nil)
		}
	}

	return nil
}

// updateLoop periodically updates the metrics.
func (c *switchBotCollector) updateLoop() {
	ticker := time.NewTicker(time.Duration(*scrapeIntervalSeconds) * time.Second)

	// First update
	c.update()

	for range ticker.C {
		c.update()
	}
}

func (c *switchBotCollector) update() {
	for _, meter := range c.devicesMeter {
		status, err := c.client.Device().Status(context.Background(), meter.ID)
		if err != nil {
			slog.Error("failed to update status", "device_id", meter.ID, "device_name", meter.Name, "error", err)
			continue
		}
		if status.Temperature == 0 && status.Humidity == 0 {
			// API sometimes returns zero values for some reason
			slog.Warn("zero values for device id", "device_id", meter.ID, "device_name", meter.Name)
			continue
		}
		c.meterTemperature[meter.ID].Set(status.Temperature)
		c.meterHumidity[meter.ID].Set(float64(status.Humidity))
	}

	for _, meter := range c.devicesMeterPro {
		status, err := c.client.Device().Status(context.Background(), meter.ID)
		if err != nil {
			slog.Error("failed to update status", "device_id", meter.ID, "device_name", meter.Name, "error", err)
			continue
		}
		if status.Temperature == 0 && status.Humidity == 0 {
			// API sometimes returns zero values for some reason
			slog.Warn("zero values for device id", "device_id", meter.ID, "device_name", meter.Name)
			continue
		}
		c.meterTemperature[meter.ID].Set(status.Temperature)
		c.meterHumidity[meter.ID].Set(float64(status.Humidity))
		c.meterCO2[meter.ID].Set(float64(status.CO2))
	}

	for _, plug := range c.devicesPlugMiniJP {
		status, err := c.client.Device().Status(context.Background(), plug.ID)
		if err != nil {
			slog.Error("failed to update status", "device_id", plug.ID, "device_name", plug.Name, "error", err)
			continue
		}
		c.plugVoltage[plug.ID].Set(status.Voltage)
		c.plugCurrent[plug.ID].Set(status.ElectricCurrent)
		c.plugPower[plug.ID].Set(status.Weight)
		c.plugMinutesOfDay[plug.ID].Set(float64(status.ElectricityOfDay))
	}
}
