package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"
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
		_ = ms.SaveToFile()
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
				if err := ms.SaveToFile(); err != nil && onError != nil {
					onError(err)
				}
			}
		}
	}()
}

// SaveToFile writes current storage state to configured file.
func (ms *MetricsService) SaveToFile() error {
	if ms.persistPath == "" {
		return nil
	}
	dump := struct {
		Gauges   map[string]float64 `json:"gauges"`
		Counters map[string]int64   `json:"counters"`
	}{
		Gauges:   ms.storage.AllGauges(),
		Counters: ms.storage.AllCounters(),
	}

	data, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(ms.persistPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	// Write atomically: write to temp file then rename.
	tmp := ms.persistPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, ms.persistPath)
}

// RestoreFromFile loads state from configured file if present and allowed.
func (ms *MetricsService) RestoreFromFile() error {
	if !ms.restore || ms.persistPath == "" {
		return nil
	}
	data, err := os.ReadFile(ms.persistPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var dump struct {
		Gauges   map[string]float64 `json:"gauges"`
		Counters map[string]int64   `json:"counters"`
	}
	if err := json.Unmarshal(data, &dump); err != nil {
		return err
	}
	for k, v := range dump.Gauges {
		ms.storage.UpdateGauge(k, v)
	}
	for k, v := range dump.Counters {
		// UpdateCounter increments; storage starts empty, so this sets absolute value.
		ms.storage.UpdateCounter(k, v)
	}
	return nil
}
