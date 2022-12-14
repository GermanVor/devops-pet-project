package main

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/GermanVor/devops-pet-project/cmd/agent/metrics"
	"github.com/GermanVor/devops-pet-project/cmd/agent/utils"
	"github.com/GermanVor/devops-pet-project/internal/common"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func init() {
	fmt.Printf("Build version:\t%s\n", buildVersion)
	fmt.Printf("Build date:\t%s\n", buildDate)
	fmt.Printf("Build commit:\t%s\n", buildCommit)
}

var Config = &common.AgentConfig{
	Address:        "localhost:8080",
	PollInterval:   1 * time.Second,
	ReportInterval: 2 * time.Second,
}

func SendMetricsV1(metricsObj *metrics.RuntimeMetrics, endpointURL string) {
	metrics.ForEach(metricsObj, func(metricType, metricName, metricValue string) {
		go func() {
			req, err := utils.BuildRequest(endpointURL, metricType, metricName, metricValue)
			if err != nil {
				log.Println(err)
				return
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Println(err)
				return
			}

			resp.Body.Close()
		}()
	})
}

func SendMetricsV2(metricsObj *metrics.RuntimeMetrics, endpointURL, key string, rsaKey *rsa.PublicKey) {
	metrics.ForEach(metricsObj, func(metricType, metricName, metricValue string) {
		go func() {
			req, err := utils.BuildRequestV2(endpointURL, metricType, metricName, metricValue, key, rsaKey)
			if err != nil {
				log.Println(err)
				return
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Println(err)
				return
			}

			resp.Body.Close()
		}()
	})
}

func SendMetricsButchV2(metricsObj *metrics.RuntimeMetrics, endpointURL, key string) {
	metricsArr := []common.Metrics{}

	metrics.ForEach(metricsObj, func(metricType, metricName, metricValue string) {
		metric := common.Metrics{
			ID:    metricName,
			MType: metricType,
		}

		switch metricType {
		case common.GaugeMetricName:
			value, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				return
			}

			metric.Value = &value
		case common.CounterMetricName:
			delta, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				return
			}

			metric.Delta = &delta
		default:
			return
		}

		if key != "" {
			metric.Hash, _ = common.GetMetricHash(&metric, key)
		}

		metricsArr = append(metricsArr, metric)
	})

	metricsBytes, err := json.Marshal(&metricsArr)
	if err != nil {
		return
	}

	req, err := http.NewRequest(http.MethodPost, endpointURL+"/updates/", bytes.NewBuffer(metricsBytes))
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return
	}

	resp.Body.Close()
}

func Start(ctx context.Context, endpointURL, key string, rsaKey *rsa.PublicKey) {
	pollTicker := time.NewTicker(Config.PollInterval)
	defer pollTicker.Stop()

	reportInterval := time.NewTicker(Config.ReportInterval)
	defer reportInterval.Stop()

	var mPointer *metrics.RuntimeMetrics
	pollCount := metrics.Counter(0)

	mux := sync.Mutex{}

	go func() {
		for {
			select {
			case <-pollTicker.C:
				metricsPointer := &metrics.RuntimeMetrics{}

				wg := sync.WaitGroup{}
				wg.Add(2)

				go func() {
					defer wg.Done()
					utils.CollectMetrics(metricsPointer)
				}()
				go func() {
					defer wg.Done()
					utils.CollectGopsutilMetrics(metricsPointer)
				}()

				wg.Wait()

				mux.Lock()

				pollCount++
				metricsPointer.PollCount = pollCount
				mPointer = metricsPointer

				mux.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case <-reportInterval.C:
				mux.Lock()

				metricsCopy := *mPointer
				pollCount = 0

				mux.Unlock()

				// SendMetricsV1(&metricsCopy, endpointURL)
				// SendMetricsV2(&metricsCopy, endpointURL, key, rsaKey)
				SendMetricsButchV2(&metricsCopy, endpointURL, key)

			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()
}

func main() {
	common.InitAgentFlagConfig(Config)
	flag.Parse()
	common.InitAgentEnvConfig(Config)

	log.Println("Agent Config", Config)

	ctx := context.Background()

	var rsaKey *rsa.PublicKey
	if Config.CryptoKey != "" {
		if keyBytes, err := os.ReadFile(Config.CryptoKey); err == nil {
			block, _ := pem.Decode([]byte(keyBytes))
			rsaKey, err = x509.ParsePKCS1PublicKey(block.Bytes)

			if err != nil {
				log.Println(err.Error())
			}

			log.Println("Agent will Encrypt Metrics (/updates/)")
		}
	}

	Start(ctx, "http://"+Config.Address, Config.Key, rsaKey)
}
