package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/storage"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/jackc/pgx/v4/pgxpool"
)

func UpdateGaugeMetric(w http.ResponseWriter, r *http.Request, currentStorage storage.StorageInterface) {
	w.Header().Add("Content-Type", "application/json")

	metricName := chi.URLParam(r, "metricName")
	metricValue, err := strconv.ParseFloat(chi.URLParam(r, "metricValue"), 64)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	currentStorage.SetGaugeMetric(metricName, metricValue)

	w.WriteHeader(http.StatusOK)
}

func UpdateCounterMetric(w http.ResponseWriter, r *http.Request, currentStorage storage.StorageInterface) {
	w.Header().Add("Content-Type", "application/json")

	metricName := chi.URLParam(r, "metricName")
	metricValue, err := strconv.ParseInt(chi.URLParam(r, "metricValue"), 10, 64)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	currentStorage.SetCounterMetric(metricName, metricValue)

	w.WriteHeader(http.StatusOK)
}

func UpdateMetric(w http.ResponseWriter, r *http.Request, currentStorage storage.StorageInterface, key string) {
	metric := &common.Metrics{}

	if err := json.NewDecoder(r.Body).Decode(metric); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if key != "" {
		metricHash, err := common.GetMetricHash(metric, key)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if metricHash != metric.Hash {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	switch metric.MType {
	case common.GaugeMetricName:
		currentStorage.SetGaugeMetric(metric.ID, *metric.Value)

	case common.CounterMetricName:
		currentStorage.SetCounterMetric(metric.ID, *metric.Delta)

	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func GetGaugeMetric(w http.ResponseWriter, r *http.Request, currentStorage storage.StorageInterface) {
	w.Header().Add("Content-Type", "text/plain")

	metricName := chi.URLParam(r, "metricName")
	value, ok := currentStorage.GetGaugeMetric(metricName)

	if ok {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprint(value)))
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func GetCounterMetric(w http.ResponseWriter, r *http.Request, currentStorage storage.StorageInterface) {
	w.Header().Add("Content-Type", "text/plain")

	metricName := chi.URLParam(r, "metricName")
	value, ok := currentStorage.GetCounterMetric(metricName)

	if ok {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprint(value)))
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func missedMetricNameHandlerFunc(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func GetMetric(w http.ResponseWriter, r *http.Request, currentStorage storage.StorageInterface, key string) {
	metric := &common.Metrics{}

	if err := json.NewDecoder(r.Body).Decode(metric); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	printMetric := func() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if key != "" {
			metric.Hash, _ = common.GetMetricHash(metric, key)
		}

		jsonResp, err := metric.MarshalJSON()
		if err != nil {
			log.Fatalf("Error happened in JSON marshal. Err: %s", err)
		}
		w.Write(jsonResp)
	}

	switch metric.MType {
	case common.GaugeMetricName:
		if value, ok := currentStorage.GetGaugeMetric(metric.ID); ok {
			metric.Value = &value

			printMetric()
			return
		}
	case common.CounterMetricName:
		if value, ok := currentStorage.GetCounterMetric(metric.ID); ok {
			metric.Delta = &value

			printMetric()
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

var defaultCompressibleContentTypes = []string{
	"application/javascript",
	"application/json",
	"text/css",
	"text/html",
	"text/plain",
	"text/xml",
}

func InitRouter(r *chi.Mux, currentStorage storage.StorageInterface, key string, conn *pgxpool.Pool) *chi.Mux {
	r.Use(middleware.Compress(5, defaultCompressibleContentTypes...))

	r.Route("/update", func(r chi.Router) {
		r.Post("/gauge/{metricName}/{metricValue}", func(wr http.ResponseWriter, r *http.Request) {
			UpdateGaugeMetric(wr, r, currentStorage)
		})

		r.Post("/counter/{metricName}/{metricValue}", func(wr http.ResponseWriter, r *http.Request) {
			UpdateCounterMetric(wr, r, currentStorage)
		})

		r.Post("/gauge/", missedMetricNameHandlerFunc)
		r.Post("/counter/", missedMetricNameHandlerFunc)

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			UpdateMetric(w, r, currentStorage, key)
		})

		r.Post("/*", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotImplemented)
		})
	})

	r.Route("/value", func(r chi.Router) {
		r.Get("/gauge/{metricName}", func(w http.ResponseWriter, r *http.Request) {
			GetGaugeMetric(w, r, currentStorage)
		})

		r.Get("/counter/{metricName}", func(w http.ResponseWriter, r *http.Request) {
			GetCounterMetric(w, r, currentStorage)
		})

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			GetMetric(w, r, currentStorage, key)
		})

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		list := ""

		currentStorage.ForEachGaugeMetric(func(metricName string, value float64) {
			list += "<li>" + metricName + " - " + fmt.Sprint(value) + "</li>"
		})
		currentStorage.ForEachCounterMetric(func(metricName string, value int64) {
			list += "<li>" + metricName + " - " + fmt.Sprint(value) + "</li>"
		})

		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<div><ul>%s</ul></div>", list)
	})

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		if conn != nil {
			if conn.Ping(r.Context()) == nil {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		w.WriteHeader(http.StatusInternalServerError)
	})

	return r
}
