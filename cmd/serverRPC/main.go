package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/storage"
	pb "github.com/GermanVor/devops-pet-project/proto"
	"google.golang.org/grpc"
)

type MetricsServerImpl struct {
	pb.UnimplementedMetricsServer
	stor storage.StorageInterface
}

func InitMetricsServer(stor storage.StorageInterface) pb.MetricsServer {
	return &MetricsServerImpl{
		stor: stor,
	}
}

func (s *MetricsServerImpl) AddMetric(ctx context.Context, in *pb.AddMetricRequest) (*pb.AddMetricResponse, error) {
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

func (s *MetricsServerImpl) GetMetric(ctx context.Context, in *pb.GetMetricRequest) (*pb.GetMetricResponse, error) {
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

func (s *MetricsServerImpl) AddMetrics(ctx context.Context, in *pb.AddMetricsRequest) (*pb.AddMetricsResponse, error) {
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

func (s *MetricsServerImpl) GetMetrics(ctx context.Context, in *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
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

var Config = &common.ServerConfig{
	Address:       "localhost:8080",
	StoreInterval: common.Duration{Duration: 300 * time.Second},
	StoreFile:     "/tmp/devops-metrics-db.json",
	IsRestore:     true,
}

func initConfig() {
	common.InitJSONConfig(Config)
	common.InitServerFlagConfig(Config)
	flag.Parse()

	common.InitServerEnvConfig(Config)
}

func main() {
	initConfig()

	var currentStor storage.StorageInterface
	if Config.DataBaseDSN != "" {
		dbContext := context.Background()
		sqlStorage, err := storage.InitV2(dbContext, Config.DataBaseDSN)

		if err != nil {
			log.Fatalf(err.Error())
		}
		defer sqlStorage.Close()

		currentStor = sqlStorage
	} else {
		var initialFilePath *string
		if Config.IsRestore && Config.StoreFile != "" {
			initialFilePath = &Config.StoreFile
		}

		stor, _ := storage.Init(initialFilePath)
		currentStor = stor
	}

	listen, err := net.Listen("tcp", Config.Address)
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	pb.RegisterMetricsServer(s, InitMetricsServer(currentStor))

	fmt.Println("Server gRPC started")

	if err := s.Serve(listen); err != nil {
		log.Fatal(err)
	}
}
