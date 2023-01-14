package service

import (
	"context"
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
)

var defaultCompressibleContentTypes = []string{
	"application/javascript",
	"application/json",
	"text/css",
	"text/html",
	"text/plain",
	"text/xml",
}

type HTTPServer struct {
	address     string
	ctx         context.Context
	r           *chi.Mux
	storWrapper *handlers.StorageWrapper
}

func (s *HTTPServer) Start() error {
	baseContext, shutDownRequests := context.WithCancel(s.ctx)
	server := http.Server{
		Addr:    s.address,
		Handler: s.r,
		BaseContext: func(l net.Listener) context.Context {
			return baseContext
		},
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sigs
		log.Println("Server is shutting down...")

		ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		shutDownRequests()
	}()

	log.Println("Server Started: http://" + s.address)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", s.address, err)
	}

	log.Println("Server finished work")

	return nil
}

func InitHTTPServer(config *common.ServerConfig, ctx context.Context, stor storage.StorageInterface) *HTTPServer {
	s := &HTTPServer{
		address:     config.Address,
		ctx:         ctx,
		r:           chi.NewRouter(),
		storWrapper: handlers.InitStorageWrapper(stor, config.Key),
	}

	s.r.Use(middleware.Logger)
	s.r.Use(middleware.Compress(5, defaultCompressibleContentTypes...))

	if config.TrustedSubnet != "" {
		log.Printf(
			"Server accepts metrics only with %s equal %s\n",
			handlers.TrustedSubnetHeader,
			config.TrustedSubnet,
		)

		s.r.Use(handlers.MiddlewareTrustedSubnet(config.TrustedSubnet))
	}

	if config.CryptoKey.PrivateKey != nil {
		log.Println("Server accepts encrypted metrics (/updates/)")

		s.r.Use(handlers.MiddlewareEncryptBodyData(config.CryptoKey.PrivateKey))
	}

	s.r.Use(handlers.MiddlewareDecompressGzip)

	if config.DataBaseDSN != "" {
		s.r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			if stor.Ping(r.Context()) == nil {
				w.WriteHeader(http.StatusOK)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
		})
	}

	s.r.Route("/update", func(r chi.Router) {
		r.Post("/{mType}/{id}/{metricValue}", s.storWrapper.UpdateMetricV1)

		r.Post("/*", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotImplemented)
		})
		r.Post("/gauge/", handlers.MissedMetricNameHandlerFunc)
		r.Post("/counter/", handlers.MissedMetricNameHandlerFunc)
	})

	s.r.Get("/value/{mType}/{id}", s.storWrapper.GetMetricV1)

	s.r.Get("/", s.storWrapper.GetAllMetrics)

	s.r.Post("/update/", s.storWrapper.UpdateMetric)

	s.r.Post("/updates/", s.storWrapper.UpdateMetrics)

	s.r.Post("/value/", s.storWrapper.GetMetric)

	return s
}
