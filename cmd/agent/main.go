package main

import (
	"context"
	"net/http"
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

	m := &metrics.RuntimeMetrics{}
	pollCount := metrics.Counter(0)

	for {
		select {
		case <-pollTicker.C:
			utils.CollectMetrics(m, pollCount)
			pollCount++
		case <-reportInterval.C:
			metrics.ForEach(m, func(metricType, metricName, metricValue string) {
				req, err := utils.BuildRequest(endpointURL, metricType, metricName, metricValue)

				if err == nil {
					resp, _ := client.Do(req)
					if resp != nil {
						resp.Body.Close()
					}
				}
			})
		case <-ctx.Done():
			return
		}
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(100)*ReportInterval)
	defer cancel()

	Start(ctx, EndpointURL, *http.DefaultClient)
}
