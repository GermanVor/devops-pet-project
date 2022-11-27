// All the necessary endpoints for storing Metrics in Storage.
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/storage"
	"github.com/go-chi/chi"
)

// UpdateMetricV1 [Depricatred] Handler to save Agent metrics by URL.
//
// URL view: /update/{mType}/{id}/{metricValue} where
// mType - (gauge|counter), id - Metric Id, metricValue - (float64|int64)
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

// UpdateMetric Handler to save Agent metrics by request Body.
//
// key - secret key to for authorization.
//
// Expected Request Body interface is Metrics.
//
//	type Metrics struct {
//		ID    string   `json:"id"`              // имя метрики
//		MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
//		Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
//		Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
//		Hash  string   `json:"hash,omitempty"`  // значение хеш-функции
//	}
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

// UpdateMetrics Handler to save pack of Metrics by request Body.
//
// Expected Request Body interface is []Metrics.
//
//	type Metrics struct {
//		ID    string   `json:"id"`              // имя метрики
//		MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
//		Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
//		Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
//		Hash  string   `json:"hash,omitempty"`  // значение хеш-функции
//	}
func UpdateMetrics(w http.ResponseWriter, r *http.Request, stor storage.StorageInterface) {
	metricsArr := []common.Metrics{}

	if err := json.NewDecoder(r.Body).Decode(&metricsArr); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := stor.UpdateMetrics(r.Context(), metricsArr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func missedMetricNameHandlerFunc(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

// GetMetricV1 [Depricatred] Handler to get Agent metrics by URL.
//
// URL view: /value/{mType}/{id} where
// mType - (gauge|counter), id - Metric Id.
//
// Response is Metric Value as String.
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

// GetMetric Handler to get Agent metrics by URL.
//
// key - secret key to for authorization.
//
// Expected Request Body interface is Metrics. Delta and Value field in Request will be ignored.
//
//	type Metrics struct {
//		ID    string   `json:"id"`              // имя метрики
//		MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
//		Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
//		Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
//		Hash  string   `json:"hash,omitempty"`  // значение хеш-функции
//	}
//
// Response is Metric Value as String.
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

func GetAllMetrics(w http.ResponseWriter, r *http.Request, stor storage.StorageInterface) {
	list := make([]string, 0)

	err := stor.ForEachMetrics(r.Context(), func(sm *storage.StorageMetric) {
		item := ""

		switch sm.MType {
		case common.GaugeMetricName:
			item = fmt.Sprint(sm.Value)
		case common.CounterMetricName:
			item = fmt.Sprint(sm.Delta)
		}

		list = append(list, fmt.Sprintf("<li>%s - %s</li>", sm.ID, item))
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<div><ul>%s</ul></div>", strings.Join(list, ""))
	}
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

	r.Post("/updates/", func(w http.ResponseWriter, r *http.Request) {
		UpdateMetrics(w, r, stor)
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
		GetAllMetrics(w, r, stor)
	})

	return r
}
