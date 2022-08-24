package storage_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/GermanVor/devops-pet-project/storage"
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

func compareBackupAndStorage(t *testing.T, backupObject *storage.BackupObject, storage *storage.Storage) {
	counterCount := 0
	storage.ForEachCounterMetric(func(metricName string, value int64) {
		assert.Equal(t, backupObject.CounterMetrics[metricName], value)
		counterCount++
	})
	assert.Equal(t, len(backupObject.CounterMetrics), counterCount)

	gaugeCount := 0
	storage.ForEachGaugeMetric(func(metricName string, value float64) {
		assert.Equal(t, backupObject.GaugeMetrics[metricName], value)
		gaugeCount++
	})
	assert.Equal(t, len(backupObject.GaugeMetrics), gaugeCount)
}

func TestMain(t *testing.T) {
	t.Run("Init from backup file", func(t *testing.T) {
		backupFileName := "./backupTestFile"

		backupObject := createBackupObject()

		file, err := os.OpenFile(backupFileName, os.O_WRONLY|os.O_CREATE, 0777)
		require.NoError(t, err)
		defer file.Close()
		defer os.Remove(file.Name())

		backupBytes, _ := json.Marshal(backupObject)

		_, err = file.WriteAt(backupBytes, 0)
		require.NoError(t, err)

		storage, initOutputs, destructor := storage.Init(&storage.InitOptions{
			InitialFilePath: &backupFileName,
			BackupFilePath:  nil,
			BackupInterval:  time.Duration(0),
		})
		defer destructor()

		require.NoError(t, initOutputs.InitialFileError)

		compareBackupAndStorage(t, backupObject, storage)
	})

	t.Run("Make backup", func(t *testing.T) {
		backupFileName := "./backupTestFile2"

		gaugeMetrics := createGaugeMetrics()
		counterMetrics := createCounterMetrics()

		stor, _, destructor := storage.Init(&storage.InitOptions{
			InitialFilePath: nil,
			BackupFilePath:  &backupFileName,
			BackupInterval:  time.Duration(0),
		})
		defer destructor()

		for name, value := range gaugeMetrics {
			stor.SetGaugeMetric(name, value)
		}
		for name, value := range counterMetrics {
			stor.SetCounterMetric(name, value)
		}

		file, err := os.OpenFile(backupFileName, os.O_RDONLY, 0777)
		require.NoError(t, err)
		defer file.Close()

		backupObject := &storage.BackupObject{
			GaugeMetrics:   make(storage.GaugeMetricsStorage),
			CounterMetrics: make(storage.CounterMetricsStorage),
		}
		err = json.NewDecoder(file).Decode(backupObject)
		require.NoError(t, err)

		compareMaps(t, gaugeMetrics, backupObject.GaugeMetrics)
		compareMaps(t, counterMetrics, backupObject.CounterMetrics)

		os.Remove(backupFileName)
	})
}
