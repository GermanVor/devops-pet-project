package handlers_test

import (
	"net/http"

	"github.com/GermanVor/devops-pet-project/cmd/server/handlers"
	"github.com/GermanVor/devops-pet-project/internal/storage"
)

func ExampleUpdateMetric() {
	storMock, _ := storage.Init(nil)

	http.HandleFunc("/update/{mType}/{id}/{metricValue}", func(w http.ResponseWriter, r *http.Request) {
		handlers.UpdateMetric(w, r, storMock, "")
	})
}

func ExampleGetMetric() {
	storMock, _ := storage.Init(nil)

	http.HandleFunc("/update/{mType}/{id}/{metricValue}", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetMetric(w, r, storMock, "")
	})
}
