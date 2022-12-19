package utils

import (
	"bytes"
	"crypto/rsa"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"

	"github.com/GermanVor/devops-pet-project/cmd/agent/metrics"
	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/crypto"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

func CollectMetrics(m *metrics.RuntimeMetrics) {
	rtm := runtime.MemStats{}
	runtime.ReadMemStats(&rtm)

	m.Alloc = metrics.Gauge(rtm.Alloc)
	m.BuckHashSys = metrics.Gauge(rtm.BuckHashSys)
	m.Frees = metrics.Gauge(rtm.Frees)
	m.GCCPUFraction = metrics.Gauge(rtm.GCCPUFraction)
	m.GCSys = metrics.Gauge(rtm.GCSys)
	m.HeapAlloc = metrics.Gauge(rtm.HeapAlloc)
	m.HeapIdle = metrics.Gauge(rtm.HeapIdle)
	m.HeapInuse = metrics.Gauge(rtm.HeapInuse)
	m.HeapObjects = metrics.Gauge(rtm.HeapObjects)
	m.HeapReleased = metrics.Gauge(rtm.HeapReleased)
	m.HeapSys = metrics.Gauge(rtm.HeapSys)
	m.LastGC = metrics.Gauge(rtm.LastGC)
	m.Lookups = metrics.Gauge(rtm.Lookups)
	m.MCacheInuse = metrics.Gauge(rtm.MCacheInuse)
	m.MCacheSys = metrics.Gauge(rtm.MCacheSys)
	m.MSpanInuse = metrics.Gauge(rtm.MSpanInuse)
	m.MSpanSys = metrics.Gauge(rtm.MSpanSys)
	m.Mallocs = metrics.Gauge(rtm.Mallocs)
	m.NextGC = metrics.Gauge(rtm.NextGC)
	m.NumForcedGC = metrics.Gauge(rtm.NumForcedGC)
	m.NumGC = metrics.Gauge(rtm.NumGC)
	m.OtherSys = metrics.Gauge(rtm.OtherSys)
	m.PauseTotalNs = metrics.Gauge(rtm.PauseTotalNs)
	m.StackInuse = metrics.Gauge(rtm.StackInuse)
	m.StackSys = metrics.Gauge(rtm.StackSys)
	m.Sys = metrics.Gauge(rtm.Sys)
	m.TotalAlloc = metrics.Gauge(rtm.TotalAlloc)

	m.RandomValue = metrics.Gauge(rand.Float64())
}

func CollectGopsutilMetrics(m *metrics.RuntimeMetrics) {
	v, _ := mem.VirtualMemory()
	m.TotalMemory = metrics.Gauge(v.Total)
	m.FreeMemory = metrics.Gauge(v.Free)

	count := metrics.Gauge(0)
	a, _ := cpu.Percent(0, true)
	for _, percent := range a {
		if percent != 0 {
			count++
		}
	}
	m.CPUutilization1 = count
}

func BuildEndpointURL(endpointURL, metricType, metricName, metricValue string) string {
	return endpointURL + "/update/" + metricType + "/" + metricName + "/" + metricValue
}

func BuildRequest(endpointURL, metricType, metricName, metricValue string) (*http.Request, error) {
	currentURL := BuildEndpointURL(endpointURL, metricType, metricName, metricValue)

	req, err := http.NewRequest(http.MethodPost, currentURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "text/plain")

	return req, err
}

func BuildRequestV2(
	endpointURL,
	metricType,
	metricName,
	metricValue,
	key string,
	rsaKey *rsa.PublicKey,
) (*http.Request, error) {
	metric := &common.Metrics{
		ID:    metricName,
		MType: metricType,
	}

	if common.GaugeMetricName == metricType {
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return nil, err
		}

		metric.Value = &value
	} else {
		delta, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return nil, err
		}

		metric.Delta = &delta
	}

	if key != "" {
		metric.Hash, _ = common.GetMetricHash(metric, key)
	}

	metricBytes, err := metric.MarshalJSON()
	if err != nil {
		return nil, err
	}

	if rsaKey != nil {
		metricBytes, _ = crypto.RSAEncrypt(metricBytes, rsaKey)
	}

	req, err := http.NewRequest(http.MethodPost, endpointURL+"/update/", bytes.NewBuffer(metricBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	return req, nil
}
