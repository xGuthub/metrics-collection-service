package handler

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"

	"github.com/xGuthub/metrics-collection-service/internal/service"
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
		val = strconv.FormatFloat(v, 'g', -1, 64)
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

func (mh *MetricsHandler) HandleHomePage() (code int, body string) {
	gauges := mh.metricsService.AllGauges()
	counters := mh.metricsService.AllCounters()

	gNames := make([]string, 0, len(gauges))
	for name := range gauges {
		gNames = append(gNames, name)
	}
	sort.Strings(gNames)

	cNames := make([]string, 0, len(counters))
	for name := range counters {
		cNames = append(cNames, name)
	}
	sort.Strings(cNames)

	body = "<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>Metrics</title></head><body>"
	body += "<h1>Metrics</h1>"

	body += "<h2>Gauges</h2><ul>"
	if len(gNames) == 0 {
		body += "<li><em>No gauges</em></li>"
	} else {
		for _, name := range gNames {
			body += fmt.Sprintf("<li>%s: %s</li>", name, strconv.FormatFloat(gauges[name], 'g', -1, 64))
		}
	}
	body += "</ul>"

	body += "<h2>Counters</h2><ul>"
	if len(cNames) == 0 {
		body += "<li><em>No counters</em></li>"
	} else {
		for _, name := range cNames {
			body += fmt.Sprintf("<li>%s: %d</li>", name, counters[name])
		}
	}
	body += "</ul>"

	body += "</body></html>"

	return http.StatusOK, body
}
