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

func (stor *Storage) GetGaugeMetric(metricName string) (float64, bool) {
	stor.gaugeMapRWM.Lock()
	defer stor.gaugeMapRWM.Unlock()

	value, ok := stor.gaugeMap[metricName]
	return value, ok
}

func (stor *Storage) ForEachGaugeMetric(handler func(metricName string, value float64)) {
	for a, b := range stor.gaugeMap {
		handler(a, b)
	}
}

func (stor *Storage) GetCounterMetric(metricName string) (int64, bool) {
	stor.counterMapRWM.Lock()
	defer stor.counterMapRWM.Unlock()

	value, ok := stor.counterMap[metricName]
	return value, ok
}

func (stor *Storage) ForEachCounterMetric(handler func(metricName string, value int64)) {
	for a, b := range stor.counterMap {
		handler(a, b)
	}
}

func (stor *Storage) IncreaseCounterMetric(metricName string, count int64) {
	stor.counterMapRWM.Lock()
	defer stor.counterMapRWM.Unlock()

	stor.counterMap[metricName] += count
}
