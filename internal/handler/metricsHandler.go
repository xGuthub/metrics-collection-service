package handler

import (
	"github.com/xGuthub/metrics-collection-service/internal/service"
	"log"
	"math"
	"net/http"
	"strconv"
)

type MetricsHandler struct {
	metricsService *service.MetricsService
}

func NewMetricsHandler(metricsService *service.MetricsService) *MetricsHandler {
	return &MetricsHandler{
		metricsService: metricsService,
	}
}

func (mh *MetricsHandler) HandleUpdate(mType, name, rawVal string) (code int, status string) {
	switch mType {
	case "gauge":
		val, err := strconv.ParseFloat(rawVal, 64)
		if err != nil || math.IsNaN(val) || math.IsInf(val, 0) {
			return http.StatusBadRequest, "bad gauge value"
		}
		mh.metricsService.UpdateGauge(name, val)
		log.Printf("report counter %s success: %v", name, val)
	case "counter":
		delta, err := strconv.ParseInt(rawVal, 10, 64)
		if err != nil {
			return http.StatusBadRequest, "bad counter value"
		}
		mh.metricsService.IncrementCounter(name, delta)
		log.Printf("report counter %s success: %d", name, delta)
	default:
		return http.StatusBadRequest, "bad metric type"
	}
	return http.StatusOK, "OK"
}
