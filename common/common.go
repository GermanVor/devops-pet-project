package common

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
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

type AgentConfig struct {
	Address        string
	PollInterval   time.Duration
	ReportInterval time.Duration
}

type ServerConfig struct {
	Address       string
	StoreInterval time.Duration
	StoreFile     string
	IsRestore     bool
}

func InitAgentEnvConfig(config *AgentConfig) *AgentConfig {
	godotenv.Load(".env")

	if pollInterval, err := time.ParseDuration(os.Getenv("POLL_INTERVAL")); err == nil {
		config.PollInterval = pollInterval
	}

	if reportInterval, err := time.ParseDuration(os.Getenv("REPORT_INTERVAL")); err == nil {
		config.ReportInterval = reportInterval
	}

	config.Address = os.Getenv("ADDRESS")

	return config
}

const agentAddrUsage = "Address to send metrics"
const agentPollUsage = "The time in seconds when Agent collects Metrics."
const agentReportUsage = "The time in seconds when Agent sent Metrics to the Server."

func InitAgentFlagConfig(config *AgentConfig) *AgentConfig {
	flag.StringVar(&config.Address, "a", config.Address, agentAddrUsage)

	flag.Func("p", agentPollUsage, func(s string) error {
		pollInterval, err := time.ParseDuration(s)

		if err == nil {
			config.PollInterval = pollInterval
		}

		return err
	})

	flag.Func("r", agentReportUsage, func(s string) error {
		reportInterval, err := time.ParseDuration(s)

		if err == nil {
			config.ReportInterval = reportInterval
		}

		return err
	})

	return config
}

func InitServerEnvConfig(config *ServerConfig) *ServerConfig {
	godotenv.Load(".env")

	config.StoreFile = os.Getenv("STORE_FILE")
	if config.StoreFile == "" {
		fmt.Println("Empty STORE_FILE. Server will not save data in local file")
	}

	if isRestore, err := strconv.ParseBool(os.Getenv("RESTORE")); err == nil {
		config.IsRestore = isRestore
	}

	if storeInterval, err := time.ParseDuration(os.Getenv("STORE_INTERVAL")); err == nil {
		config.StoreInterval = storeInterval
	}

	config.Address = os.Getenv("ADDRESS")

	return config
}

const aUsage = "Address to listen on"
const fUsage = "The name of the file in which Server will store Metrics (Empty name turn off storing Metrics)"
const rUsage = "Bool value. `true` - At startup Server will try to load data from `STORE_FILE`. `false` - Server will create new `STORE_FILE` file in startup."
const iUsage = "The time in seconds after which the current server readings are reset to disk \n (value 0 — makes the recording synchronous)."

func InitServerFlagConfig(config *ServerConfig) *ServerConfig {
	flag.StringVar(&config.Address, "a", config.Address, aUsage)
	flag.StringVar(&config.StoreFile, "f", config.StoreFile, fUsage)

	flag.Func("r", rUsage, func(s string) error {
		isRestore, err := strconv.ParseBool(s)

		if err == nil {
			config.IsRestore = isRestore
		}

		return err
	})

	flag.Func("i", iUsage, func(s string) error {
		storeInterval, err := time.ParseDuration(s)

		if err == nil {
			config.StoreInterval = storeInterval
		}

		return err
	})

	return config
}
