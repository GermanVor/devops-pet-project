package metrics

import (
	"fmt"
	"reflect"
	"strings"
)

type Counter int64

const CounterTypeName = "counter"

type Gauge float64

const GaugeTypeName = "gauge"

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

	TotalMemory     Gauge
	FreeMemory      Gauge
	CPUutilization1 Gauge

	PollCount   Counter
	RandomValue Gauge
}

func ForEach(metricsP *RuntimeMetrics, metricHandler func(metricType, metricName, metricValue string)) {
	if metricsP == nil {
		return
	}

	v := reflect.ValueOf(*metricsP)

	for i := 0; i < v.NumField(); i++ {
		metricType := strings.ToLower(v.Field(i).Type().Name())
		metricName := v.Type().Field(i).Name
		metricValue := fmt.Sprint(v.Field(i))

		metricHandler(metricType, metricName, metricValue)
	}
}
