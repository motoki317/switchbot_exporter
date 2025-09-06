package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/castai/promwrite"
	"github.com/nasa9084/go-switchbot/v3"
)

var (
	remoteWriteURL = flag.String("remote-write-url", "http://localhost:8428/api/v1/write", "The remote write URL")

	deviceType = flag.String("device-type", string(switchbot.Meter), "SwitchBot device type")
	deviceID   = flag.String("device-id", "", "SwitchBot device ID")
	deviceName = flag.String("device-name", "", "SwitchBot device name")

	csvFile  = flag.String("csv", "", "CSV file path")
	timezone = flag.String("timezone", "Asia/Tokyo", "Timezone")
)

func main() {
	flag.Parse()
	if *remoteWriteURL == "" {
		panic("remote write URL is required")
	}

	switch *deviceType {
	case string(switchbot.Meter):
	default:
		panic("unsupported device type")
	}
	if *deviceID == "" {
		panic("device ID is required")
	}
	if *deviceName == "" {
		panic("device name is required")
	}

	if *csvFile == "" {
		panic("csv file path is required")
	}
	loc, err := time.LoadLocation(*timezone)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	client := promwrite.NewClient(*remoteWriteURL)

	slog.Info("start backfilling...", "remote_write_url", *remoteWriteURL)
	ch := make(chan meterRecord, 1000)
	go func() {
		err := readMeterCSV(*csvFile, loc, ch)
		if err != nil {
			panic(err)
		}
	}()
	err = backfillMeter(ctx, client, *deviceID, *deviceName, ch)
	if err != nil {
		panic(err)
	}
	slog.Info("backfilling completed")
}

type meterRecord struct {
	timestamp   time.Time
	temperature float64
	humidity    float64
}

func readMeterCSV(csvFile string, loc *time.Location, ch chan<- meterRecord) error {
	const (
		timestampColName   = "Timestamp"
		temperatureColName = "Temperature_Celsius(°C)"
		humidityColName    = "Relative_Humidity(%)"
	)
	var (
		timestampColIndex   = -1
		temperatureColIndex = -1
		humidityColIndex    = -1
	)

	f, err := os.Open(csvFile)
	if err != nil {
		return err
	}
	defer f.Close()
	reader := csv.NewReader(f)

	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("reading CSV file: %w", err)
	}
	for i, col := range header {
		switch col {
		case timestampColName:
			timestampColIndex = i
		case temperatureColName:
			temperatureColIndex = i
		case humidityColName:
			humidityColIndex = i
		}
	}
	if timestampColIndex == -1 || temperatureColIndex == -1 || humidityColIndex == -1 {
		return errors.New("failed to find timestamp, temperature, or humidity column in CSV file")
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading CSV file: %w", err)
		}
		timestamp, err := time.ParseInLocation("Jan 02, 2006 15:04:05", record[timestampColIndex], loc)
		if err != nil {
			return fmt.Errorf("parsing timestamp: %w", err)
		}
		temperature, err := strconv.ParseFloat(record[temperatureColIndex], 64)
		if err != nil {
			return fmt.Errorf("parsing temperature: %w", err)
		}
		humidity, err := strconv.ParseFloat(record[humidityColIndex], 64)
		if err != nil {
			return fmt.Errorf("parsing humidity: %w", err)
		}
		ch <- meterRecord{
			timestamp:   timestamp,
			temperature: temperature,
			humidity:    humidity,
		}
	}
	close(ch)
	return nil
}

func backfillMeter(ctx context.Context, client *promwrite.Client, deviceID, deviceName string, ch <-chan meterRecord) error {
	const (
		writeRecordsBatch = 10000
	)
	commonLabels := []promwrite.Label{
		{Name: "device_id", Value: deviceID},
		{Name: "device_name", Value: deviceName},
	}
	temperatureLabels := []promwrite.Label{{Name: "__name__", Value: "switchbot_temperature"}}
	temperatureLabels = append(temperatureLabels, commonLabels...)
	humidityLabels := []promwrite.Label{{Name: "__name__", Value: "switchbot_humidity"}}
	humidityLabels = append(humidityLabels, commonLabels...)

	totalRecords := 0
	records := make([]promwrite.TimeSeries, 0, writeRecordsBatch)
	for record := range ch {
		records = append(records, promwrite.TimeSeries{
			Labels: temperatureLabels,
			Sample: promwrite.Sample{
				Time:  record.timestamp,
				Value: record.temperature,
			},
		})
		records = append(records, promwrite.TimeSeries{
			Labels: humidityLabels,
			Sample: promwrite.Sample{
				Time:  record.timestamp,
				Value: record.humidity,
			},
		})
		totalRecords += 2

		if len(records) >= writeRecordsBatch {
			slog.Info("writing records", "total_records", totalRecords)
			_, err := client.Write(ctx, &promwrite.WriteRequest{TimeSeries: records})
			if err != nil {
				return fmt.Errorf("writing to remote write: %w", err)
			}
			records = records[:0]
		}
	}

	if len(records) > 0 {
		slog.Info("writing records", "total_records", totalRecords)
		_, err := client.Write(ctx, &promwrite.WriteRequest{TimeSeries: records})
		if err != nil {
			return fmt.Errorf("writing to remote write: %w", err)
		}
	}
	return nil
}
