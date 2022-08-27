package common

import (
	"flag"
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

	if pollIntervalStr, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		if pollInterval, err := time.ParseDuration(pollIntervalStr); err == nil {
			config.PollInterval = pollInterval
		}
	}

	if reportIntervalStr, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		if reportInterval, err := time.ParseDuration(reportIntervalStr); err == nil {
			config.ReportInterval = reportInterval
		}
	}

	if address, ok := os.LookupEnv("ADDRESS"); ok {
		config.Address = address
	}

	return config
}

const (
	agentAddrUsage   = "Address to send metrics"
	agentPollUsage   = "The time in seconds when Agent collects Metrics."
	agentReportUsage = "The time in seconds when Agent sent Metrics to the Server."
)

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

	if storeFile, ok := os.LookupEnv("STORE_FILE"); ok {
		config.StoreFile = storeFile
	}

	if isRestoreStr, ok := os.LookupEnv("RESTORE"); ok {
		if isRestore, err := strconv.ParseBool(isRestoreStr); err == nil {
			config.IsRestore = isRestore
		}
	}

	if storeIntervalStr, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		if storeInterval, err := time.ParseDuration(storeIntervalStr); err == nil {
			config.StoreInterval = storeInterval
		}
	}

	if address, ok := os.LookupEnv("ADDRESS"); ok {
		config.Address = address
	}

	return config
}

const (
	aUsage = "Address to listen on"
	fUsage = "The name of the file in which Server will store Metrics (Empty name turn off storing Metrics)"
	rUsage = "Bool value. `true` - At startup Server will try to load data from `STORE_FILE`. `false` - Server will create new `STORE_FILE` file in startup."
	iUsage = "The time in seconds after which the current server readings are reset to disk \n (value 0 — makes the recording synchronous)."
)

func InitServerFlagConfig(config *ServerConfig) *ServerConfig {
	flag.StringVar(&config.Address, "a", config.Address, aUsage)
	flag.StringVar(&config.StoreFile, "f", config.StoreFile, fUsage)
	flag.BoolVar(&config.IsRestore, "r", config.IsRestore, rUsage)

	flag.Func("i", iUsage, func(s string) error {
		storeInterval, err := time.ParseDuration(s)

		if err == nil {
			config.StoreInterval = storeInterval
		}

		return err
	})

	return config
}
