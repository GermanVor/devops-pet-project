package service

import (
	"context"
	"log"
	"time"

	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/storage"
)

type ServiceInterface interface {
	Start() error
}

type service struct {
	server     ServiceInterface
	destructor func()
}

func InitService(config *common.ServerConfig, ctx context.Context, serviceType common.ServiceType) *service {
	service := &service{}

	var currentStor storage.StorageInterface
	if config.DataBaseDSN != "" {
		dbContext := context.Background()
		sqlStorage, err := storage.InitV2(dbContext, config.DataBaseDSN)

		if err != nil {
			log.Fatalf(err.Error())
		}

		currentStor = sqlStorage
		service.destructor = sqlStorage.Close
	} else {
		var initialFilePath *string
		if config.IsRestore && config.StoreFile != "" {
			initialFilePath = &config.StoreFile
		}

		stor, _ := storage.Init(initialFilePath)
		currentStor = stor

		if config.StoreFile != "" {
			if config.StoreInterval.Duration == time.Duration(0) {
				currentStor = storage.WithBackup(stor, config.StoreFile)
			} else {
				log.Println("Server works with InitBackupTicker", config.StoreFile, config.StoreInterval.Duration)
				service.destructor = storage.InitBackupTicker(stor, config.StoreFile, config.StoreInterval.Duration)
			}
		}
	}

	switch serviceType {
	case common.HTTP:
		service.server = InitHTTPServer(config, ctx, currentStor)
	case common.GRPC:
		service.server = InitRPCServer(config, ctx, currentStor)
	default:
		log.Fatal()
	}

	return service
}

func (s *service) Start() error {
	return s.server.Start()
}

func (s *service) Destructor() {
	s.destructor()
}
