package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rsa"
	"encoding/json"
	"log"
	"net/http"

	"github.com/GermanVor/devops-pet-project/cmd/agent/metric"
	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/crypto"
)

type HTTPClient struct {
	endpointURL string
	hashKey     string

	rsaKey *rsa.PublicKey
}

func (s *HTTPClient) SendMetrics(runtimeMetrics metric.RuntimeMetrics) {
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

func (s *HTTPClient) SendMetricsOneByOne(runtimeMetrics metric.RuntimeMetrics) {
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

func InitHTTPClient(config common.AgentConfig, ctx context.Context) *HTTPClient {
	return &HTTPClient{
		endpointURL: "http://" + config.Address,
		rsaKey:      config.CryptoKey.PublicKey,
		hashKey:     config.Key,
	}
}
