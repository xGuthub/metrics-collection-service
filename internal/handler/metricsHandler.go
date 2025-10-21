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
		log.Printf("report gauge %s success: %v", name, val)
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

func (mh *MetricsHandler) HandleGet(mType, name string) (code int, result string) {
	var val string

	switch mType {
	case "gauge":
		v, exists := mh.metricsService.GetGauge(name)
		if !exists {
			return http.StatusNotFound, "gauge not found"
		}
		val = strconv.FormatFloat(v, 'f', 3, 64)
		log.Printf("get gauge %s success: %s", name, val)
	case "counter":
		v, exists := mh.metricsService.GetCounter(name)
		if !exists {
			return http.StatusNotFound, "counter not found"
		}
		val = strconv.FormatInt(v, 10)
		log.Printf("get counter %s success: %s", name, val)
	default:
		return http.StatusBadRequest, "bad metric type"
	}
	return http.StatusOK, val
}
