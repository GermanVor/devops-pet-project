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

type Storage struct {
	gaugeMap    GaugeMetricsStorage
	gaugeMapRWM *sync.Mutex

	counterMap    CounterMetricsStorage
	counterMapRWM *sync.Mutex

	backupFile     *os.File
	backupInterval time.Duration
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

func withBackup(storage *Storage, backupFilePath string, backupInterval time.Duration) error {
	file, err := os.OpenFile(backupFilePath, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		return err
	}

	storage.backupFile = file
	storage.backupInterval = backupInterval

	return nil
}

func makeBackup(stor *Storage) error {
	stor.gaugeMapRWM.Lock()
	defer stor.gaugeMapRWM.Unlock()

	stor.counterMapRWM.Lock()
	defer stor.counterMapRWM.Unlock()

	backup := BackupObject{
		GaugeMetrics:   stor.gaugeMap,
		CounterMetrics: stor.counterMap,
	}

	backupBytes, _ := json.Marshal(&backup)

	err := stor.backupFile.Truncate(0)
	if err != nil {
		return err
	}

	_, err = stor.backupFile.WriteAt(backupBytes, 0)

	return err
}

type InitOptions struct {
	InitialFilePath *string
	BackupFilePath  *string
	BackupInterval  time.Duration
}

type InitOutputs struct {
	InitialFileError error
	BackupFileTicker *time.Ticker
}

type Destructor func()

func Init(opts *InitOptions) (*Storage, InitOutputs, Destructor) {
	storage := &Storage{
		gaugeMapRWM:   &sync.Mutex{},
		counterMapRWM: &sync.Mutex{},

		gaugeMap:   make(GaugeMetricsStorage),
		counterMap: make(CounterMetricsStorage),
	}

	initOutputs := InitOutputs{}

	if opts != nil && opts.InitialFilePath != nil {
		initOutputs.InitialFileError = createStorageFromBackup(storage, *opts.InitialFilePath)

		if initOutputs.InitialFileError == nil {
			fmt.Println("Storage is successfully restored from backup")
		} else {
			fmt.Println("Storage is not restored from backup,", initOutputs.InitialFileError)
		}
	}

	if opts != nil && opts.BackupFilePath != nil {
		backupFilePath := *opts.BackupFilePath
		err := withBackup(storage, backupFilePath, opts.BackupInterval)

		if err == nil && opts.BackupInterval != time.Duration(0) {
			initOutputs.BackupFileTicker = time.NewTicker(opts.BackupInterval)

			go func() {
				for {
					<-initOutputs.BackupFileTicker.C
					err := makeBackup(storage)

					if err != nil {
						fmt.Println("Storage can not create backup, ", err)
					}
				}
			}()
		}

		if err == nil {
			fmt.Println("Storage is connected with backup file,", backupFilePath)
		} else {
			fmt.Println("Storage is not connected with backup file,", backupFilePath, err)
		}
	}

	destructor := func() {
		if initOutputs.BackupFileTicker != nil {
			initOutputs.BackupFileTicker.Stop()
		}

		if storage.backupFile != nil {
			storage.backupFile.Close()
		}
	}

	return storage, initOutputs, destructor
}

func triggerBackup(stor *Storage) {
	if stor.backupFile != nil && stor.backupInterval == time.Duration(0) {
		err := makeBackup(stor)

		if err != nil {
			fmt.Println("Storage can not create backup, ", err)
		}
	}
}

func (stor *Storage) SetGaugeMetric(metricName string, value float64) {
	stor.gaugeMapRWM.Lock()
	stor.gaugeMap[metricName] = value
	stor.gaugeMapRWM.Unlock()

	triggerBackup(stor)
}

func (stor *Storage) GetGaugeMetric(metricName string) (float64, bool) {
	stor.gaugeMapRWM.Lock()
	defer stor.gaugeMapRWM.Unlock()

	value, ok := stor.gaugeMap[metricName]
	return value, ok
}

func (stor *Storage) ForEachGaugeMetric(handler func(metricName string, value float64)) {
	stor.gaugeMapRWM.Lock()
	defer stor.gaugeMapRWM.Unlock()

	for a, b := range stor.gaugeMap {
		handler(a, b)
	}
}

func (stor *Storage) GetCounterMetric(metricName string) (int64, bool) {
	stor.counterMapRWM.Lock()
	defer stor.counterMapRWM.Unlock()

	value, ok := stor.counterMap[metricName]
	return value, ok
}

func (stor *Storage) ForEachCounterMetric(handler func(metricName string, value int64)) {
	stor.counterMapRWM.Lock()
	defer stor.counterMapRWM.Unlock()

	for a, b := range stor.counterMap {
		handler(a, b)
	}
}

func (stor *Storage) SetCounterMetric(metricName string, count int64) {
	stor.counterMapRWM.Lock()
	stor.counterMap[metricName] += count
	stor.counterMapRWM.Unlock()

	triggerBackup(stor)
}
