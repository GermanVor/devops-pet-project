package handlers

import (
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

// GetAllMetrics [Depricatred] Handler to get all metrics as html page
// with content format:
//
//	<div>
//		<ul>
//			<li>${metricId} - ${metricValue}</li>
//			...
//		</ul>
//	</div>
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

func InitRouterV1(r *chi.Mux, stor storage.StorageInterface) *chi.Mux {
	if stor == nil {
		log.Fatalln("Storage do not created")
	}

	r.Route("/update", func(r chi.Router) {
		r.Post("/{mType}/{id}/{metricValue}", func(w http.ResponseWriter, r *http.Request) {
			UpdateMetricV1(w, r, stor)
		})

		r.Post("/*", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotImplemented)
		})
		r.Post("/gauge/", missedMetricNameHandlerFunc)
		r.Post("/counter/", missedMetricNameHandlerFunc)
	})

	r.Get("/value/{mType}/{id}", func(w http.ResponseWriter, r *http.Request) {
		GetMetricV1(w, r, stor)
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		GetAllMetrics(w, r, stor)
	})

	return r
}
