package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/GermanVor/devops-pet-project/cmd/server/service"
	"github.com/GermanVor/devops-pet-project/internal/common"

	_ "net/http/pprof"
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
	log.Println("Config is", Config)

	s, err := service.InitService(Config, context.Background(), common.HTTP)
	if err != nil {
		log.Fatalln(err.Error())
	}

	defer s.Destructor()
	s.Start()
}
