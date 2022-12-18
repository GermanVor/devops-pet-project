package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rsa"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/GermanVor/devops-pet-project/cmd/agent/metrics"
	"github.com/GermanVor/devops-pet-project/cmd/agent/utils"
	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/crypto"
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
	PollInterval:   common.Duration{Duration: time.Second},
	ReportInterval: common.Duration{Duration: 2 * time.Second},
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

func SendMetricsButchV2(metricsObj *metrics.RuntimeMetrics, endpointURL, key string, rsaKey *rsa.PublicKey) {
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

	if rsaKey != nil {
		var buf bytes.Buffer
		g := gzip.NewWriter(&buf)
		if _, err = g.Write(metricsBytes); err != nil {
			log.Println(err)
			return
		}
		if err = g.Close(); err != nil {
			log.Println(err)
			return
		}

		metricsBytes, err = crypto.RSAEncrypt(buf.Bytes(), rsaKey)
		if err != nil {
			log.Println(err)
			return
		}
	}

	req, err := http.NewRequest(http.MethodPost, endpointURL+"/updates/", bytes.NewBuffer(metricsBytes))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")

	if rsaKey != nil {
		req.Header.Set("Content-Encoding", "gzip")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return
	}

	resp.Body.Close()
}

func Start(ctx context.Context, endpointURL string, rsaKey *rsa.PublicKey) {
	pollTicker := time.NewTicker(Config.PollInterval.Duration)
	defer pollTicker.Stop()

	reportInterval := time.NewTicker(Config.ReportInterval.Duration)
	defer reportInterval.Stop()

	var mPointer *metrics.RuntimeMetrics
	pollCount := metrics.Counter(0)

	mux := sync.Mutex{}

	mainWG := sync.WaitGroup{}
	mainWG.Add(2)

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
				mainWG.Done()
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
				// SendMetricsV2(&metricsCopy, endpointURL, Config.Key, rsaKey)
				SendMetricsButchV2(&metricsCopy, endpointURL, Config.Key, rsaKey)

			case <-ctx.Done():
				mainWG.Done()
				return
			}
		}
	}()

	mainWG.Wait()
}

func initConfig() {
	common.InitJSONConfig(Config)
	common.InitAgentFlagConfig(Config)
	flag.Parse()

	common.InitAgentEnvConfig(Config)
}

func main() {
	initConfig()
	log.Println("Agent Config", Config)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-sigs
		cancel()
	}()

	if Config.CryptoKey.PublicKey != nil {
		log.Println("Agent will Encrypt Metrics (/updates/)")
	}

	Start(ctx, "http://"+Config.Address, Config.CryptoKey.PublicKey)
	log.Println("Agent finished work")
}
