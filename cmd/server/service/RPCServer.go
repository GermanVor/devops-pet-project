package service

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/storage"
	pb "github.com/GermanVor/devops-pet-project/proto"
	"google.golang.org/grpc"
)

type RPCImpl struct {
	pb.UnimplementedMetricsServer
	stor storage.StorageInterface
}

func InitRPCImpl(stor storage.StorageInterface) *RPCImpl {
	return &RPCImpl{
		stor: stor,
	}
}

type RPCServer struct {
	address string
	server  *grpc.Server
	impl    *RPCImpl
}

func (s *RPCImpl) AddMetric(ctx context.Context, in *pb.AddMetricRequest) (*pb.AddMetricResponse, error) {
	resp := &pb.AddMetricResponse{}

	if in.Metric == nil {
		resp.Error = &pb.Error{
			Code:    http.StatusBadRequest,
			Message: "empty metric object",
		}

		return resp, nil
	}

	err := s.stor.UpdateMetric(ctx, *in.Metric.GetRequestMetric())

	if err != nil {
		resp.Error = &pb.Error{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}
	}

	return resp, nil
}

func (s *RPCImpl) GetMetric(ctx context.Context, in *pb.GetMetricRequest) (*pb.GetMetricResponse, error) {
	resp := &pb.GetMetricResponse{}

	switch in.Type {
	case common.GaugeMetricName:
	case common.CounterMetricName:
	default:
		resp.Error = &pb.Error{
			Code:    http.StatusNotFound,
			Message: "unknown metric type",
		}

		return resp, nil
	}

	storageMetric, err := s.stor.GetMetric(ctx, in.Type, in.Id)

	if err != nil {
		resp.Error = &pb.Error{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}

		return resp, nil
	}

	if storageMetric == nil {
		resp.Error = &pb.Error{
			Code:    http.StatusInternalServerError,
			Message: "",
		}

		return resp, nil
	}

	resp.Metric = pb.GetProtoStorageMetric(storageMetric)

	return resp, nil
}

func (s *RPCImpl) AddMetrics(ctx context.Context, in *pb.AddMetricsRequest) (*pb.AddMetricsResponse, error) {
	resp := &pb.AddMetricsResponse{}

	metricsList := make([]common.Metric, 0)
	for _, m := range in.Metrics {
		metricsList = append(metricsList, *m.GetRequestMetric())
	}

	err := s.stor.UpdateMetrics(ctx, metricsList)

	if err != nil {
		resp.Error = &pb.Error{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}
	}

	return resp, nil
}

func (s *RPCImpl) GetMetrics(ctx context.Context, in *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	resp := &pb.GetMetricsResponse{
		Metrics: make([]*pb.Metric, 0),
	}

	err := s.stor.ForEachMetrics(ctx, func(sm *storage.StorageMetric) {
		resp.Metrics = append(resp.Metrics, pb.GetProtoStorageMetric(sm))
	})

	if err != nil {
		return &pb.GetMetricsResponse{
			Error: &pb.Error{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			},
		}, nil
	}

	return resp, nil
}

func (s *RPCImpl) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingResponse, error) {
	err := s.stor.Ping(ctx)

	return &pb.PingResponse{Status: err == nil}, nil
}

func (s *RPCServer) Start() error {
	listen, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	pb.RegisterMetricsServer(s.server, s.impl)

	fmt.Println("Server gRPC started")

	return s.server.Serve(listen)
}

func InitRPCServer(config *common.ServerConfig, ctx context.Context, stor storage.StorageInterface) *RPCServer {
	s := &RPCServer{
		address: config.Address,
		server:  grpc.NewServer(),
		impl:    &RPCImpl{stor: stor},
	}

	return s
}
