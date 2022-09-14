package storage_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/GermanVor/devops-pet-project/internal/storage"
	"github.com/bmizerany/assert"
	"github.com/stretchr/testify/require"
)

func createGaugeMetrics() storage.GaugeMetricsStorage {
	gaugeMetrics := make(storage.GaugeMetricsStorage)
	gaugeMetrics["qwerty"] = 24
	gaugeMetrics["qwerty2"] = 2424

	return gaugeMetrics
}

func createCounterMetrics() storage.CounterMetricsStorage {
	counterMetrics := make(storage.CounterMetricsStorage)
	counterMetrics["ytreeq"] = 5
	counterMetrics["ytre2eq"] = 55

	return counterMetrics
}

func createBackupObject() *storage.BackupObject {
	return &storage.BackupObject{
		GaugeMetrics:   createGaugeMetrics(),
		CounterMetrics: createCounterMetrics(),
	}
}

func compareMaps[T any](t *testing.T, expectedMap, currentMap map[string]T) {
	counterCount := 0

	for name, value := range expectedMap {
		assert.Equal(t, currentMap[name], value)
		counterCount++
	}
	assert.Equal(t, len(currentMap), counterCount)
}

func compareBackupAndStorage(t *testing.T, backupObject *storage.BackupObject, stor *storage.Storage) {
	counterCount := 0
	gaugeCount := 0

	stor.ForEachMetrics(context.TODO(), func(sm *storage.StorageMetric) {
		switch sm.MType {
		case common.GaugeMetricName:
			assert.Equal(t, backupObject.GaugeMetrics[sm.ID], sm.Value)
			gaugeCount++
		case common.CounterMetricName:
			assert.Equal(t, backupObject.CounterMetrics[sm.ID], sm.Delta)
			counterCount++
		}
	})

	assert.Equal(t, len(backupObject.CounterMetrics), counterCount)
	assert.Equal(t, len(backupObject.GaugeMetrics), gaugeCount)
}

func TestMain(t *testing.T) {
	t.Run("Init from backup file", func(t *testing.T) {
		backupFileName := "/tmp/devops-metrics-db.json"

		backupObject := createBackupObject()

		file, err := os.OpenFile(backupFileName, os.O_WRONLY|os.O_CREATE, 0777)
		require.NoError(t, err)
		defer file.Close()
		defer os.Remove(file.Name())

		backupBytes, _ := json.Marshal(backupObject)

		_, err = file.WriteAt(backupBytes, 0)
		require.NoError(t, err)

		storage, err := storage.Init(&backupFileName)
		require.NoError(t, err)

		compareBackupAndStorage(t, backupObject, storage)
	})

	t.Run("Make backup", func(t *testing.T) {
		backupFileName := "./backupTestFile2"

		gaugeMetrics := createGaugeMetrics()
		counterMetrics := createCounterMetrics()

		baseStor, _ := storage.Init(nil)
		stor := storage.WithBackup(baseStor, backupFileName)

		for id, value := range gaugeMetrics {
			err := stor.UpdateMetric(context.TODO(), common.Metrics{
				MType: common.GaugeMetricName,
				ID:    id,
				Value: &value,
			})
			require.NoError(t, err)
		}
		for id, delta := range counterMetrics {
			err := stor.UpdateMetric(context.TODO(), common.Metrics{
				MType: common.CounterMetricName,
				ID:    id,
				Delta: &delta,
			})
			require.NoError(t, err)
		}

		file, err := os.OpenFile(backupFileName, os.O_RDONLY, 0777)
		require.NoError(t, err)
		defer file.Close()
		defer os.Remove(file.Name())

		backupObject := &storage.BackupObject{
			GaugeMetrics:   make(storage.GaugeMetricsStorage),
			CounterMetrics: make(storage.CounterMetricsStorage),
		}
		err = json.NewDecoder(file).Decode(backupObject)
		require.NoError(t, err)

		compareMaps(t, gaugeMetrics, backupObject.GaugeMetrics)
		compareMaps(t, counterMetrics, backupObject.CounterMetrics)
	})
}
