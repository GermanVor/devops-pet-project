package handlers_test

import (
	"net/http"

	"github.com/GermanVor/devops-pet-project/cmd/server/handlers"
	"github.com/GermanVor/devops-pet-project/internal/storage"
)

func ExampleStorageWrapper_UpdateMetric() {
	storMock, _ := storage.Init(nil)
	s := handlers.InitStorageWrapper(storMock, "")

	http.HandleFunc("/update/{mType}/{id}/{metricValue}", s.UpdateMetric)
}

func ExampleStorageWrapper_GetMetric() {
	storMock, _ := storage.Init(nil)
	s := handlers.InitStorageWrapper(storMock, "")

	http.HandleFunc("/update/{mType}/{id}/{metricValue}", s.GetMetric)
}
