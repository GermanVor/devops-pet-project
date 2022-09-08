package storage

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/GermanVor/devops-pet-project/internal/common"
	"github.com/jackc/pgtype"
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
	UpdateMetric(ctx context.Context, metric common.Metrics) error
}

type StorageV2 struct {
	StorageInterface
	dbPool *pgxpool.Pool
}

// go run ./cmd/server/main.go -d=postgres://zzman:@localhost:5432/postgres
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
		"value double precision" +
		");"

	_, err = conn.Exec(context.TODO(), sql)
	if err != nil {
		return nil, err
	}

	log.Println("Created metrics Table successfully")

	return &StorageV2{dbPool: conn}, nil
}

func (stor *StorageV2) GetConn() *pgxpool.Pool {
	return stor.dbPool
}

func (stor *StorageV2) Close() {
	stor.dbPool.Close()
}

func (stor *StorageV2) ForEachMetrics(ctx context.Context, handler func(*StorageMetric)) error {
	rows, err := stor.dbPool.Query(context.Background(), "SELECT id, mType, delta, value FROM metrics")
	if err != nil {
		return err
	}

	value := pgtype.Float8{}

	for rows.Next() {
		storageMetric := &StorageMetric{}

		err := rows.Scan(&storageMetric.ID, &storageMetric.MType, &storageMetric.Delta, &value)
		if err != nil {
			return err
		}

		switch storageMetric.MType {
		case common.GaugeMetricName:
			if value.Status == pgtype.Present {
				value.AssignTo(&storageMetric.Value)
				handler(storageMetric)
			}
		case common.CounterMetricName:
			handler(storageMetric)
		}
	}

	return nil
}

func (stor *StorageV2) GetMetric(ctx context.Context, mType string, id string) (*StorageMetric, error) {
	storageMetric := &StorageMetric{
		MType: mType,
		ID:    id,
	}

	switch mType {
	case common.GaugeMetricName:
		row := stor.dbPool.QueryRow(ctx, "SELECT value FROM metrics WHERE id=$1", id)
		value := pgtype.Float8{}

		err := row.Scan(&value)
		if err != nil && err != pgx.ErrNoRows {
			return nil, err
		}

		if value.Status == pgtype.Present {
			value.AssignTo(&storageMetric.Value)
			return storageMetric, nil
		}
	case common.CounterMetricName:
		row := stor.dbPool.QueryRow(ctx, "SELECT delta FROM metrics WHERE id=$1", id)
		delta := pgtype.Int8{}

		err := row.Scan(&delta)
		if err != nil && err != pgx.ErrNoRows {
			return nil, err
		}

		if delta.Status == pgtype.Present {
			delta.AssignTo(&storageMetric.Delta)
			return storageMetric, nil
		}
	default:
		return nil, errors.New("unknown metric type: " + mType)
	}

	return nil, nil
}

func (stor *StorageV2) UpdateMetric(ctx context.Context, metric common.Metrics) error {
	switch metric.MType {
	case common.GaugeMetricName:
		sql := "INSERT INTO metrics (id, mType, value) " +
			"VALUES ($1, $2, $3) " +
			"ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value;"

		_, err := stor.dbPool.Exec(ctx, sql, metric.ID, metric.MType, *metric.Value)
		if err != nil {
			return err
		}
	case common.CounterMetricName:
		sql := "INSERT INTO metrics (id, mType, delta) " +
			"VALUES ($1, $2, $3) " +
			"ON CONFLICT (id) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta;"

		_, err := stor.dbPool.Exec(ctx, sql, metric.ID, metric.MType, *metric.Delta)
		if err != nil {
			return err
		}
	default:
		return errors.New("unknown metric type: " + metric.MType)
	}

	return nil
}

type GaugeMetricsStorage map[string]float64
type CounterMetricsStorage map[string]int64

type Storage struct {
	StorageInterface

	gaugeMap   GaugeMetricsStorage
	counterMap CounterMetricsStorage
	storageRWM sync.RWMutex
}

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
		return nil, errors.New("unknown metric type: " + mType)
	}

	return nil, nil
}

func (stor *Storage) UpdateMetric(ctx context.Context, metric common.Metrics) error {
	stor.storageRWM.Lock()
	defer stor.storageRWM.Unlock()

	switch metric.MType {
	case common.GaugeMetricName:
		stor.gaugeMap[metric.ID] = *metric.Value
	case common.CounterMetricName:
		stor.counterMap[metric.ID] += *metric.Delta
	default:
		return errors.New("unknown metric type: " + metric.MType)
	}

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

func (stor *BackupStorageWrapper) UpdateMetric(ctx context.Context, metric common.Metrics) error {
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
