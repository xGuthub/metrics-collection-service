package handler

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

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

func (mh *MetricsHandler) HomeHandler(w http.ResponseWriter, _ *http.Request) {
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

	body := "<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>Metrics</title></head><body>"
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

	writeHTML(w, http.StatusOK, body)
}

func (mh *MetricsHandler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if err := validateContentType(r, "text/plain"); err != nil {
		writePlain(w, http.StatusUnsupportedMediaType, "unsupported media type: expected text/plain")

		return
	}

	mType, name, rawVal, err := parsePath(r.URL.Path)
	if err != nil {
		if err.Error() == "missing name" {
			writePlain(w, http.StatusNotFound, "metric name is required")

			return
		}
		writePlain(w, http.StatusNotFound, "not found")

		return
	}

	err = mh.metricsService.UpdateMetric(mType, name, rawVal)

	if err != nil {
		if err.Error() == "bad value" {
			writePlain(w, http.StatusBadRequest, "bad value")

			return
		}
		if err.Error() == "bad metric type" {
			writePlain(w, http.StatusBadRequest, "bad metric type")

			return
		}
	}

	writePlain(w, http.StatusOK, "OK")
}

func (mh *MetricsHandler) ValueHandler(w http.ResponseWriter, r *http.Request) {
	mType, name, _, err := parsePath(r.URL.Path)
	if err != nil {
		if err.Error() == "missing name" {
			writePlain(w, http.StatusNotFound, "metric name is required")

			return
		}
		writePlain(w, http.StatusNotFound, "not found")

		return
	}

	val, err := mh.metricsService.GetMetric(mType, name)

	if err != nil {
		if err.Error() == "not found" {
			writePlain(w, http.StatusNotFound, "bad value")

			return
		}
		if err.Error() == "bad metric type" {
			writePlain(w, http.StatusBadRequest, "bad metric type")

			return
		}
	}

	writePlain(w, http.StatusOK, val)
}

func writeHTML(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(msg))
}

func writePlain(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(msg))
}

func validateContentType(r *http.Request, contType string) error {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		// Допускаем пустой Content-Type (многие агенты так делают).
		return nil
	}
	// Разрешаем только text/plain (+ возможный charset).
	if !strings.HasPrefix(strings.ToLower(ct), contType) {
		return errors.New("unsupported media type")
	}

	return nil
}

// parsePath ожидает строго /update/{type}/{name}/{value}.
func parsePath(path string) (mType, name, value string, err error) {
	// Исключаем возможные лишние слэши в конце без редиректов.
	trimmed := strings.TrimSuffix(path, "/")
	parts := strings.Split(trimmed, "/")
	// Примеры:
	// "" "update" "{type}" "{name}" "{value}"  -> len=5
	if len(parts) != 4 && len(parts) != 5 {
		return "", "", "", errors.New("not found")
	}

	mType = parts[2]
	name = parts[3]
	if len(parts) == 5 {
		value = parts[4]
	}

	if name == "" {
		// Специальный кейс из задания — 404 при отсутствии имени
		return "", "", "", errors.New("missing name")
	}

	return mType, name, value, nil
}
