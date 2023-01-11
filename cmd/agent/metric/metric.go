package metric

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
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

func (m *RuntimeMetrics) ForEach(key string, metricHandler func(metric *common.Metric)) {
	if m == nil {
		return
	}

	v := reflect.ValueOf(*m)

	for i := 0; i < v.NumField(); i++ {
		metricType := strings.ToLower(v.Field(i).Type().Name())
		metricName := v.Type().Field(i).Name
		metricValue := fmt.Sprint(v.Field(i))

		metric := &common.Metric{
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
			metric.SetHash(key)
		}

		metricHandler(metric)
	}
}

func (m *RuntimeMetrics) CollectMetrics() {
	rtm := runtime.MemStats{}
	runtime.ReadMemStats(&rtm)

	m.Alloc = Gauge(rtm.Alloc)
	m.BuckHashSys = Gauge(rtm.BuckHashSys)
	m.Frees = Gauge(rtm.Frees)
	m.GCCPUFraction = Gauge(rtm.GCCPUFraction)
	m.GCSys = Gauge(rtm.GCSys)
	m.HeapAlloc = Gauge(rtm.HeapAlloc)
	m.HeapIdle = Gauge(rtm.HeapIdle)
	m.HeapInuse = Gauge(rtm.HeapInuse)
	m.HeapObjects = Gauge(rtm.HeapObjects)
	m.HeapReleased = Gauge(rtm.HeapReleased)
	m.HeapSys = Gauge(rtm.HeapSys)
	m.LastGC = Gauge(rtm.LastGC)
	m.Lookups = Gauge(rtm.Lookups)
	m.MCacheInuse = Gauge(rtm.MCacheInuse)
	m.MCacheSys = Gauge(rtm.MCacheSys)
	m.MSpanInuse = Gauge(rtm.MSpanInuse)
	m.MSpanSys = Gauge(rtm.MSpanSys)
	m.Mallocs = Gauge(rtm.Mallocs)
	m.NextGC = Gauge(rtm.NextGC)
	m.NumForcedGC = Gauge(rtm.NumForcedGC)
	m.NumGC = Gauge(rtm.NumGC)
	m.OtherSys = Gauge(rtm.OtherSys)
	m.PauseTotalNs = Gauge(rtm.PauseTotalNs)
	m.StackInuse = Gauge(rtm.StackInuse)
	m.StackSys = Gauge(rtm.StackSys)
	m.Sys = Gauge(rtm.Sys)
	m.TotalAlloc = Gauge(rtm.TotalAlloc)

	m.RandomValue = Gauge(rand.Float64())
}

func (m *RuntimeMetrics) CollectGopsutilMetrics() {
	v, _ := mem.VirtualMemory()
	m.TotalMemory = Gauge(v.Total)
	m.FreeMemory = Gauge(v.Free)

	count := Gauge(0)
	a, _ := cpu.Percent(0, true)
	for _, percent := range a {
		if percent != 0 {
			count++
		}
	}

	m.CPUutilization1 = count
}
