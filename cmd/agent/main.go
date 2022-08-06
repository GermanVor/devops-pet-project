package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	metrics "github.com/GermanVor/devops-pet-project/cmd/agent/metrics"
	"github.com/GermanVor/devops-pet-project/cmd/agent/utils"
)

const (
	PollInterval   = 2 * time.Second
	ReportInterval = 10 * time.Second

	EndpointURL = "http://localhost:8080/"
)

func Start(ctx context.Context, endpointURL string, client http.Client) {
	pollTicker := time.NewTicker(PollInterval)
	defer pollTicker.Stop()

	reportInterval := time.NewTicker(ReportInterval)
	defer reportInterval.Stop()

	var mPointer *metrics.RuntimeMetrics
	mux := sync.Mutex{}

	go func() {
		pollCount := metrics.Counter(0)

		for {
			select {
			case <-pollTicker.C:
				mux.Lock()

				mPointer = utils.CollectMetrics()
				mPointer.PollCount = pollCount
				pollCount++

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
					req, err := utils.BuildRequest(endpointURL, metricType, metricName, metricValue)

					if err != nil {
						fmt.Println(err)
						return
					}

					go func() {
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

	Start(ctx, EndpointURL, *http.DefaultClient)
}
