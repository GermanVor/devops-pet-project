package utils

import (
	"bytes"
	"crypto/rsa"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"

	"github.com/GermanVor/devops-pet-project/cmd/agent/metric"
	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/crypto"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

func CollectMetrics(m *metric.RuntimeMetrics) {
	rtm := runtime.MemStats{}
	runtime.ReadMemStats(&rtm)

	m.Alloc = metric.Gauge(rtm.Alloc)
	m.BuckHashSys = metric.Gauge(rtm.BuckHashSys)
	m.Frees = metric.Gauge(rtm.Frees)
	m.GCCPUFraction = metric.Gauge(rtm.GCCPUFraction)
	m.GCSys = metric.Gauge(rtm.GCSys)
	m.HeapAlloc = metric.Gauge(rtm.HeapAlloc)
	m.HeapIdle = metric.Gauge(rtm.HeapIdle)
	m.HeapInuse = metric.Gauge(rtm.HeapInuse)
	m.HeapObjects = metric.Gauge(rtm.HeapObjects)
	m.HeapReleased = metric.Gauge(rtm.HeapReleased)
	m.HeapSys = metric.Gauge(rtm.HeapSys)
	m.LastGC = metric.Gauge(rtm.LastGC)
	m.Lookups = metric.Gauge(rtm.Lookups)
	m.MCacheInuse = metric.Gauge(rtm.MCacheInuse)
	m.MCacheSys = metric.Gauge(rtm.MCacheSys)
	m.MSpanInuse = metric.Gauge(rtm.MSpanInuse)
	m.MSpanSys = metric.Gauge(rtm.MSpanSys)
	m.Mallocs = metric.Gauge(rtm.Mallocs)
	m.NextGC = metric.Gauge(rtm.NextGC)
	m.NumForcedGC = metric.Gauge(rtm.NumForcedGC)
	m.NumGC = metric.Gauge(rtm.NumGC)
	m.OtherSys = metric.Gauge(rtm.OtherSys)
	m.PauseTotalNs = metric.Gauge(rtm.PauseTotalNs)
	m.StackInuse = metric.Gauge(rtm.StackInuse)
	m.StackSys = metric.Gauge(rtm.StackSys)
	m.Sys = metric.Gauge(rtm.Sys)
	m.TotalAlloc = metric.Gauge(rtm.TotalAlloc)

	m.RandomValue = metric.Gauge(rand.Float64())
}

func CollectGopsutilMetrics(m *metric.RuntimeMetrics) {
	v, _ := mem.VirtualMemory()
	m.TotalMemory = metric.Gauge(v.Total)
	m.FreeMemory = metric.Gauge(v.Free)

	count := metric.Gauge(0)
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
	metric := &common.Metric{
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
		metric.SetHash(key)
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
