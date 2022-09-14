package main

import (
	"context"
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

var defaultCompressibleContentTypes = []string{
	"application/javascript",
	"application/json",
	"text/css",
	"text/html",
	"text/plain",
	"text/xml",
}

func initConfig() {
	common.InitServerFlagConfig(Config)
	flag.Parse()
	common.InitServerEnvConfig(Config)

	log.Println("Config is", Config)
}

func main() {
	initConfig()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Compress(5, defaultCompressibleContentTypes...))

	var currentStorage storage.StorageInterface

	if Config.DataBaseDSN != "" {
		dbContext := context.Background()
		sqlStorage, err := storage.InitV2(dbContext, Config.DataBaseDSN)

		if err != nil {
			log.Fatalf(err.Error())
		}
		defer sqlStorage.Close()

		currentStorage = sqlStorage
		conn := sqlStorage.GetConn()

		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			if conn != nil {
				if conn.Ping(r.Context()) == nil {
					w.WriteHeader(http.StatusOK)
					return
				}
			}

			w.WriteHeader(http.StatusInternalServerError)
		})
	} else {
		var initialFilePath *string
		if Config.IsRestore && Config.StoreFile != "" {
			initialFilePath = &Config.StoreFile
		}

		stor, _ := storage.Init(initialFilePath)
		currentStorage = stor

		if Config.StoreFile != "" {
			if Config.StoreInterval == time.Duration(0) {
				currentStorage = storage.WithBackup(stor, Config.StoreFile)
			} else {
				stopBackupTicker := storage.InitBackupTicker(stor, Config.StoreFile, Config.StoreInterval)
				defer stopBackupTicker()
			}
		}
	}

	handlers.InitRouter(r, currentStorage, Config.Key)

	log.Println("Server Started: http://" + Config.Address)

	log.Fatal(http.ListenAndServe(Config.Address, r))
}
