package handler

type MetricsService struct {
	storage *MemStorage
}

func NewMetricsService() *MetricsService {
	return &MetricsService{
		storage: NewMemStorage(),
	}
}

func (ms *MetricsService) IncrementCounter(name string, delta int64) {
	ms.storage.UpdateCounter(name, delta)
}

func (ms *MetricsService) UpdateGauge(name string, val float64) {
	ms.storage.UpdateGauge(name, val)
}
