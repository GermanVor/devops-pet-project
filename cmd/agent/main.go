package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"sync"
	"time"

	metrics "github.com/GermanVor/devops-pet-project/cmd/agent/metrics"
	"github.com/GermanVor/devops-pet-project/cmd/agent/utils"
	"github.com/GermanVor/devops-pet-project/internal/common"
)

var Config = &common.AgentConfig{
	Address:        "localhost:8080",
	PollInterval:   2 * time.Second,
	ReportInterval: 10 * time.Second,
}

func Start(ctx context.Context, endpointURL string, client http.Client) {
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
				metricsPointer := utils.CollectMetrics()

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

				// metrics.ForEach(&metricsCopy, func(metricType, metricName, metricValue string) {
				// 	go func() {
				// 		req, err := utils.BuildRequest(endpointURL, metricType, metricName, metricValue)
				// 		if err != nil {
				// 			log.Println(err)
				// 			return
				// 		}

				// 		resp, err := client.Do(req)
				// 		if err != nil {
				// 			log.Println(err)
				// 			return
				// 		}

				// 		resp.Body.Close()
				// 	}()
				// })

				metrics.ForEach(&metricsCopy, func(metricType, metricName, metricValue string) {
					go func() {
						req, err := utils.BuildRequestV2(endpointURL, metricType, metricName, metricValue)
						if err != nil {
							log.Println(err)
							return
						}

						resp, err := client.Do(req)
						if err != nil {
							log.Println(err)
							return
						}

						resp.Body.Close()
					}()
				})
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

	ctx := context.Background()

	Start(ctx, "http://"+Config.Address, *http.DefaultClient)
}
