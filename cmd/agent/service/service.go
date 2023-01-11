package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rsa"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/GermanVor/devops-pet-project/cmd/agent/metric"
	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/crypto"
	pb "github.com/GermanVor/devops-pet-project/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ClientInterface interface {
	SendMetrics(metrics metric.RuntimeMetrics)
	SendMetricsOneByOne(metrics metric.RuntimeMetrics)
}

type HTTPService struct {
	endpointURL string
	hashKey     string

	rsaKey *rsa.PublicKey
}

func (s *HTTPService) SendMetrics(runtimeMetrics metric.RuntimeMetrics) {
	metricsArr := []*common.Metric{}
	runtimeMetrics.ForEach(s.hashKey, func(metric *common.Metric) {
		metricsArr = append(metricsArr, metric)
	})

	metricsBytes, err := json.Marshal(&metricsArr)
	if err != nil {
		log.Println(err)
		return
	}

	if s.rsaKey != nil {
		var buf bytes.Buffer
		g := gzip.NewWriter(&buf)
		if _, err = g.Write(metricsBytes); err != nil {
			log.Println(err)
			return
		}
		if err = g.Close(); err != nil {
			log.Println(err)
			return
		}

		metricsBytes, err = crypto.RSAEncrypt(buf.Bytes(), s.rsaKey)
		if err != nil {
			log.Println(err)
			return
		}
	}

	url := s.endpointURL + "/updates/"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(metricsBytes))
	if err != nil {
		log.Println(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	if s.rsaKey != nil {
		req.Header.Set("Content-Encoding", "gzip")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return
	}

	log.Println(url, resp.Status)

	resp.Body.Close()
}

func (s *HTTPService) SendMetricsOneByOne(runtimeMetrics metric.RuntimeMetrics) {
	runtimeMetrics.ForEach(s.hashKey, func(metric *common.Metric) {
		metricBytes, err := metric.MarshalJSON()
		if err != nil {
			log.Println(err)
			return
		}

		if s.rsaKey != nil {
			metricBytes, _ = crypto.RSAEncrypt(metricBytes, s.rsaKey)
		}

		req, err := http.NewRequest(http.MethodPost, s.endpointURL+"/update/", bytes.NewBuffer(metricBytes))
		if err != nil {
			log.Println(err)
			return
		}

		if s.rsaKey != nil {
			req.Header.Set("Content-Encoding", "gzip")
		}

		req.Header.Add("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println(err)
			return
		}

		log.Println(resp)

		resp.Body.Close()
	})
}

type RPCService struct {
	hashKey string
	c       pb.MetricsClient
	ctx     context.Context
}

func (s *RPCService) SendMetrics(runtimeMetrics metric.RuntimeMetrics) {
	metricsArr := make([]*pb.Metric, 0)
	runtimeMetrics.ForEach(s.hashKey, func(metric *common.Metric) {
		metricsArr = append(metricsArr, pb.GetProtoMetric(metric))
	})

	s.c.AddMetrics(s.ctx, &pb.AddMetricsRequest{Metrics: metricsArr})
}

func (s *RPCService) SendMetricsOneByOne(runtimeMetrics metric.RuntimeMetrics) {
	runtimeMetrics.ForEach(s.hashKey, func(metric *common.Metric) {
		s.c.AddMetric(s.ctx, &pb.AddMetricRequest{
			Metric: pb.GetProtoMetric(metric),
		})
	})
}

type service struct {
	ctx            context.Context
	pollInterval   time.Duration
	reportInterval time.Duration
	client         ClientInterface
}

type ServiceType int64

const (
	HTTP ServiceType = 0
	GRPC
)

func InitService(config common.AgentConfig, ctx context.Context, serviceType ServiceType) *service {
	service := &service{
		ctx:            ctx,
		pollInterval:   config.PollInterval.Duration,
		reportInterval: config.ReportInterval.Duration,
	}

	switch serviceType {
	case 0:
		service.client = &HTTPService{
			endpointURL: "http://" + config.Address,
			rsaKey:      config.CryptoKey.PublicKey,
			hashKey:     config.Key,
		}
	case 1:
		conn, err := grpc.Dial(config.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatal(err)
		}

		service.client = &RPCService{
			hashKey: config.Key,
			c:       pb.NewMetricsClient(conn),
			ctx:     ctx,
		}
	default:
		log.Fatal()
	}

	return service
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
