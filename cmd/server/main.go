package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/GermanVor/devops-pet-project/cmd/server/handlers"
	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/storage"
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

	log.Println("Config is", Config)

	var initialFilePath *string

	if Config.IsRestore {
		initialFilePath = &Config.StoreFile
	}

	var currentStorage storage.StorageInterface
	stor, _ := storage.Init(initialFilePath)

	if Config.StoreFile != "" {
		if Config.StoreInterval == time.Duration(0) {
			currentStorage = storage.WithBackup(stor, Config.StoreFile)
		} else {
			stopBackupTicker := storage.InitBackupTicker(stor, Config.StoreFile, Config.StoreInterval)
			defer stopBackupTicker()

			currentStorage = stor
		}
	}

	r := chi.NewRouter()

	r.Use(middleware.Logger)

	handlers.InitRouter(r, currentStorage)

	log.Println("Server Started: http://" + Config.Address)

	log.Fatal(http.ListenAndServe(Config.Address, r))
}
