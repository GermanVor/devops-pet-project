package common

import (
	"crypto/hmac"
	"crypto/sha256"
	json "encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

//go:generate easyjson common.go

const (
	CounterMetricName = "counter"
	GaugeMetricName   = "gauge"
)

//easyjson:json
type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	Hash  string   `json:"hash,omitempty"`  // значение хеш-функции
}

func createMetricHash(metricsStatsStr, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(metricsStatsStr))

	return fmt.Sprintf("%x", h.Sum(nil))
}

var (
	ErrGetMetricHash = errors.New("do not call SetMetricHash before metric.value is assigned")
)

// GetMetricHash build hash of the metrics
// based on the sha256
func GetMetricHash(metrics *Metrics, key string) (string, error) {
	var hash string

	if metrics.MType == GaugeMetricName {
		if metrics.Value == nil {
			return "", ErrGetMetricHash
		}

		hash = createMetricHash(fmt.Sprintf("%s:gauge:%f", metrics.ID, *metrics.Value), key)
	} else if metrics.MType == CounterMetricName {
		if metrics.Delta == nil {
			return "", ErrGetMetricHash
		}

		hash = createMetricHash(fmt.Sprintf("%s:counter:%d", metrics.ID, *metrics.Delta), key)
	} else {
		return "", errors.New("unknown metric type: " + metrics.MType)
	}

	return hash, nil
}

type AgentConfig struct {
	Address        string `json:"address,omitempty"`
	PollInterval   string `json:"poll_interval,omitempty"`
	ReportInterval string `json:"report_interval,omitempty"`

	CryptoKey string `json:"crypto_key,omitempty"`

	Key string
}

type ServerConfig struct {
	Address       string `json:"address,omitempty"`
	StoreInterval string `json:"store_interval,omitempty"`
	StoreFile     string `json:"store_file,omitempty"`
	IsRestore     bool   `json:"restore,omitempty"`

	CryptoKey string `json:"crypto_key,omitempty"`

	DataBaseDSN string `json:"database_dsn,omitempty"`

	Key string
}

func InitAgentEnvConfig(config *AgentConfig) *AgentConfig {
	godotenv.Load(".env")

	if pollIntervalStr, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		if _, err := time.ParseDuration(pollIntervalStr); err == nil {
			config.PollInterval = pollIntervalStr
		}
	}

	if reportIntervalStr, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		if _, err := time.ParseDuration(reportIntervalStr); err == nil {
			config.ReportInterval = reportIntervalStr
		}
	}

	if address, ok := os.LookupEnv("ADDRESS"); ok {
		config.Address = address
	}

	if hashKey, ok := os.LookupEnv("KEY"); ok {
		config.Key = hashKey
	}

	if cryptoKey, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		config.CryptoKey = cryptoKey
	}

	return config
}

const (
	agentAddrUsage   = "Address to send metrics"
	agentPollUsage   = "The time in seconds when Agent collects Metrics."
	agentReportUsage = "The time in seconds when Agent sent Metrics to the Server."
	agentKey         = "Static key (for educational purposes) for hash generation"
	agentCKUsage     = "Asymmetric encryption publick key"
)

func InitAgentFlagConfig(config *AgentConfig) *AgentConfig {
	flag.StringVar(&config.Address, "a", config.Address, agentAddrUsage)
	flag.StringVar(&config.Key, "k", config.Key, agentKey)

	flag.Func("p", agentPollUsage, func(s string) error {
		_, err := time.ParseDuration(s)

		if err == nil {
			config.PollInterval = s
		}

		return err
	})

	flag.Func("r", agentReportUsage, func(s string) error {
		_, err := time.ParseDuration(s)

		if err == nil {
			config.ReportInterval = s
		}

		return err
	})

	flag.StringVar(&config.CryptoKey, "crypto-key", config.CryptoKey, agentCKUsage)

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
		if _, err := time.ParseDuration(storeIntervalStr); err == nil {
			config.StoreInterval = storeIntervalStr
		}
	}

	if address, ok := os.LookupEnv("ADDRESS"); ok {
		config.Address = address
	}

	if hashKey, ok := os.LookupEnv("KEY"); ok {
		config.Key = hashKey
	}

	if dataBaseDSN, ok := os.LookupEnv("DATABASE_DSN"); ok {
		config.DataBaseDSN = dataBaseDSN
	}

	if cryptoKey, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		config.CryptoKey = cryptoKey
	}

	return config
}

const (
	aUsage  = "Address to listen on"
	fUsage  = "The name of the file in which Server will store Metrics (Empty name turn off storing Metrics)"
	rUsage  = "Bool value. `true` - At startup Server will try to load data from `STORE_FILE`. `false` - Server will create new `STORE_FILE` file in startup."
	iUsage  = "The time in seconds after which the current server readings are reset to disk \n (value 0 — makes the recording synchronous)."
	kUsage  = "Static key (for educational purposes) for hash generation"
	dUsage  = "Database address to connect server with (for exemple postgres://zzman:@localhost:5432/postgres)"
	ckUsage = "Asymmetric encryption private key"
)

func InitServerFlagConfig(config *ServerConfig) *ServerConfig {
	flag.StringVar(&config.Address, "a", config.Address, aUsage)
	flag.StringVar(&config.StoreFile, "f", config.StoreFile, fUsage)
	flag.BoolVar(&config.IsRestore, "r", config.IsRestore, rUsage)
	flag.StringVar(&config.Key, "k", config.Key, kUsage)
	flag.StringVar(&config.DataBaseDSN, "d", config.DataBaseDSN, dUsage)
	flag.StringVar(&config.CryptoKey, "crypto-key", config.CryptoKey, ckUsage)

	flag.Func("i", iUsage, func(s string) error {
		_, err := time.ParseDuration(s)

		if err == nil {
			config.StoreInterval = s
		}

		return err
	})

	return config
}

func InitJSONConfig[T AgentConfig | ServerConfig](config *T) *T {
	configPath := ""

	if path, ok := os.LookupEnv("CONFIG"); ok {
		configPath = path
	} else {
		flag.StringVar(&configPath, "c", configPath, "")
		flag.StringVar(&configPath, "config", configPath, "")
	}

	if configPath == "" {
		return config
	}

	configFile, err := os.Open(configPath)
	if err != nil {
		log.Println("Opening config file", err.Error())
	}

	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(config); err != nil {
		log.Println("Parsing config file", err.Error())
	}

	return config
}
