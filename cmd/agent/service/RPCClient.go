package service

import (
	"context"

	"github.com/GermanVor/devops-pet-project/cmd/agent/metric"
	"github.com/GermanVor/devops-pet-project/internal/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/GermanVor/devops-pet-project/proto"
)

type RPCClient struct {
	hashKey string
	c       pb.MetricsClient
	ctx     context.Context
}

func (s *RPCClient) SendMetrics(runtimeMetrics metric.RuntimeMetrics) {
	metricsArr := make([]*pb.Metric, 0)
	runtimeMetrics.ForEach(s.hashKey, func(metric *common.Metric) {
		metricsArr = append(metricsArr, pb.GetProtoMetric(metric))
	})

	s.c.AddMetrics(s.ctx, &pb.AddMetricsRequest{Metrics: metricsArr})
}

func (s *RPCClient) SendMetricsOneByOne(runtimeMetrics metric.RuntimeMetrics) {
	runtimeMetrics.ForEach(s.hashKey, func(metric *common.Metric) {
		s.c.AddMetric(s.ctx, &pb.AddMetricRequest{
			Metric: pb.GetProtoMetric(metric),
		})
	})
}

func InitRPCClient(config common.AgentConfig, ctx context.Context) (*RPCClient, error) {
	conn, err := grpc.Dial(config.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &RPCClient{
		hashKey: config.Key,
		c:       pb.NewMetricsClient(conn),
		ctx:     ctx,
	}, err
}
