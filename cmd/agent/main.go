package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	metrics "github.com/GermanVor/devops-pet-project/cmd/agent/metrics"
	"github.com/GermanVor/devops-pet-project/cmd/agent/utils"
	"github.com/GermanVor/devops-pet-project/common"
	"github.com/joho/godotenv"
)

var Config *common.Config

func init() {
	godotenv.Load(".env")
	Config = common.InitConfig()
}

func Start(ctx context.Context, endpointURL string, client http.Client) {
	pollTicker := time.NewTicker(Config.PollInterval)
	defer pollTicker.Stop()

	reportInterval := time.NewTicker(Config.ReportInterval)
	defer reportInterval.Stop()

	var mPointer *metrics.RuntimeMetrics
	mux := sync.Mutex{}

	go func() {
		pollCount := metrics.Counter(0)

		for {
			select {
			case <-pollTicker.C:
				metricsPointer := utils.CollectMetrics()
				metricsPointer.PollCount = pollCount
				pollCount++

				mux.Lock()
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
				mux.Unlock()

				metrics.ForEach(&metricsCopy, func(metricType, metricName, metricValue string) {
					go func() {
						req, err := utils.BuildRequest(endpointURL, metricType, metricName, metricValue)
						if err != nil {
							fmt.Println(err)
							return
						}

						resp, err := client.Do(req)
						if err != nil {
							fmt.Println(err)
							return
						}

						resp.Body.Close()
					}()
				})

				metrics.ForEach(&metricsCopy, func(metricType, metricName, metricValue string) {
					go func() {
						req, err := utils.BuildRequestV2(endpointURL, metricType, metricName, metricValue)
						if err != nil {
							fmt.Println(err)
							return
						}

						resp, err := client.Do(req)
						if err != nil {
							fmt.Println(err)
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
	ctx := context.Background()

	Start(ctx, "http://"+Config.Address, *http.DefaultClient)
}
