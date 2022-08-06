package handlers_test

import (
	"devops-pet-project/cmd/agent/metrics"
	"devops-pet-project/cmd/agent/utils"
	"devops-pet-project/cmd/server/handlers"
	"devops-pet-project/storage"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/require"
)

func float64frombytes(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint64(bytes)
	float := math.Float64frombits(bits)

	return float
}

func TestIndex(t *testing.T) {
	currentStorage := storage.Init()
	r := chi.NewRouter()

	handlers.InitRouter(r, currentStorage)

	ts := httptest.NewServer(r)
	defer ts.Close()

	endpointURL := ts.URL + "/"

	gaugeMetricName := "qwerty"
	gaugeMetricValue := rand.Float64()

	t.Run("Update Gauge metric", func(t *testing.T) {
		req, err := utils.BuildRequest(endpointURL, metrics.GaugeTypeName, gaugeMetricName, fmt.Sprint(gaugeMetricValue))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		storageMetcric, _ := currentStorage.GetGaugeMetric(gaugeMetricName)
		assert.Equal(t, gaugeMetricValue, storageMetcric)
	})

	t.Run("Get Gauge metric", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, endpointURL+"value/gauge/"+gaugeMetricName, nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))

		metricValueFromServer, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, fmt.Sprint(gaugeMetricValue), string(metricValueFromServer))
	})

	counterMetricName := "qwerty2"
	counterMetricValue := rand.Int63()

	t.Run("Update Counter metric", func(t *testing.T) {
		req, err := utils.BuildRequest(endpointURL, metrics.CounterTypeName, counterMetricName, fmt.Sprint(counterMetricValue))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		err = resp.Body.Close()
		require.NoError(t, err)

		storageMetcric, _ := currentStorage.GetCounterMetric(counterMetricName)
		assert.Equal(t, counterMetricValue, storageMetcric)
	})

	t.Run("Gauge bad metricName", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, endpointURL+"update/gauge/", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		err = resp.Body.Close()
		require.NoError(t, err)
	})

	t.Run("Counter bad metricName", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, endpointURL+"update/counter/", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		err = resp.Body.Close()
		require.NoError(t, err)
	})

	t.Run("Gauge bad value", func(t *testing.T) {
		metricName := "qwerty3"
		req, err := http.NewRequest(http.MethodPost, endpointURL+"update/gauge/"+metricName+"/qwe", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		err = resp.Body.Close()
		require.NoError(t, err)

		storageMetcric, _ := currentStorage.GetGaugeMetric(metricName)
		assert.Equal(t, float64(0), storageMetcric)
	})

	t.Run("Counter bad value", func(t *testing.T) {
		metricName := "qwerty4"
		req, err := http.NewRequest(http.MethodPost, endpointURL+"update/counter/"+metricName+"/qwe", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		err = resp.Body.Close()
		require.NoError(t, err)

		storageMetcric, _ := currentStorage.GetCounterMetric(metricName)
		assert.Equal(t, int64(0), storageMetcric)
	})

	t.Run("Get all metrics List", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, endpointURL, nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		stringBody := string(body)

		assert.Equal(t, true, strings.Contains(stringBody, gaugeMetricName+" - "+fmt.Sprint(gaugeMetricValue)))
		assert.Equal(t, true, strings.Contains(stringBody, counterMetricName+" - "+fmt.Sprint(counterMetricValue)))

		assert.Equal(t, 4, strings.Count(stringBody, "li"))
	})
}