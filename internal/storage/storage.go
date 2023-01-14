package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type StorageMetric struct {
	ID    string
	MType string
	Delta int64
	Value float64
}

type StorageInterface interface {
	ForEachMetrics(context.Context, func(*StorageMetric)) error
	GetMetric(ctx context.Context, mType string, id string) (*StorageMetric, error)
	UpdateMetric(ctx context.Context, metric common.Metric) error
	UpdateMetrics(ctx context.Context, metricsList []common.Metric) error
	Ping(ctx context.Context) error
}

type StorageV2 struct {
	dbPool *pgxpool.Pool
}

const (
	// INSERT INTO metrics (id, mType, delta)
	// VALUES ($1, $2, $3)
	// ON CONFLICT (id) DO UPDATE SET delta = metrics.delta EXCLUDED.delta;
	insertDeltaSQL = "INSERT INTO metrics (id, mType, delta) " +
		"VALUES ($1, $2, $3) " +
		"ON CONFLICT (id) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta;"

	// INSERT INTO metrics (id, mType, value)
	// VALUES ($1, $2, $3)
	// ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value;"
	insertValueSQL = "INSERT INTO metrics (id, mType, value) " +
		"VALUES ($1, $2, $3) " +
		"ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value;"

	// SELECT delta FROM metrics WHERE id=$1
	selectDeltaSQL = "SELECT delta FROM metrics WHERE id=$1"

	// SELECT value FROM metrics WHERE id=$1
	selectValueSQL = "SELECT value FROM metrics WHERE id=$1"

	// SELECT id, mType, delta, value FROM metrics
	selectDeltaValueSQL = "SELECT id, mType, delta, value FROM metrics"
)

var ErrUnknowMetricType = errors.New("unknown metric type")

func newUnknownMetricTypeError(str string) error {
	return fmt.Errorf(`%w: %s`, ErrUnknowMetricType, str)
}

func InitV2(dbContext context.Context, connString string) (*StorageV2, error) {
	conn, err := pgxpool.Connect(dbContext, connString)
	if err != nil {
		return nil, err
	}

	log.Printf("Connected to DB %s successfully\n", connString)

	sql := "CREATE TABLE IF NOT EXISTS metrics (" +
		"id text UNIQUE, " +
		"mType text, " +
		"delta bigint DEFAULT 0, " +
		"value double precision DEFAULT 0" +
		");"

	_, err = conn.Exec(context.TODO(), sql)
	if err != nil {
		return nil, err
	}

	log.Println("Created metrics Table successfully")

	return &StorageV2{dbPool: conn}, nil
}

// Ping checks if the connection to database established
func (stor *StorageV2) Ping(ctx context.Context) error {
	return stor.dbPool.Ping(ctx)
}

func (stor *StorageV2) Close() {
	stor.dbPool.Close()
}

// ForEachMetrics passes through all metrics in database and call handler
func (stor *StorageV2) ForEachMetrics(ctx context.Context, handler func(*StorageMetric)) error {
	rows, err := stor.dbPool.Query(context.Background(), selectDeltaValueSQL)
	if err != nil {
		return err
	}

	for rows.Next() {
		storageMetric := &StorageMetric{}

		err := rows.Scan(&storageMetric.ID, &storageMetric.MType, &storageMetric.Delta, &storageMetric.Value)
		if err != nil {
			return err
		}

		handler(storageMetric)
	}

	return nil
}

func (stor *StorageV2) GetMetric(ctx context.Context, mType string, id string) (*StorageMetric, error) {
	storageMetric := &StorageMetric{
		MType: mType,
		ID:    id,
	}

	var err error

	switch mType {
	case common.GaugeMetricName:
		err = stor.dbPool.QueryRow(ctx, selectValueSQL, id).
			Scan(&storageMetric.Value)
	case common.CounterMetricName:
		err = stor.dbPool.QueryRow(ctx, selectDeltaSQL, id).
			Scan(&storageMetric.Delta)
	default:
		err = newUnknownMetricTypeError(mType)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return storageMetric, nil
}

func (stor *StorageV2) UpdateMetric(ctx context.Context, metric common.Metric) error {
	var err error

	switch metric.MType {
	case common.GaugeMetricName:
		_, err = stor.dbPool.Exec(ctx, insertValueSQL, metric.ID, metric.MType, *metric.Value)
	case common.CounterMetricName:
		_, err = stor.dbPool.Exec(ctx, insertDeltaSQL, metric.ID, metric.MType, *metric.Delta)
	default:
		err = newUnknownMetricTypeError(metric.MType)
	}

	return err
}

func (stor *StorageV2) UpdateMetrics(ctx context.Context, metricsList []common.Metric) error {
	tx, err := stor.dbPool.Begin(ctx)
	if err != nil {
		return err
	}

	for _, metric := range metricsList {
		switch metric.MType {
		case common.GaugeMetricName:
			_, err = tx.Exec(ctx, insertValueSQL, metric.ID, metric.MType, *metric.Value)
		case common.CounterMetricName:
			_, err = tx.Exec(ctx, insertDeltaSQL, metric.ID, metric.MType, *metric.Delta)
		default:
			return tx.Rollback(ctx)
		}

		if err != nil {
			return tx.Rollback(ctx)
		}
	}

	return tx.Commit(ctx)
}

type GaugeMetricsStorage map[string]float64
type CounterMetricsStorage map[string]int64

type Storage struct {
	gaugeMap   GaugeMetricsStorage
	counterMap CounterMetricsStorage
	storageRWM sync.RWMutex
}

// ForEachMetrics passes through all metrics in database and call handler
func (stor *Storage) ForEachMetrics(ctx context.Context, handler func(*StorageMetric)) error {
	stor.storageRWM.RLock()
	defer stor.storageRWM.RUnlock()

	for id, value := range stor.gaugeMap {
		handler(&StorageMetric{
			ID:    id,
			MType: common.GaugeMetricName,
			Value: value,
		})
	}
	for id, delta := range stor.counterMap {
		handler(&StorageMetric{
			ID:    id,
			MType: common.CounterMetricName,
			Delta: delta,
		})
	}

	return nil
}

func (stor *Storage) GetMetric(ctx context.Context, mType string, id string) (*StorageMetric, error) {
	stor.storageRWM.RLock()
	defer stor.storageRWM.RUnlock()

	storageMetric := &StorageMetric{
		MType: mType,
		ID:    id,
	}

	switch mType {
	case common.GaugeMetricName:
		if value, ok := stor.gaugeMap[id]; ok {
			storageMetric.Value = value
			return storageMetric, nil
		}
	case common.CounterMetricName:
		if delta, ok := stor.counterMap[id]; ok {
			storageMetric.Delta = delta
			return storageMetric, nil
		}
	default:
		return nil, newUnknownMetricTypeError(mType)
	}

	return nil, nil
}

func (stor *Storage) UpdateMetric(ctx context.Context, metric common.Metric) error {
	stor.storageRWM.Lock()
	defer stor.storageRWM.Unlock()

	switch metric.MType {
	case common.GaugeMetricName:
		stor.gaugeMap[metric.ID] = *metric.Value
	case common.CounterMetricName:
		stor.counterMap[metric.ID] += *metric.Delta
	default:
		return newUnknownMetricTypeError(metric.MType)
	}

	return nil
}

func (stor *Storage) UpdateMetrics(ctx context.Context, metricsList []common.Metric) error {
	stor.storageRWM.Lock()
	defer stor.storageRWM.Unlock()

	gaugeMap := make(GaugeMetricsStorage)
	counterMap := make(CounterMetricsStorage)

	for _, metric := range metricsList {
		switch metric.MType {
		case common.GaugeMetricName:
			gaugeMap[metric.ID] = *metric.Value
		case common.CounterMetricName:
			counterMap[metric.ID] = *metric.Delta
		default:
			return newUnknownMetricTypeError(metric.MType)
		}
	}

	for key, value := range gaugeMap {
		stor.gaugeMap[key] = value
	}
	for key, delta := range counterMap {
		stor.counterMap[key] += delta
	}

	return nil
}

func (stor *Storage) Ping(ctx context.Context) error {
	return nil
}

type BackupStorageWrapper struct {
	*Storage
	backupFilePath string
	fileRWM        sync.Mutex
}

type BackupObject struct {
	GaugeMetrics   GaugeMetricsStorage
	CounterMetrics CounterMetricsStorage
}

func writeStoreBackup(stor *Storage, backupFilePath string) error {
	stor.storageRWM.RLock()

	backup := BackupObject{
		GaugeMetrics:   stor.gaugeMap,
		CounterMetrics: stor.counterMap,
	}

	backupBytes, _ := json.Marshal(&backup)

	stor.storageRWM.RUnlock()

	return os.WriteFile(backupFilePath, backupBytes, 0644)
}

func (stor *BackupStorageWrapper) UpdateMetric(ctx context.Context, metric common.Metric) error {
	err := stor.Storage.UpdateMetric(ctx, metric)
	if err != nil {
		return err
	}

	stor.fileRWM.Lock()
	defer stor.fileRWM.Unlock()

	writeStoreBackup(stor.Storage, stor.backupFilePath)

	return nil
}

func createStorageFromBackup(stor *Storage, initialFilePath string) error {
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
		stor.counterMap = backupObject.CounterMetrics
		stor.gaugeMap = backupObject.GaugeMetrics
	}

	return err
}

func Init(initialFilePath *string) (*Storage, error) {
	stor := &Storage{
		gaugeMap:   make(GaugeMetricsStorage),
		counterMap: make(CounterMetricsStorage),
	}

	var err error

	if initialFilePath != nil {
		err = createStorageFromBackup(stor, *initialFilePath)

		if err == nil {
			log.Println("Storage is successfully restored from backup")
		} else {
			log.Println("Storage is not restored from backup,", err)
		}
	}

	return stor, err
}

func WithBackup(stor *Storage, backupFilePath string) StorageInterface {
	return &BackupStorageWrapper{
		backupFilePath: backupFilePath,
		Storage:        stor,
	}
}

type Empty struct{}

func InitBackupTicker(stor *Storage, backupFilePath string, backupInterval time.Duration) func() {
	ticker := time.NewTicker(backupInterval)
	doneFlag := make(chan Empty)

	stopTicker := func() {
		doneFlag <- Empty{}
	}

	go func() {
		for {
			select {
			case <-doneFlag:
				return
			case <-ticker.C:
				err := writeStoreBackup(stor, backupFilePath)

				if err != nil {
					log.Println("Could not create backup", err)
				}
			}
		}
	}()

	return stopTicker
}

type MockStorage struct {
	StorageInterface

	ForEachMetricsResponse error
	ForEachMetricsArr      []*StorageMetric

	GetMetricResponse      *StorageMetric
	GetMetricErrorResponse error

	UpdateMetricResponse error

	UpdateMetricsResponse error
}

func (s *MockStorage) ForEachMetrics(ctx context.Context, h func(*StorageMetric)) error {
	for _, s := range s.ForEachMetricsArr {
		h(s)
	}

	return s.ForEachMetricsResponse
}

func (s *MockStorage) GetMetric(ctx context.Context, mType string, id string) (*StorageMetric, error) {
	return s.GetMetricResponse, s.GetMetricErrorResponse
}

func (s *MockStorage) UpdateMetric(ctx context.Context, metric common.Metric) error {
	return s.UpdateMetricResponse
}

func (s *MockStorage) UpdateMetrics(ctx context.Context, metricsList []common.Metric) error {
	return s.UpdateMetricsResponse
}

func (s *MockStorage) Ping(ctx context.Context) error {
	return nil
}
