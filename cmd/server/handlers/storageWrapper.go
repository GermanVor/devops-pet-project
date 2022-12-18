package handlers

import (
	"github.com/GermanVor/devops-pet-project/internal/storage"
)

type StorageWrapper struct {
	stor storage.StorageInterface
	key  string
}

func InitStorageWrapper(stor storage.StorageInterface, key string) *StorageWrapper {
	return &StorageWrapper{
		stor,
		key,
	}
}
