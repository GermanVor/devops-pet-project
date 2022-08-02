package handlers_test

import (
	"devops-pet-project/cmd/agent/metrics"
	"devops-pet-project/cmd/agent/utils"
	"devops-pet-project/cmd/server/handlers"
	"devops-pet-project/storage"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/stretchr/testify/require"
)

func TestIndex(t *testing.T) {
	currentStorage := storage.Init()
	r := handlers.InitRouter(currentStorage)

	ts := httptest.NewServer(r)
	defer ts.Close()

	endpointURL := ts.URL + "/"

	t.Run("Gauge item", func(t *testing.T) {
		metricName := "qwerty"
		metricValue := rand.Float64()

		req, err := utils.BuildRequest(endpointURL, metrics.GaugeTypeName, metricName, fmt.Sprint(metricValue))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		storageMetcric, _ := currentStorage.GetGaugeMetric(metricName)
		assert.Equal(t, metricValue, storageMetcric)
	})

	t.Run("Counter item", func(t *testing.T) {
		metricName := "qwerty2"
		metricValue := rand.Int63()

		req, err := utils.BuildRequest(endpointURL, metrics.CounterTypeName, metricName, fmt.Sprint(metricValue))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		defer resp.Body.Close()

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		err = resp.Body.Close()
		require.NoError(t, err)

		storageMetcric, _ := currentStorage.GetCounterMetric(metricName)
		assert.Equal(t, metricValue, storageMetcric)
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
}
