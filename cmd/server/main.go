package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/GermanVor/devops-pet-project/cmd/server/handlers"
	"github.com/GermanVor/devops-pet-project/common"
	"github.com/GermanVor/devops-pet-project/storage"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

var Config = &common.ServerConfig{
	Address:       "localhost:8080",
	StoreInterval: 300 * time.Second,
	StoreFile:     "/tmp/devops-metrics-db.json",
	IsRestore:     true,
}

func main() {
	common.InitServerFlagConfig(Config)
	flag.Parse()
	common.InitServerEnvConfig(Config)

	fmt.Println("Config is", Config)

	initOptions := &storage.InitOptions{}

	if Config.StoreFile != "" {
		backupFilePath := Config.StoreFile
		initOptions.BackupFilePath = &backupFilePath
		initOptions.BackupInterval = Config.StoreInterval
	}

	if Config.IsRestore {
		initialFilePath := Config.StoreFile
		initOptions.InitialFilePath = &initialFilePath
	}

	currentStorage, _, destructor := storage.Init(initOptions)
	defer destructor()

	r := chi.NewRouter()

	r.Use(middleware.Logger)

	handlers.InitRouter(r, currentStorage)

	fmt.Println("Server Started: http://" + Config.Address)

	log.Fatal(http.ListenAndServe(Config.Address, r))
}
