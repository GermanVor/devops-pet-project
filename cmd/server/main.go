package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GermanVor/devops-pet-project/cmd/server/handlers"
	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/storage"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"

	_ "net/http/pprof"
)

var defaultCompressibleContentTypes = []string{
	"application/javascript",
	"application/json",
	"text/css",
	"text/html",
	"text/plain",
	"text/xml",
}

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
	StoreInterval: "300s",
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

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

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

		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			if sqlStorage.Ping(r.Context()) == nil {
				w.WriteHeader(http.StatusOK)
				return
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
			if Config.StoreInterval == "" {
				currentStorage = storage.WithBackup(stor, Config.StoreFile)
			} else {
				interval, _ := time.ParseDuration(Config.StoreInterval)

				stopBackupTicker := storage.InitBackupTicker(stor, Config.StoreFile, interval)
				defer stopBackupTicker()
			}
		}
	}

	var rsaKey *rsa.PrivateKey
	if Config.CryptoKey != "" {
		ketData, _ := os.ReadFile(Config.CryptoKey)
		block, _ := pem.Decode(ketData)
		rsaKey, _ = x509.ParsePKCS1PrivateKey(block.Bytes)

		log.Println("Server will accept encrypted metrics (/updates/)")
	}

	handlers.InitRouterV1(r, currentStorage)
	handlers.InitRouter(r, currentStorage, Config.Key, rsaKey)

	baseContext, shutDownRequests := context.WithCancel(context.Background())
	server := http.Server{
		Addr:    Config.Address,
		Handler: r,
		BaseContext: func(l net.Listener) context.Context {
			return baseContext
		},
	}

	go func() {
		<-sigs
		log.Println("Server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		shutDownRequests()
	}()

	log.Println("Server Started: http://" + Config.Address)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", Config.Address, err)
	}

	log.Println("Server finished work")
}
