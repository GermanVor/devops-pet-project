package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GermanVor/devops-pet-project/cmd/agent/metrics"
	"github.com/GermanVor/devops-pet-project/cmd/agent/utils"
	"github.com/GermanVor/devops-pet-project/cmd/server/handlers"
	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/storage"
	"github.com/bmizerany/assert"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/require"
)

func createTestEnvironment(key string) (*storage.Storage, string, func()) {
	currentStorage, _ := storage.Init(nil)

	r := chi.NewRouter()

	handlers.InitRouter(r, currentStorage, key)

	ts := httptest.NewServer(r)

	destructor := func() {
		ts.Close()
	}

	return currentStorage, ts.URL, destructor
}

func TestServerOperations(t *testing.T) {
	t.Run("Update Gauge metric", func(t *testing.T) {
		gaugeMetricName := "qwerty"
		gaugeMetricValue := rand.Float64()

		currentStorage, endpointURL, destructor := createTestEnvironment("")
		defer destructor()

		{
			req, err := utils.BuildRequest(endpointURL, metrics.GaugeTypeName, gaugeMetricName, fmt.Sprint(gaugeMetricValue))
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

			storageMetcric, err := currentStorage.GetMetric(context.TODO(), common.GaugeMetricName, gaugeMetricName)
			require.NoError(t, err)
			assert.Equal(t, gaugeMetricValue, storageMetcric.Value)
		}

		{
			req, err := http.NewRequest(http.MethodGet, endpointURL+"/value/gauge/"+gaugeMetricName, nil)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))

			metricValueFromServer, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, fmt.Sprint(gaugeMetricValue), string(metricValueFromServer))
		}
	})

	t.Run("Update Counter metric", func(t *testing.T) {
		counterMetricName := "qwerty2"
		delta := rand.Int63()

		currentStorage, endpointURL, destructor := createTestEnvironment("")
		defer destructor()

		req, err := utils.BuildRequest(endpointURL, metrics.CounterTypeName, counterMetricName, fmt.Sprint(delta))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		storageMetcric, err := currentStorage.GetMetric(context.TODO(), common.CounterMetricName, counterMetricName)
		require.NoError(t, err)
		assert.Equal(t, delta, storageMetcric.Delta)
	})

	t.Run("Gauge bad metricName", func(t *testing.T) {
		_, endpointURL, destructor := createTestEnvironment("")
		defer destructor()

		req, err := http.NewRequest(http.MethodPost, endpointURL+"/update/gauge/", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		err = resp.Body.Close()
		require.NoError(t, err)
	})

	t.Run("Counter bad metricName", func(t *testing.T) {
		_, endpointURL, destructor := createTestEnvironment("")
		defer destructor()

		req, err := http.NewRequest(http.MethodPost, endpointURL+"/update/counter/", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		err = resp.Body.Close()
		require.NoError(t, err)
	})

	t.Run("Gauge bad value", func(t *testing.T) {
		currentStorage, endpointURL, destructor := createTestEnvironment("")
		defer destructor()

		metricName := "qwerty3"
		req, err := http.NewRequest(http.MethodPost, endpointURL+"/update/gauge/"+metricName+"/qwe", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		err = resp.Body.Close()
		require.NoError(t, err)

		storageMetcric, err := currentStorage.GetMetric(context.TODO(), common.GaugeMetricName, metricName)
		assert.Equal(t, nil, err)
		assert.Equal(t, (*storage.StorageMetric)(nil), storageMetcric)
	})

	t.Run("Counter bad value", func(t *testing.T) {
		currentStorage, endpointURL, destructor := createTestEnvironment("")
		defer destructor()

		metricName := "qwerty4"
		req, err := http.NewRequest(http.MethodPost, endpointURL+"/update/counter/"+metricName+"/qwe", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		err = resp.Body.Close()
		require.NoError(t, err)

		storageMetcric, err := currentStorage.GetMetric(context.TODO(), common.CounterMetricName, metricName)
		assert.Equal(t, nil, err)
		assert.Equal(t, (*storage.StorageMetric)(nil), storageMetcric)
	})

	t.Run("Get all metrics List", func(t *testing.T) {
		currentStorage, endpointURL, destructor := createTestEnvironment("")
		defer destructor()

		gaugeMetricName := "qwerty"
		gaugeMetricValue := rand.Float64()

		{
			req, err := utils.BuildRequest(endpointURL, metrics.GaugeTypeName, gaugeMetricName, fmt.Sprint(gaugeMetricValue))
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
		}

		counterMetricName := "qwerty2"
		counterMetricValue := rand.Int63()

		{
			req, err := utils.BuildRequest(endpointURL, metrics.CounterTypeName, counterMetricName, fmt.Sprint(counterMetricValue))
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
		}

		req, err := http.NewRequest(http.MethodGet, endpointURL, nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		stringBody := string(body)

		assert.Equal(t, true, strings.Contains(stringBody, gaugeMetricName+" - "+fmt.Sprint(gaugeMetricValue)))
		assert.Equal(t, true, strings.Contains(stringBody, counterMetricName+" - "+fmt.Sprint(counterMetricValue)))

		assert.Equal(t, 4, strings.Count(stringBody, "li"))

		storageGaugeMetcric, _ := currentStorage.GetMetric(context.TODO(), common.GaugeMetricName, gaugeMetricName)
		assert.Equal(t, gaugeMetricValue, storageGaugeMetcric.Value)

		storageCounterMetcric, _ := currentStorage.GetMetric(context.TODO(), common.CounterMetricName, counterMetricName)
		assert.Equal(t, counterMetricValue, storageCounterMetcric.Delta)
	})
}

func TestServerOperationsV2(t *testing.T) {
	gaugeTestFunc := func(t *testing.T, key string) {
		currentStorage, endpointURL, destructor := createTestEnvironment(key)
		defer destructor()

		value := rand.Float64()
		metric := &common.Metrics{
			ID:    "qwerty",
			MType: common.GaugeMetricName,
			Value: &value,
		}

		if key != "" {
			metric.Hash, _ = common.GetMetricHash(metric, key)
		}

		jsonResp, err := metric.MarshalJSON()
		require.NoError(t, err)

		{
			resp, err := http.DefaultClient.Post(endpointURL+"/update/", "application/json", bytes.NewReader(jsonResp))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			storageMetcric, err := currentStorage.GetMetric(context.TODO(), common.GaugeMetricName, metric.ID)
			require.NoError(t, err)
			assert.Equal(t, value, storageMetcric.Value)
		}

		{
			resp, err := http.DefaultClient.Post(endpointURL+"/value/", "application/json", bytes.NewReader(jsonResp))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			respMetric := common.Metrics{}
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&respMetric))

			assert.Equal(t, metric.Value, respMetric.Value)
			assert.Equal(t, metric.ID, respMetric.ID)
			assert.Equal(t, metric.MType, respMetric.MType)
			assert.Equal(t, metric.Hash, respMetric.Hash)
		}
	}

	t.Run("Update Gauge metric", func(t *testing.T) {
		gaugeTestFunc(t, "")
	})
	t.Run("Update Gauge metric with key", func(t *testing.T) {
		gaugeTestFunc(t, "cx,;s;dfends")
	})

	counterTestFunc := func(t *testing.T, key string) {
		currentStorage, endpointURL, destructor := createTestEnvironment(key)
		defer destructor()

		delta := rand.Int63()
		metric := &common.Metrics{
			ID:    "qwerty",
			MType: common.CounterMetricName,
			Delta: &delta,
		}

		if key != "" {
			metric.Hash, _ = common.GetMetricHash(metric, key)
		}

		jsonResp, err := metric.MarshalJSON()
		require.NoError(t, err)

		{
			resp, err := http.DefaultClient.Post(endpointURL+"/update/", "application/json", bytes.NewReader(jsonResp))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			storageMetcric, err := currentStorage.GetMetric(context.TODO(), common.CounterMetricName, metric.ID)
			require.NoError(t, err)
			assert.Equal(t, delta, storageMetcric.Delta)
		}

		{
			resp, err := http.DefaultClient.Post(endpointURL+"/value/", "application/json", bytes.NewReader(jsonResp))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			respMetric := common.Metrics{}
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&respMetric))

			t.Log(respMetric)

			assert.Equal(t, metric.Delta, respMetric.Delta)
			assert.Equal(t, metric.ID, respMetric.ID)
			assert.Equal(t, metric.MType, respMetric.MType)
			assert.Equal(t, metric.Hash, respMetric.Hash)
		}
	}

	t.Run("Update Counter metric", func(t *testing.T) {
		counterTestFunc(t, "")
	})
	t.Run("Update Counter metric with key", func(t *testing.T) {
		counterTestFunc(t, "zxmxlcjsda")
	})
}
