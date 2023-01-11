package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/GermanVor/devops-pet-project/cmd/agent/metric"
	"github.com/GermanVor/devops-pet-project/cmd/agent/utils"
	"github.com/GermanVor/devops-pet-project/internal/common"
	pb "github.com/GermanVor/devops-pet-project/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func init() {
	fmt.Printf("Build version:\t%s\n", buildVersion)
	fmt.Printf("Build date:\t%s\n", buildDate)
	fmt.Printf("Build commit:\t%s\n", buildCommit)
}

func Start(ctx context.Context, target string) {
	pollTicker := time.NewTicker(Config.PollInterval.Duration)
	defer pollTicker.Stop()

	reportInterval := time.NewTicker(Config.ReportInterval.Duration)
	defer reportInterval.Stop()

	var mPointer *metric.RuntimeMetrics
	pollCount := metric.Counter(0)

	mux := sync.Mutex{}

	mainWG := sync.WaitGroup{}
	mainWG.Add(2)

	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	c := pb.NewMetricsClient(conn)

	go func() {
		for {
			select {
			case <-pollTicker.C:
				metricsPointer := &metric.RuntimeMetrics{}

				wg := sync.WaitGroup{}
				wg.Add(2)

				go func() {
					defer wg.Done()
					utils.CollectMetrics(metricsPointer)
				}()
				go func() {
					defer wg.Done()
					utils.CollectGopsutilMetrics(metricsPointer)
				}()

				wg.Wait()

				mux.Lock()

				pollCount++
				metricsPointer.PollCount = pollCount
				mPointer = metricsPointer

				mux.Unlock()
			case <-ctx.Done():
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

				metric.ForEach(&metricsCopy, func(metricType, metricName, metricValue string) {
					m := &pb.Metric{
						Id: metricName,
					}

					switch metricType {
					case common.GaugeMetricName:
						value, err := strconv.ParseFloat(metricValue, 64)
						if err != nil {
							return
						}

						m.Spec = &pb.Metric_Gauge{Gauge: &pb.GaugeMetric{Value: value}}
					case common.CounterMetricName:
						delta, err := strconv.ParseInt(metricValue, 10, 64)
						if err != nil {
							return
						}

						m.Spec = &pb.Metric_Counter{Counter: &pb.CounterMetric{Delta: delta}}
					}

					c.AddMetric(ctx, &pb.AddMetricRequest{
						Metric: m,
					})
				})
			case <-ctx.Done():
				mainWG.Done()
				return
			}
		}
	}()

	mainWG.Wait()
}

func initConfig() {
	common.InitJSONConfig(Config)
	common.InitAgentFlagConfig(Config)
	flag.Parse()

	common.InitAgentEnvConfig(Config)
}

var Config = &common.AgentConfig{
	Address:        "localhost:8080",
	PollInterval:   common.Duration{Duration: time.Second},
	ReportInterval: common.Duration{Duration: 2 * time.Second},
}

func main() {
	initConfig()
	log.Println("Agent Config", Config)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-sigs
		cancel()
	}()

	Start(ctx, Config.Address)
	log.Println("Agent finished work")
}
