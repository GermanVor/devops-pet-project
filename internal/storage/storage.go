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
	getGaugeMetrics() (GaugeMetricsStorage, func())
	GetGaugeMetric(string) (float64, bool)
	SetGaugeMetric(string, float64)
	ForEachGaugeMetric(func(string, float64))

	getCounterMetrics() (CounterMetricsStorage, func())
	GetCounterMetric(string) (int64, bool)
	SetCounterMetric(string, int64)
	ForEachCounterMetric(func(string, int64))
}

type Storage struct {
	StorageInterface

	gaugeMap    GaugeMetricsStorage
	gaugeMapRWM sync.Mutex

	counterMap    CounterMetricsStorage
	counterMapRWM sync.Mutex
}

type BackupStorageWrapper struct {
	*Storage
	backupFilePath string
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
	gaugeMap, unlockGaugeMetrics := stor.getGaugeMetrics()
	defer unlockGaugeMetrics()

	counterMap, unlockCounterMap := stor.getCounterMetrics()
	defer unlockCounterMap()

	backup := BackupObject{
		GaugeMetrics:   gaugeMap,
		CounterMetrics: counterMap,
	}

	backupBytes, _ := json.Marshal(&backup)
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

func (stor *BackupStorageWrapper) writeBackup() error {
	return writeStoreBackup(stor.Storage, stor.backupFilePath)
}

func (stor *Storage) getGaugeMetrics() (GaugeMetricsStorage, func()) {
	stor.gaugeMapRWM.Lock()

	return stor.gaugeMap, stor.gaugeMapRWM.Unlock
}

func (stor *Storage) SetGaugeMetric(metricName string, value float64) {
	gaugeMap, deferFunc := stor.getGaugeMetrics()
	defer deferFunc()

	gaugeMap[metricName] = value
}

func (stor *BackupStorageWrapper) SetGaugeMetric(metricName string, value float64) {
	stor.Storage.SetGaugeMetric(metricName, value)
	stor.writeBackup()
}

func (stor *Storage) GetGaugeMetric(metricName string) (float64, bool) {
	gaugeMap, deferFunc := stor.getGaugeMetrics()
	defer deferFunc()

	value, ok := gaugeMap[metricName]
	return value, ok
}

func (stor *Storage) ForEachGaugeMetric(handler func(metricName string, value float64)) {
	gaugeMap, unlockGaugeMetrics := stor.getGaugeMetrics()
	defer unlockGaugeMetrics()

	for a, b := range gaugeMap {
		handler(a, b)
	}
}

func (stor *Storage) getCounterMetrics() (CounterMetricsStorage, func()) {
	stor.counterMapRWM.Lock()
	return stor.counterMap, stor.counterMapRWM.Unlock
}

func (stor *Storage) GetCounterMetric(metricName string) (int64, bool) {
	counterMap, unlockCounterMap := stor.getCounterMetrics()
	defer unlockCounterMap()

	value, ok := counterMap[metricName]
	return value, ok
}

func (stor *BackupStorageWrapper) SetCounterMetric(metricName string, value int64) {
	stor.Storage.SetCounterMetric(metricName, value)
	stor.writeBackup()
}

func (stor *Storage) SetCounterMetric(metricName string, count int64) {
	counterMap, unlockCounterMap := stor.getCounterMetrics()
	defer unlockCounterMap()

	counterMap[metricName] += count
}

func (stor *Storage) ForEachCounterMetric(handler func(metricName string, value int64)) {
	counterMap, unlockCounterMap := stor.getCounterMetrics()
	defer unlockCounterMap()

	for a, b := range counterMap {
		handler(a, b)
	}
}
