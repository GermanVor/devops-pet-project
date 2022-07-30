package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"
)

const (
	PollInterval   = 2 * time.Second
	ReportInterval = 10 * time.Second

	BaseURL = "http://localhost:8080/"
)

type Counter int64
type Gauge float64

type RuntimeMetrics struct {
	Alloc         Gauge
	BuckHashSys   Gauge
	Frees         Gauge
	GCCPUFraction Gauge
	GCSys         Gauge
	HeapAlloc     Gauge
	HeapIdle      Gauge
	HeapInuse     Gauge
	HeapObjects   Gauge
	HeapReleased  Gauge
	HeapSys       Gauge
	LastGC        Gauge
	Lookups       Gauge
	MCacheInuse   Gauge
	MCacheSys     Gauge
	MSpanInuse    Gauge
	MSpanSys      Gauge
	Mallocs       Gauge
	NextGC        Gauge
	NumForcedGC   Gauge
	NumGC         Gauge
	OtherSys      Gauge
	PauseTotalNs  Gauge
	StackInuse    Gauge
	StackSys      Gauge
	Sys           Gauge
	TotalAlloc    Gauge

	PollCount   Counter
	RandomValue Gauge
}

func collectMetrics(metrics *RuntimeMetrics, pollCount Counter) {
	rtm := runtime.MemStats{}
	runtime.ReadMemStats(&rtm)

	metrics.Alloc = Gauge(rtm.Alloc)
	metrics.BuckHashSys = Gauge(rtm.BuckHashSys)
	metrics.Frees = Gauge(rtm.Frees)
	metrics.GCCPUFraction = Gauge(rtm.GCCPUFraction)
	metrics.GCSys = Gauge(rtm.GCSys)
	metrics.HeapAlloc = Gauge(rtm.HeapAlloc)
	metrics.HeapIdle = Gauge(rtm.HeapIdle)
	metrics.HeapInuse = Gauge(rtm.HeapInuse)
	metrics.HeapObjects = Gauge(rtm.HeapObjects)
	metrics.HeapReleased = Gauge(rtm.HeapReleased)
	metrics.HeapSys = Gauge(rtm.HeapSys)
	metrics.LastGC = Gauge(rtm.LastGC)
	metrics.Lookups = Gauge(rtm.Lookups)
	metrics.MCacheInuse = Gauge(rtm.MCacheInuse)
	metrics.MCacheSys = Gauge(rtm.MCacheSys)
	metrics.MSpanInuse = Gauge(rtm.MSpanInuse)
	metrics.MSpanSys = Gauge(rtm.MSpanSys)
	metrics.Mallocs = Gauge(rtm.Mallocs)
	metrics.NextGC = Gauge(rtm.NextGC)
	metrics.NumForcedGC = Gauge(rtm.NumForcedGC)
	metrics.NumGC = Gauge(rtm.NumGC)
	metrics.OtherSys = Gauge(rtm.OtherSys)
	metrics.PauseTotalNs = Gauge(rtm.PauseTotalNs)
	metrics.StackInuse = Gauge(rtm.StackInuse)
	metrics.StackSys = Gauge(rtm.StackSys)
	metrics.Sys = Gauge(rtm.Sys)
	metrics.TotalAlloc = Gauge(rtm.TotalAlloc)

	metrics.PollCount = pollCount
	metrics.RandomValue = Gauge(rand.Float64())
}

func reportMetrics(metrics *RuntimeMetrics) error {
	v := reflect.ValueOf(*metrics)

	for i := 0; i < v.NumField(); i++ {
		metricType := strings.ToLower(v.Field(i).Type().Name())
		metricName := v.Type().Field(i).Name
		metricValue := fmt.Sprintf("%v", v.Field(i))

		currentURL := BaseURL + "update/" + metricType + "/" + metricName + "/" + metricValue

		req, err := http.NewRequest(http.MethodPost, currentURL, nil)
		if err != nil {
			return err
		}

		req.Header.Add("Content-Type", "text/plain")

		resp, err := http.DefaultClient.Do(req)
		if resp != nil {
			resp.Body.Close()
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func NewMonitor() {
	pollTicker := time.NewTicker(PollInterval)
	reportInterval := time.NewTicker(ReportInterval)

	metrics := &RuntimeMetrics{}
	pollCount := Counter(1)

	for {
		select {
		case <-pollTicker.C:
			collectMetrics(metrics, pollCount)
			pollCount++
		case <-reportInterval.C:
			reportMetrics(metrics)
		}
	}
}

func main() {
	NewMonitor()
}
