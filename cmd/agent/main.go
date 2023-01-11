package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GermanVor/devops-pet-project/cmd/agent/service"
	"github.com/GermanVor/devops-pet-project/internal/common"
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

var Config = &common.AgentConfig{
	Address:        "localhost:8080",
	PollInterval:   common.Duration{Duration: time.Second},
	ReportInterval: common.Duration{Duration: 2 * time.Second},
}

func initConfig() {
	common.InitJSONConfig(Config)
	common.InitAgentFlagConfig(Config)
	flag.Parse()

	common.InitAgentEnvConfig(Config)
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

	service := service.InitService(*Config, ctx, service.HTTP)
	service.StartSending()

	log.Println("Agent finished work")
}
