package common

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

const (
	CounterMetricName = "counter"
	GaugeMetricName   = "gauge"
)

type Config struct {
	Address        string
	PollInterval   time.Duration
	ReportInterval time.Duration
}

func InitConfig() *Config {
	config := &Config{}

	pollIntervalStr := os.Getenv("POLL_INTERVAL")
	if pollIntervalStr == "" {
		fmt.Println("Empty POLL_INTERVAL")
	}
	pollIntervalStr = pollIntervalStr[:len(pollIntervalStr)-1]

	if pollInterval, err := strconv.ParseInt(pollIntervalStr, 10, 64); err == nil {
		config.PollInterval = time.Duration(pollInterval) * time.Second
	} else {
		fmt.Println(err.Error())
	}

	reportIntervalStr := os.Getenv("REPORT_INTERVAL")
	if reportIntervalStr == "" {
		fmt.Println("Empty POLL_INTERVAL")
	}
	reportIntervalStr = reportIntervalStr[:len(reportIntervalStr)-1]

	if reportInterval, err := strconv.ParseInt(reportIntervalStr, 10, 64); err == nil {
		config.ReportInterval = time.Duration(reportInterval) * time.Second
	} else {
		fmt.Println(err.Error())
	}

	config.Address = os.Getenv("ADDRESS")
	if config.Address == "" {
		fmt.Println("Empty ADDRESS")
	}

	return config
}
