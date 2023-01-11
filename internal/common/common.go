package common

import (
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	json "encoding/json"
	"encoding/pem"
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
type Metric struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	Hash  *string  `json:"hash,omitempty"`  // значение хеш-функции
}

var (
	ErrGetMetricHash = errors.New("do not call SetMetricHash before metric.value is assigned")
)

func createMetricHash(metricsStatsStr, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(metricsStatsStr))

	return fmt.Sprintf("%x", h.Sum(nil))
}

// getMetricHash build hash of the metrics
// based on the sha256
func getHashOfMetric(metrics *Metric, key string) (string, error) {
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

func (m *Metric) SetHash(key string) error {
	hash, err := getHashOfMetric(m, key)
	if err != nil {
		return err
	}

	m.Hash = &hash
	return nil
}

func (m *Metric) CheckHash(key string) (bool, error) {
	hash, err := getHashOfMetric(m, key)
	if err != nil {
		return false, err
	}

	return *m.Hash == hash, nil
}

func readPublicCryptoKey(keyFilePath string) (*rsa.PublicKey, error) {
	keyBytes, err := os.ReadFile(keyFilePath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode([]byte(keyBytes))
	rsaKey, err := x509.ParsePKCS1PublicKey(block.Bytes)

	if err != nil {
		return nil, err
	}

	return rsaKey, nil
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}

		return nil
	default:
		return errors.New("invalid duration")
	}
}

type PublicKey struct {
	*rsa.PublicKey
}

func (k *PublicKey) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		key, err := readPublicCryptoKey(value)
		if err == nil {
			k.PublicKey = key
		} else {
			log.Println("Can not read Crypto Key", err)
		}

		return nil
	default:
		return errors.New("invalid duration")
	}
}

type AgentConfig struct {
	Address        string   `json:"address,omitempty"`
	PollInterval   Duration `json:"poll_interval,omitempty"`
	ReportInterval Duration `json:"report_interval,omitempty"`

	CryptoKey PublicKey `json:"crypto_key,omitempty"`

	Key string
}

func readPrivateCryptoKey(keyFilePath string) (*rsa.PrivateKey, error) {
	ketData, err := os.ReadFile(keyFilePath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(ketData)

	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return rsaKey, nil
}

type PrivateKey struct {
	*rsa.PrivateKey
}

func (k *PrivateKey) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		key, err := readPrivateCryptoKey(value)
		if err == nil {
			k.PrivateKey = key
		} else {
			log.Println("Can not read Crypto Key", err)
		}

		return nil
	default:
		return errors.New("invalid duration")
	}
}

type ServerConfig struct {
	Address       string   `json:"address,omitempty"`
	StoreInterval Duration `json:"store_interval,omitempty"`
	StoreFile     string   `json:"store_file,omitempty"`
	IsRestore     bool     `json:"restore,omitempty"`

	CryptoKey PrivateKey `json:"crypto_key,omitempty"`

	DataBaseDSN string `json:"database_dsn,omitempty"`

	Key string

	TrustedSubnet string `json:"trusted_subnet,omitempty"`
}

func InitAgentEnvConfig(config *AgentConfig) *AgentConfig {
	godotenv.Load(".env")

	if pollIntervalStr, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		if pollInterval, err := time.ParseDuration(pollIntervalStr); err == nil {
			config.PollInterval.Duration = pollInterval
		}
	}

	if reportIntervalStr, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		if reportInterval, err := time.ParseDuration(reportIntervalStr); err == nil {
			config.ReportInterval.Duration = reportInterval
		}
	}

	if address, ok := os.LookupEnv("ADDRESS"); ok {
		config.Address = address
	}

	if hashKey, ok := os.LookupEnv("KEY"); ok {
		config.Key = hashKey
	}

	if cryptoKeyPath, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		key, err := readPublicCryptoKey(cryptoKeyPath)
		if err == nil {
			config.CryptoKey = PublicKey{key}
		}
	}

	return config
}

const (
	agentAddrUsage   = "Address to send metrics"
	agentPollUsage   = "The time in seconds when Agent collects Metric."
	agentReportUsage = "The time in seconds when Agent sent Metric to the Server."
	agentKey         = "Static key (for educational purposes) for hash generation"
	agentCKUsage     = "Asymmetric encryption publick key"
)

func InitAgentFlagConfig(config *AgentConfig) *AgentConfig {
	flag.StringVar(&config.Address, "a", config.Address, agentAddrUsage)
	flag.StringVar(&config.Key, "k", config.Key, agentKey)

	flag.Func("p", agentPollUsage, func(s string) error {
		pollInterval, err := time.ParseDuration(s)

		if err == nil {
			config.PollInterval.Duration = pollInterval
		}

		return err
	})

	flag.Func("r", agentReportUsage, func(s string) error {
		reportInterval, err := time.ParseDuration(s)

		if err == nil {
			config.ReportInterval.Duration = reportInterval
		}

		return err
	})

	flag.Func("crypto-key", agentCKUsage, func(cryptoKeyPath string) error {
		if cryptoKeyPath == "" {
			return nil
		}

		key, err := readPublicCryptoKey(cryptoKeyPath)
		if err != nil {
			return err
		}

		config.CryptoKey = PublicKey{key}
		return nil
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
			config.StoreInterval = Duration{storeInterval}
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

	if cryptoKeyPath, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		key, err := readPrivateCryptoKey(cryptoKeyPath)
		if err == nil {
			config.CryptoKey = PrivateKey{key}
		}
	}

	if trustedSubnet, ok := os.LookupEnv("TRUSTED_SUBNET"); ok {
		config.TrustedSubnet = trustedSubnet
	}

	return config
}

const (
	aUsage  = "Address to listen on"
	fUsage  = "The name of the file in which Server will store Metric (Empty name turn off storing Metric)"
	rUsage  = "Bool value. `true` - At startup Server will try to load data from `STORE_FILE`. `false` - Server will create new `STORE_FILE` file in startup."
	iUsage  = "The time in seconds after which the current server readings are reset to disk \n (value 0 — makes the recording synchronous)."
	kUsage  = "Static key (for educational purposes) for hash generation"
	dUsage  = "Database address to connect server with (for exemple postgres://zzman:@localhost:5432/postgres)"
	ckUsage = "Asymmetric encryption private key"
	tUsage  = ""
)

func InitServerFlagConfig(config *ServerConfig) *ServerConfig {
	flag.StringVar(&config.Address, "a", config.Address, aUsage)
	flag.StringVar(&config.StoreFile, "f", config.StoreFile, fUsage)
	flag.BoolVar(&config.IsRestore, "r", config.IsRestore, rUsage)
	flag.StringVar(&config.Key, "k", config.Key, kUsage)
	flag.StringVar(&config.DataBaseDSN, "d", config.DataBaseDSN, dUsage)
	flag.StringVar(&config.TrustedSubnet, "t", config.TrustedSubnet, tUsage)

	flag.Func("crypto-key", agentCKUsage, func(cryptoKeyPath string) error {
		if cryptoKeyPath == "" {
			return nil
		}

		key, err := readPrivateCryptoKey(cryptoKeyPath)
		if err != nil {
			return err
		}

		config.CryptoKey = PrivateKey{key}
		return nil
	})

	flag.Func("i", iUsage, func(s string) error {
		storeInterval, err := time.ParseDuration(s)

		if err == nil {
			config.StoreInterval.Duration = storeInterval
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

func min[T ~int](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func Chunks[T any](arr []T, chunkSize int) [][]T {
	if len(arr) == 0 || chunkSize <= 0 {
		return nil
	}

	chunks := make([][]T, (len(arr)-1)/chunkSize+1)

	for i := range chunks {
		leftIdx := i * chunkSize
		rightIdx := leftIdx + min(chunkSize, len(arr)-leftIdx)

		chunks[i] = arr[leftIdx:rightIdx:rightIdx]
	}

	return chunks
}
