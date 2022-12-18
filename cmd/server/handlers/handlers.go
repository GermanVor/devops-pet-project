// All the necessary endpoints for storing Metrics in Storage.
package handlers

import (
	"bytes"
	"compress/gzip"
	"crypto/rsa"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/crypto"
	"github.com/GermanVor/devops-pet-project/internal/storage"
	"github.com/go-chi/chi"
)

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
func UpdateMetric(
	w http.ResponseWriter,
	r *http.Request,
	stor storage.StorageInterface,
	key string,
) {
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
func UpdateMetrics(
	w http.ResponseWriter,
	r *http.Request,
	stor storage.StorageInterface,
) {
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

// func UseQwerty(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		next.ServeHTTP(w, r.WithContext(ctx))
// 	})
// }

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

func MiddlewareDecompressGzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			bodyBytes, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			var bb []byte

			gz, err := gzip.NewReader(ioutil.NopCloser(bytes.NewBuffer(bodyBytes)))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer gz.Close()

			bb, err = ioutil.ReadAll(gz)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			r.ContentLength = int64(len(bb))
			r.Body = ioutil.NopCloser(bytes.NewReader(bb))
		}

		next.ServeHTTP(w, r)
	})
}

func MiddlewareEncryptBodyData(rsaKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			metricBytes, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer r.Body.Close()

			decryptedMetricBytes, err := crypto.RSADecrypt(metricBytes, rsaKey)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			
			r.ContentLength = int64(len(decryptedMetricBytes))
			r.Body = ioutil.NopCloser(bytes.NewReader(decryptedMetricBytes))

			next.ServeHTTP(w, r)
		})
	}
}

func InitRouter(r *chi.Mux, stor storage.StorageInterface, key string) *chi.Mux {
	if stor == nil {
		log.Fatalln("Storage do not created")
	}

	r.Post("/update/", func(w http.ResponseWriter, r *http.Request) {
		UpdateMetric(w, r, stor, key)
	})

	r.Post("/updates/", func(w http.ResponseWriter, r *http.Request) {
		UpdateMetrics(w, r, stor)
	})

	r.Post("/value/", func(w http.ResponseWriter, r *http.Request) {
		GetMetric(w, r, stor, key)
	})

	return r
}
