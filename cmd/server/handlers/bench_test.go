package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	handlers "github.com/GermanVor/devops-pet-project/cmd/server/handlers"
	"github.com/GermanVor/devops-pet-project/internal/common"
	storage "github.com/GermanVor/devops-pet-project/internal/storage"
)

func BenchmarkGetAllMetrics(b *testing.B) {
	stor := &storage.MockStorage{
		ForEachMetricsArr: []*storage.StorageMetric{
			{ID: "qwe1", MType: common.GaugeMetricName, Value: 1},
			{ID: "qwe2", MType: common.GaugeMetricName, Value: 1},
			{ID: "qwe3", MType: common.GaugeMetricName, Value: 1},
			{ID: "qwe4", MType: common.GaugeMetricName, Value: 1},
			{ID: "qwe5", MType: common.GaugeMetricName, Value: 1},
		},
	}

	req, err := http.NewRequest("GET", "/get-metrics", nil)
	if err != nil {
		b.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.GetAllMetrics(w, r, stor)
	})

	handler.ServeHTTP(rr, req)
}
