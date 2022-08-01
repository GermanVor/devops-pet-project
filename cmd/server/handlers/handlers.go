package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/GermanVor/devops-pet-project/storage"
)

func UpdateStorageFunc(w http.ResponseWriter, r *http.Request, storage *storage.Storage) {
	w.Header().Add("Content-Type", "application/json")

	urlParts := strings.Split(r.URL.Path[1:], "/")
	if len(urlParts) < 1 || urlParts[0] != "update" {
		w.WriteHeader(http.StatusNotFound)
		w.Write(nil)
		return
	}

	metricType := urlParts[1]
	if len(urlParts) < 2 || (metricType != "gauge" && metricType != "counter") {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write(nil)
		return
	}

	if len(urlParts) < 3 || urlParts[2] == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write(nil)
		return
	}

	if len(urlParts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(nil)
		return
	}

	metricName := urlParts[2]
	metricValue := urlParts[3]

	if metricType == "gauge" {
		gaugeMetricValue, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(nil)
			return
		}

		storage.SetGaugeMetric(metricName, gaugeMetricValue)
		w.WriteHeader(http.StatusOK)
		w.Write(nil)
	} else {
		counterMetricValue, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(nil)
			return
		}

		storage.IncreaseCounterMetric(metricName, counterMetricValue)

		w.WriteHeader(http.StatusOK)
		w.Write(nil)
	}
}
