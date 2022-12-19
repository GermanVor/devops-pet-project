package main

import (
	"context"
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

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Compress(5, defaultCompressibleContentTypes...))

	if Config.CryptoKey.PrivateKey != nil {
		log.Println("Server will accept encrypted metrics (/updates/)")

		r.Use(handlers.MiddlewareEncryptBodyData(Config.CryptoKey.PrivateKey))
	}

	r.Use(handlers.MiddlewareDecompressGzip)

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
			if Config.StoreInterval.Duration == time.Duration(0) {
				currentStorage = storage.WithBackup(stor, Config.StoreFile)
			} else {
				stopBackupTicker := storage.InitBackupTicker(stor, Config.StoreFile, Config.StoreInterval.Duration)
				defer stopBackupTicker()
			}
		}
	}

	s := handlers.InitStorageWrapper(currentStorage, Config.Key)

	r.Route("/update", func(r chi.Router) {
		r.Post("/{mType}/{id}/{metricValue}", s.UpdateMetricV1)

		r.Post("/*", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotImplemented)
		})
		r.Post("/gauge/", handlers.MissedMetricNameHandlerFunc)
		r.Post("/counter/", handlers.MissedMetricNameHandlerFunc)
	})

	r.Get("/value/{mType}/{id}", s.GetMetricV1)

	r.Get("/", s.GetAllMetrics)
	
	r.Post("/update/", s.UpdateMetric)

	r.Post("/updates/", s.UpdateMetrics)

	r.Post("/value/", s.GetMetric)

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
