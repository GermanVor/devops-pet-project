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
)

func UpdateMetricV1(w http.ResponseWriter, r *http.Request, stor storage.StorageInterface) {
	metric := common.Metrics{
		MType: chi.URLParam(r, "mType"),
		ID:    chi.URLParam(r, "id"),
	}

	switch metric.MType {
	case common.GaugeMetricName:
		value, err := strconv.ParseFloat(chi.URLParam(r, "metricValue"), 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		metric.Value = &value
	case common.CounterMetricName:
		delta, err := strconv.ParseInt(chi.URLParam(r, "metricValue"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		metric.Delta = &delta
	default:
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	err := stor.UpdateMetric(r.Context(), metric)

	if err == nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func UpdateMetric(w http.ResponseWriter, r *http.Request, stor storage.StorageInterface, key string) {
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
	case common.CounterMetricName:
	case common.GaugeMetricName:
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := stor.UpdateMetric(r.Context(), *metric); err == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func missedMetricNameHandlerFunc(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func GetMetricV1(w http.ResponseWriter, r *http.Request, stor storage.StorageInterface) {
	mType := chi.URLParam(r, "mType")
	id := chi.URLParam(r, "id")

	switch mType {
	case common.GaugeMetricName:
	case common.CounterMetricName:
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metric, err := stor.GetMetric(r.Context(), mType, id)

	if err == nil {
		if metric != nil {
			w.Header().Add("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)

			switch mType {
			case common.GaugeMetricName:
				w.Write([]byte(fmt.Sprint(metric.Value)))
			case common.CounterMetricName:
				w.Write([]byte(fmt.Sprint(metric.Delta)))
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func GetMetric(w http.ResponseWriter, r *http.Request, stor storage.StorageInterface, key string) {
	w.Header().Set("Content-Type", "application/json")
	metric := &common.Metrics{}

	if err := json.NewDecoder(r.Body).Decode(metric); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch metric.MType {
	case common.GaugeMetricName:
	case common.CounterMetricName:
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}

	storMetric, err := stor.GetMetric(r.Context(), metric.MType, metric.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if storMetric == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch metric.MType {
	case common.GaugeMetricName:
		metric.Value = &storMetric.Value
	case common.CounterMetricName:
		metric.Delta = &storMetric.Delta
	}

	if key != "" {
		metric.Hash, _ = common.GetMetricHash(metric, key)
	}

	jsonResp, _ := metric.MarshalJSON()
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResp)
}

func InitRouter(r *chi.Mux, stor storage.StorageInterface, key string) *chi.Mux {
	if stor == nil {
		log.Fatalln("Storage do not created")
	}

	r.Route("/update", func(r chi.Router) {
		r.Post("/{mType}/{id}/{metricValue}", func(w http.ResponseWriter, r *http.Request) {
			UpdateMetricV1(w, r, stor)
		})

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			UpdateMetric(w, r, stor, key)
		})

		r.Post("/*", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotImplemented)
		})
		r.Post("/gauge/", missedMetricNameHandlerFunc)
		r.Post("/counter/", missedMetricNameHandlerFunc)
	})

	r.Route("/value", func(r chi.Router) {
		r.Get("/{mType}/{id}", func(w http.ResponseWriter, r *http.Request) {
			GetMetricV1(w, r, stor)
		})

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			GetMetric(w, r, stor, key)
		})

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		list := ""

		err := stor.ForEachMetrics(r.Context(), func(sm *storage.StorageMetric) {
			list += "<li>" + sm.ID + " - "

			switch sm.MType {
			case common.GaugeMetricName:
				list += fmt.Sprint(sm.Value)
			case common.CounterMetricName:
				list += fmt.Sprint(sm.Delta)
			}

			list += "</li>"
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.Header().Add("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, "<div><ul>%s</ul></div>", list)
		}
	})

	return r
}
