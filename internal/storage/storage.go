package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type GaugeMetricsStorage map[string]float64
type CounterMetricsStorage map[string]int64

type StorageInterface interface {
	GetGaugeMetric(string) (float64, bool)
	SetGaugeMetric(string, float64)
	ForEachGaugeMetric(func(string, float64))

	GetCounterMetric(string) (int64, bool)
	SetCounterMetric(string, int64)
	ForEachCounterMetric(func(string, int64))
}

type Storage struct {
	StorageInterface

	gaugeMap    GaugeMetricsStorage
	gaugeMapRWM sync.RWMutex

	counterMap    CounterMetricsStorage
	counterMapRWM sync.RWMutex
}

type BackupStorageWrapper struct {
	*Storage
	backupFilePath string
	fileRWM sync.Mutex
}

type BackupObject struct {
	GaugeMetrics   GaugeMetricsStorage
	CounterMetrics CounterMetricsStorage
}

func createStorageFromBackup(storage *Storage, initialFilePath string) error {
	file, err := os.OpenFile(initialFilePath, os.O_RDONLY, 0777)
	if err != nil {
		return err
	}
	defer file.Close()

	backupObject := &BackupObject{
		GaugeMetrics:   make(GaugeMetricsStorage),
		CounterMetrics: make(CounterMetricsStorage),
	}
	err = json.NewDecoder(file).Decode(backupObject)

	if err == nil {
		storage.counterMap = backupObject.CounterMetrics
		storage.gaugeMap = backupObject.GaugeMetrics
	}

	return err
}

func Init(initialFilePath *string) (*Storage, error) {
	storage := &Storage{
		gaugeMap:   make(GaugeMetricsStorage),
		counterMap: make(CounterMetricsStorage),
	}

	var err error

	if initialFilePath != nil {
		err = createStorageFromBackup(storage, *initialFilePath)

		if err == nil {
			fmt.Println("Storage is successfully restored from backup")
		} else {
			fmt.Println("Storage is not restored from backup,", err)
		}
	}

	return storage, err
}

func writeStoreBackup(stor *Storage, backupFilePath string) error {
	stor.gaugeMapRWM.RLock()
	stor.counterMapRWM.RLock()

	backup := BackupObject{
		GaugeMetrics:   stor.gaugeMap,
		CounterMetrics: stor.counterMap,
	}

	backupBytes, _ := json.Marshal(&backup)

	stor.gaugeMapRWM.RUnlock()
	stor.counterMapRWM.RUnlock()

	return os.WriteFile(backupFilePath, backupBytes, 0644)
}

func WithBackup(storage *Storage, backupFilePath string) StorageInterface {
	return &BackupStorageWrapper{
		backupFilePath: backupFilePath,
		Storage:        storage,
	}
}

func InitBackupTicker(storage *Storage, backupFilePath string, backupInterval time.Duration) func() {
	ticker := time.NewTicker(backupInterval)
	doneFlag := make(chan struct{})

	stopTicker := func() {
		doneFlag <- struct{}{}
	}

	go func() {
		for {
			select {
			case <-doneFlag:
				fmt.Println("Готово!")
				return
			case <-ticker.C:
				err := writeStoreBackup(storage, backupFilePath)

				if err != nil {
					fmt.Println("Couldnot create backup", err)
				}
			}
		}
	}()

	return stopTicker
}

func (stor *Storage) SetGaugeMetric(metricName string, value float64) {
	stor.gaugeMapRWM.Lock()
	defer stor.gaugeMapRWM.Unlock()

	stor.gaugeMap[metricName] = value
}

func (stor *BackupStorageWrapper) SetGaugeMetric(metricName string, value float64) {
	stor.Storage.SetGaugeMetric(metricName, value)

	stor.fileRWM.Lock()
	defer stor.fileRWM.Unlock()

	writeStoreBackup(stor.Storage, stor.backupFilePath)
}

func (stor *Storage) GetGaugeMetric(metricName string) (float64, bool) {
	stor.gaugeMapRWM.RLock()
	defer stor.gaugeMapRWM.RUnlock()

	value, ok := stor.gaugeMap[metricName]
	return value, ok
}

func (stor *Storage) ForEachGaugeMetric(handler func(metricName string, value float64)) {
	stor.gaugeMapRWM.RLock()
	defer stor.gaugeMapRWM.RUnlock()

	for a, b := range stor.gaugeMap {
		handler(a, b)
	}
}

func (stor *Storage) GetCounterMetric(metricName string) (int64, bool) {
	stor.counterMapRWM.RLock()
	defer stor.counterMapRWM.RUnlock()

	value, ok := stor.counterMap[metricName]
	return value, ok
}

func (stor *BackupStorageWrapper) SetCounterMetric(metricName string, value int64) {
	stor.Storage.SetCounterMetric(metricName, value)

	stor.fileRWM.Lock()
	defer stor.fileRWM.Unlock()

	writeStoreBackup(stor.Storage, stor.backupFilePath)
}

func (stor *Storage) SetCounterMetric(metricName string, count int64) {
	stor.counterMapRWM.Lock()
	defer stor.counterMapRWM.Unlock()

	stor.counterMap[metricName] += count
}

func (stor *Storage) ForEachCounterMetric(handler func(metricName string, value int64)) {
	stor.counterMapRWM.RLock()
	defer stor.counterMapRWM.RUnlock()

	for a, b := range stor.counterMap {
		handler(a, b)
	}
}
