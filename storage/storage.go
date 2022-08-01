package storage

import (
	"sync"
)

type Storage struct {
	gaugeMap    map[string]float64
	gaugeMapRWM *sync.Mutex

	counterMap    map[string]int64
	counterMapRWM *sync.Mutex
}

func Init() *Storage {
	return &Storage{
		gaugeMap:    make(map[string]float64),
		gaugeMapRWM: &sync.Mutex{},

		counterMap:    make(map[string]int64),
		counterMapRWM: &sync.Mutex{},
	}
}

func (stor *Storage) SetGaugeMetric(metricName string, value float64) {
	stor.gaugeMapRWM.Lock()
	defer stor.gaugeMapRWM.Unlock()

	stor.gaugeMap[metricName] = value
}

func (stor *Storage) GetGaugeMetric(metricName string) float64 {
	stor.gaugeMapRWM.Lock()
	defer stor.gaugeMapRWM.Unlock()

	return stor.gaugeMap[metricName]
}

func (stor *Storage) GetCounterMetric(metricName string) int64 {
	stor.counterMapRWM.Lock()
	defer stor.counterMapRWM.Unlock()

	return stor.counterMap[metricName]
}

func (stor *Storage) IncreaseCounterMetric(metricName string, count int64) {
	stor.counterMapRWM.Lock()
	defer stor.counterMapRWM.Unlock()

	stor.counterMap[metricName] += count
}
