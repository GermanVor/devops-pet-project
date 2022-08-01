package handlers_test

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GermanVor/devops-pet-project/cmd/agent/metrics"
	"github.com/GermanVor/devops-pet-project/cmd/agent/utils"
	"github.com/GermanVor/devops-pet-project/cmd/server/handlers"
	"github.com/GermanVor/devops-pet-project/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PositiveInputs(t *testing.T) {
	endpointURL := "http://localhost:8080/"
	currentStorage := storage.Init()

	t.Run("Gauge item", func(t *testing.T) {
		metricName := "qwerty"
		metricValue := rand.Float64()

		request, err := utils.BuildRequest(endpointURL, metrics.GaugeTypeName, metricName, fmt.Sprint(metricValue))
		require.NoError(t, err)

		w := httptest.NewRecorder()
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlers.UpdateStorageFunc(w, r, currentStorage)
		})
		h.ServeHTTP(w, request)
		result := w.Result()

		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.Equal(t, "application/json", result.Header.Get("Content-Type"))

		err = result.Body.Close()
		require.NoError(t, err)

		storageMetcric, _ := currentStorage.GetGaugeMetric(metricName)
		assert.Equal(t, metricValue, storageMetcric)
	})

	t.Run("Counter item", func(t *testing.T) {
		metricName := "qwerty"
		metricValue := rand.Int63()

		request, err := utils.BuildRequest(endpointURL, metrics.CounterTypeName, metricName, fmt.Sprint(metricValue))
		require.NoError(t, err)

		w := httptest.NewRecorder()
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlers.UpdateStorageFunc(w, r, currentStorage)
		})
		h.ServeHTTP(w, request)
		result := w.Result()

		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.Equal(t, "application/json", result.Header.Get("Content-Type"))

		err = result.Body.Close()
		require.NoError(t, err)

		storageMetcric, _ := currentStorage.GetCounterMetric(metricName)
		assert.Equal(t, metricValue, storageMetcric)
	})

	t.Run("Gauge bad metricName", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, "http://localhost:8080/update/gauge/", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Log(strings.Split(r.URL.Path[1:], "/"))

			handlers.UpdateStorageFunc(w, r, currentStorage)
		})
		h.ServeHTTP(w, request)
		result := w.Result()

		assert.Equal(t, http.StatusNotFound, result.StatusCode)
		assert.Equal(t, "application/json", result.Header.Get("Content-Type"))

		err = result.Body.Close()
		require.NoError(t, err)
	})

	t.Run("Counter bad metricName", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, "http://localhost:8080/update/counter/", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlers.UpdateStorageFunc(w, r, currentStorage)
		})
		h.ServeHTTP(w, request)
		result := w.Result()

		assert.Equal(t, http.StatusNotFound, result.StatusCode)
		assert.Equal(t, "application/json", result.Header.Get("Content-Type"))

		err = result.Body.Close()
		require.NoError(t, err)
	})

	t.Run("Gauge bad value", func(t *testing.T) {
		metricName := "qwerty2"
		request, err := http.NewRequest(http.MethodPost, "http://localhost:8080/update/gauge/"+metricName+"/", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlers.UpdateStorageFunc(w, r, currentStorage)
		})
		h.ServeHTTP(w, request)
		result := w.Result()

		assert.Equal(t, http.StatusBadRequest, result.StatusCode)
		assert.Equal(t, "application/json", result.Header.Get("Content-Type"))

		err = result.Body.Close()
		require.NoError(t, err)

		storageMetcric, _ := currentStorage.GetGaugeMetric(metricName)
		assert.Equal(t, float64(0), storageMetcric)
	})

	t.Run("Counter bad value", func(t *testing.T) {
		metricName := "qwerty2"
		request, err := http.NewRequest(http.MethodPost, "http://localhost:8080/update/counter/"+metricName+"/", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlers.UpdateStorageFunc(w, r, currentStorage)
		})
		h.ServeHTTP(w, request)
		result := w.Result()

		assert.Equal(t, http.StatusBadRequest, result.StatusCode)
		assert.Equal(t, "application/json", result.Header.Get("Content-Type"))

		err = result.Body.Close()
		require.NoError(t, err)

		storageMetcric, _ := currentStorage.GetCounterMetric(metricName)
		assert.Equal(t, int64(0), storageMetcric)
	})
}
