package service

import (
	"errors"
	"math"
	"strconv"
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
}

func NewMetricsService(memStorage Storage) *MetricsService {
	return &MetricsService{
		storage: memStorage,
	}
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

	return nil
}
