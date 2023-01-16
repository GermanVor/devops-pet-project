package service

import (
	"context"
	"sync"
	"time"

	"github.com/GermanVor/devops-pet-project/cmd/agent/metric"
	"github.com/GermanVor/devops-pet-project/internal/common"
)

type ClientInterface interface {
	SendMetrics(metrics metric.RuntimeMetrics)
	SendMetricsOneByOne(metrics metric.RuntimeMetrics)
}

type service struct {
	ctx            context.Context
	pollInterval   time.Duration
	reportInterval time.Duration
	client         ClientInterface
}

func (s *service) StartSending() {
	pollTicker := time.NewTicker(s.pollInterval)
	defer pollTicker.Stop()

	reportInterval := time.NewTicker(s.reportInterval)
	defer reportInterval.Stop()

	var mPointer *metric.RuntimeMetrics
	pollCount := metric.Counter(0)

	mux := sync.Mutex{}

	mainWG := sync.WaitGroup{}
	mainWG.Add(2)

	go func() {
		for {
			select {
			case <-pollTicker.C:
				metricsPointer := &metric.RuntimeMetrics{}

				wg := sync.WaitGroup{}
				wg.Add(2)

				go func() {
					defer wg.Done()
					metricsPointer.CollectMetrics()
				}()
				go func() {
					defer wg.Done()
					metricsPointer.CollectGopsutilMetrics()
				}()

				wg.Wait()

				mux.Lock()

				pollCount++
				metricsPointer.PollCount = pollCount
				mPointer = metricsPointer

				mux.Unlock()
			case <-s.ctx.Done():
				mainWG.Done()
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
				pollCount = 0

				mux.Unlock()

				// s.client.SendMetricsOneByOne(metricsCopy)
				s.client.SendMetrics(metricsCopy)

			case <-s.ctx.Done():
				mainWG.Done()
				return
			}
		}
	}()

	mainWG.Wait()
}

func InitService(config common.AgentConfig, ctx context.Context, serviceType common.ServiceType) (*service, error) {
	service := &service{
		ctx:            ctx,
		pollInterval:   config.PollInterval.Duration,
		reportInterval: config.ReportInterval.Duration,
	}

	var err error

	switch serviceType {
	case common.HTTP:
		service.client = InitHTTPClient(config, ctx)
	case common.GRPC:
		service.client, err = InitRPCClient(config, ctx)
	default:
		return nil, common.ErrUnknownServiceType
	}

	return service, err
}
