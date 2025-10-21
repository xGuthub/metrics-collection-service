package service

type Storage interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, delta int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
}

type MetricsService struct {
	storage Storage
}

func NewMetricsService(memStorage Storage) *MetricsService {
	return &MetricsService{
		storage: memStorage,
	}
}

func (ms *MetricsService) IncrementCounter(name string, delta int64) {
	ms.storage.UpdateCounter(name, delta)
}

func (ms *MetricsService) UpdateGauge(name string, val float64) {
	ms.storage.UpdateGauge(name, val)
}

func (ms *MetricsService) GetCounter(name string) (int64, bool) {
	return ms.storage.GetCounter(name)
}

func (ms *MetricsService) GetGauge(name string) (float64, bool) {
	return ms.storage.GetGauge(name)
}
