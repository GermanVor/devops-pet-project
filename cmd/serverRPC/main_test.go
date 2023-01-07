package main_test

import (
	"context"
	"log"
	"net"
	"testing"

	main "github.com/GermanVor/devops-pet-project/cmd/serverRPC"
	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/storage"
	pb "github.com/GermanVor/devops-pet-project/proto"
	"github.com/bmizerany/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var stor, _ = storage.Init(nil)
var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()

	pb.RegisterMetricsServer(s, main.InitMetricsServer(stor))

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestSingleMetric(t *testing.T) {
	metricID := "qwerty"
	metricValue := float64(1)
	metricType := common.GaugeMetricName

	ctx := context.Background()

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := pb.NewMetricsClient(conn)

	t.Run("AddMetric", func(t *testing.T) {
		resp, err := client.AddMetric(ctx, &pb.AddMetricRequest{
			Metric: &pb.Metric{
				Id:    metricID,
				Value: metricValue,
				Type:  metricType,
			},
		})

		require.NoError(t, err)
		assert.Equal(t, (*pb.Error)(nil), resp.Error)
	})

	t.Run("GetMetric", func(t *testing.T) {
		resp, err := client.GetMetric(ctx, &pb.GetMetricRequest{
			Id:   metricID,
			Type: metricType,
		})

		require.NoError(t, err)

		assert.Equal(t, (*pb.Error)(nil), resp.Error)

		require.NotNil(t, resp.Metric)
		assert.Equal(t, metricID, resp.Metric.Id)
		assert.Equal(t, metricType, resp.Metric.Type)
		assert.Equal(t, metricValue, resp.Metric.Value)
	})
}

func TestMiltiplyMetrics(t *testing.T) {
	metrics := []*pb.Metric{
		{
			Id:    "ID-1",
			Type:  common.GaugeMetricName,
			Value: 55,
		},
		{
			Id:    "Id-2",
			Type:  common.CounterMetricName,
			Delta: 22,
		},
	}

	ctx := context.Background()

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := pb.NewMetricsClient(conn)

	t.Run("AddMetrics", func(t *testing.T) {
		resp, err := client.AddMetrics(ctx, &pb.AddMetricsRequest{
			Metrics: metrics,
		})

		require.NoError(t, err)
		assert.Equal(t, (*pb.Error)(nil), resp.Error)
	})

	t.Run("GetMetrics", func(t *testing.T) {
		resp, err := client.GetMetrics(ctx, &pb.GetMetricsRequest{})

		require.NoError(t, err)
		require.NotEqual(t, nil, resp.Metrics)

		for _, m := range metrics {
			f := false
			for _, rm := range resp.Metrics {
				t.Log(rm, m)

				if rm.Id == m.Id && rm.Type == m.Type && rm.Value == m.Value && rm.Delta == m.Delta {
					f = true
					break
				}
			}

			assert.Equal(t, true, f)
		}
	})
}
