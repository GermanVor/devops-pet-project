package proto

import (
	"log"

	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/storage"
)

func (protoMetric *Metric) Equal(protoMetricA *Metric) bool {
	if protoMetric.Id != protoMetricA.Id {
		return false
	}

	switch mA := protoMetric.Spec.(type) {
	case *Metric_Counter:
		{
			mB, ok := protoMetricA.Spec.(*Metric_Counter)

			if !ok || mA.Counter.Delta != mB.Counter.Delta {
				return false
			}
		}
	case *Metric_Gauge:
		{
			mB, ok := protoMetricA.Spec.(*Metric_Gauge)

			if !ok || mA.Gauge.Value != mB.Gauge.Value {
				return false
			}
		}
	default:
		return false
	}

	return true
}

func (protoMetric *Metric) GetRequestMetric() *common.Metrics {
	metric := &common.Metrics{
		ID:   protoMetric.Id,
		Hash: protoMetric.Hash,
	}

	switch m := protoMetric.Spec.(type) {
	case *Metric_Counter:
		{
			metric.Delta = &m.Counter.Delta
			metric.MType = common.CounterMetricName
		}
	case *Metric_Gauge:
		{
			metric.Value = &m.Gauge.Value
			metric.MType = common.GaugeMetricName
		}
	}

	return metric
}

func GetProtoStorageMetric(storageMetric *storage.StorageMetric) *Metric {
	protoMetric := &Metric{
		Id: storageMetric.ID,
	}

	switch storageMetric.MType {
	case common.GaugeMetricName:
		protoMetric.Spec = &Metric_Gauge{Gauge: &GaugeMetric{Value: storageMetric.Value}}
	case common.CounterMetricName:
		protoMetric.Spec = &Metric_Counter{Counter: &CounterMetric{Delta: storageMetric.Delta}}
	default:
		log.Fatal()
	}

	return protoMetric
}

func GetProtoMetric(metric *common.Metrics) *Metric {
	protoMetric := &Metric{
		Id: metric.ID,
	}

	switch metric.MType {
	case common.GaugeMetricName:
		if metric.Value == nil {
			protoMetric.Spec = &Metric_Gauge{Gauge: &GaugeMetric{Value: 0}}
		} else {
			protoMetric.Spec = &Metric_Gauge{Gauge: &GaugeMetric{Value: *metric.Value}}
		}
	case common.CounterMetricName:
		if metric.Delta == nil {
			protoMetric.Spec = &Metric_Counter{Counter: &CounterMetric{Delta: *metric.Delta}}
		} else {
			protoMetric.Spec = &Metric_Counter{Counter: &CounterMetric{Delta: 0}}
		}
	default:
		log.Fatal()
	}

	return protoMetric
}
