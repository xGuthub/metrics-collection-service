package service

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/xGuthub/metrics-collection-service/internal/repository"
)

type Storage interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, delta int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	AllGauges() map[string]float64
	AllCounters() map[string]int64
}

type MetricsService struct {
	storage Storage
	// persistence config
	persistPath   string
	storeInterval time.Duration
	restore       bool
	stateStore    repository.StateStore
}

func NewMetricsService(memStorage Storage) *MetricsService {
	return &MetricsService{
		storage: memStorage,
	}
}

// PersistenceConfig describes how and where to persist metrics state.
type PersistenceConfig struct {
	FilePath      string
	StoreInterval time.Duration
	Restore       bool
}

// ConfigurePersistence sets up persistence options. Can be called once on boot.
func (ms *MetricsService) ConfigurePersistence(cfg PersistenceConfig) {
	ms.persistPath = cfg.FilePath
	ms.storeInterval = cfg.StoreInterval
	ms.restore = cfg.Restore
}

// SetStateStore injects the repository responsible for persisting state.
func (ms *MetricsService) SetStateStore(store repository.StateStore) {
	ms.stateStore = store
}

func (ms *MetricsService) AllGauges() map[string]float64 {
	return ms.storage.AllGauges()
}

func (ms *MetricsService) AllCounters() map[string]int64 {
	return ms.storage.AllCounters()
}

func (ms *MetricsService) GetMetric(mType, name string) (string, error) {
	var val string

	switch mType {
	case "gauge":
		v, exists := ms.storage.GetGauge(name)
		if !exists {
			return "", errors.New("not found")
		}
		val = strconv.FormatFloat(v, 'g', -1, 64)
	case "counter":
		v, exists := ms.storage.GetCounter(name)
		if !exists {
			return "", errors.New("not found")
		}
		val = strconv.FormatInt(v, 10)
	default:
		return "", errors.New("bad metric type")
	}

	return val, nil
}

func (ms *MetricsService) UpdateMetric(mType, name, val string) error {
	switch mType {
	case "gauge":
		val, err := strconv.ParseFloat(val, 64)
		if err != nil || math.IsNaN(val) || math.IsInf(val, 0) {
			return errors.New("bad value")
		}
		ms.storage.UpdateGauge(name, val)
	case "counter":
		delta, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return errors.New("bad value")
		}
		ms.storage.UpdateCounter(name, delta)
	default:
		return errors.New("bad metric type")
	}
	// Immediate save when store interval is zero and path configured.
	if ms.storeInterval == 0 && ms.persistPath != "" {
		_ = ms.SaveState()
	}
	return nil
}

// StartAutoSave launches periodic persistence if StoreInterval > 0.
// onError is optional; if provided, it receives save errors.
func (ms *MetricsService) StartAutoSave(ctx context.Context, onError func(error)) {
	if ms.storeInterval <= 0 || ms.persistPath == "" {
		return
	}
	ticker := time.NewTicker(ms.storeInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := ms.SaveState(); err != nil && onError != nil {
					onError(err)
				}
			}
		}
	}()
}

// SaveState persists current storage state via injected repository.
func (ms *MetricsService) SaveState() error {
	if ms.persistPath == "" || ms.stateStore == nil {
		return nil
	}
	return ms.stateStore.Save(ms.persistPath, ms.storage.AllGauges(), ms.storage.AllCounters())
}

// RestoreState loads persisted state via injected repository.
func (ms *MetricsService) RestoreState() error {
	if !ms.restore || ms.persistPath == "" || ms.stateStore == nil {
		return nil
	}
	gauges, counters, err := ms.stateStore.Load(ms.persistPath)
	if err != nil {
		return err
	}
	for k, v := range gauges {
		ms.storage.UpdateGauge(k, v)
	}
	for k, v := range counters {
		ms.storage.UpdateCounter(k, v)
	}
	return nil
}
