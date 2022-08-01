package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/GermanVor/devops-pet-project/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()
	currentStorage := storage.Init()

	r.Use(middleware.Logger)

	r.Route("/update", func(r chi.Router) {
		r.Post("/gauge/{metricName}/{metricValue}", func(rw http.ResponseWriter, r *http.Request) {
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
		})

		r.Post("/counter/{metricName}/{metricValue}", func(rw http.ResponseWriter, r *http.Request) {
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
		})
	})

	r.Route("/value", func(r chi.Router) {
		r.Get("/gauge/{metricName}", func(rw http.ResponseWriter, r *http.Request) {
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
		})

		r.Get("/counter/{metricName}", func(rw http.ResponseWriter, r *http.Request) {
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

		fmt.Fprintf(rw, "<div><ul>%s</ul></div>", list)
	})

	fmt.Println("Server Started: http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", r))
}
