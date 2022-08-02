package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"devops-pet-project/storage"

	"github.com/go-chi/chi"
)

func UpdateGaugeMetric(rw http.ResponseWriter, r *http.Request, currentStorage *storage.Storage) {
	rw.Header().Add("Content-Type", "application/json")

	metricName := chi.URLParam(r, "metricName")
	metricValue, err := strconv.ParseFloat(chi.URLParam(r, "metricValue"), 64)

	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write(nil)
		return
	}

	currentStorage.SetGaugeMetric(metricName, metricValue)

	rw.WriteHeader(http.StatusOK)
	rw.Write(nil)
}

func UpdateCounterMetric(rw http.ResponseWriter, r *http.Request, currentStorage *storage.Storage) {
	rw.Header().Add("Content-Type", "application/json")

	metricName := chi.URLParam(r, "metricName")
	metricValue, err := strconv.ParseInt(chi.URLParam(r, "metricValue"), 10, 64)

	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write(nil)
		return
	}

	currentStorage.IncreaseCounterMetric(metricName, metricValue)

	rw.WriteHeader(http.StatusOK)
	rw.Write(nil)
}

func GetGaugeMetric(rw http.ResponseWriter, r *http.Request, currentStorage *storage.Storage) {
	rw.Header().Add("Content-Type", "text/plain")

	metricName := chi.URLParam(r, "metricName")
	value, ok := currentStorage.GetGaugeMetric(metricName)

	if ok {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(fmt.Sprint(value)))
	} else {
		rw.WriteHeader(http.StatusNotFound)
		rw.Write(nil)
	}
}

func GetCounterMetric(rw http.ResponseWriter, r *http.Request, currentStorage *storage.Storage) {
	rw.Header().Add("Content-Type", "text/plain")

	metricName := chi.URLParam(r, "metricName")
	value, ok := currentStorage.GetCounterMetric(metricName)

	if ok {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(fmt.Sprint(value)))
	} else {
		rw.WriteHeader(http.StatusNotFound)
		rw.Write(nil)
	}
}

func missedMetricNameHandlerFunc(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(http.StatusNotFound)
	rw.Write(nil)
}

func InitRouter(currentStorage *storage.Storage) *chi.Mux {
	r := chi.NewRouter()

	r.Route("/update", func(r chi.Router) {
		r.Post("/gauge/{metricName}/{metricValue}", func(wr http.ResponseWriter, r *http.Request) {
			UpdateGaugeMetric(wr, r, currentStorage)
		})

		r.Post("/counter/{metricName}/{metricValue}", func(wr http.ResponseWriter, r *http.Request) {
			UpdateCounterMetric(wr, r, currentStorage)
		})

		r.Post("/gauge/", missedMetricNameHandlerFunc)
		r.Post("/counter/", missedMetricNameHandlerFunc)

		r.Post("/*", func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(http.StatusNotImplemented)
			rw.Write(nil)
		})
	})

	r.Route("/value", func(r chi.Router) {
		r.Get("/gauge/{metricName}", func(rw http.ResponseWriter, r *http.Request) {
			GetGaugeMetric(rw, r, currentStorage)
		})

		r.Get("/counter/{metricName}", func(rw http.ResponseWriter, r *http.Request) {
			GetCounterMetric(rw, r, currentStorage)
		})

		r.Get("/", func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(http.StatusNotFound)
			rw.Write(nil)
		})
	})

	r.Get("/", func(rw http.ResponseWriter, r *http.Request) {
		list := ""

		currentStorage.ForEachGaugeMetric(func(metricName string, value float64) {
			list += "<li>" + metricName + " - " + fmt.Sprint(value) + "</li>"
		})
		currentStorage.ForEachCounterMetric(func(metricName string, value int64) {
			list += "<li>" + metricName + " - " + fmt.Sprint(value) + "</li>"
		})

		rw.Header().Add("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(rw, "<div><ul>%s</ul></div>", list)
	})

	return r
}
