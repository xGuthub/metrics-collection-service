package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/xGuthub/metrics-collection-service/internal/handler"
)

type Server struct {
	mHandler *handler.MetricsHandler
}

func NewServer() *Server {
	return &Server{
		mHandler: handler.NewMetricsHandler(),
	}
}

// writePlain — единообразная отправка текстового ответа.
func writePlain(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(msg))
}

// validateContentType проверяет Content-Type, если он передан.
func validateContentType(r *http.Request) error {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		// Допускаем пустой Content-Type (многие агенты так делают).
		return nil
	}
	// Разрешаем только text/plain (+ возможный charset).
	if !strings.HasPrefix(strings.ToLower(ct), "text/plain") {
		return errors.New("unsupported media type")
	}
	return nil
}

// parsePath ожидает строго /update/{type}/{name}/{value}
func parsePath(path string) (mType, name, value string, err error) {
	// Исключаем возможные лишние слэши в конце без редиректов.
	trimmed := strings.TrimSuffix(path, "/")
	parts := strings.Split(trimmed, "/")
	// Примеры:
	// "" "update" "{type}" "{name}" "{value}"  -> len=5
	if len(parts) != 5 || parts[1] != "update" {
		return "", "", "", errors.New("not found")
	}

	mType = parts[2]
	name = parts[3]
	value = parts[4]

	if name == "" {
		// Специальный кейс из задания — 404 при отсутствии имени
		return "", "", "", errors.New("missing name")
	}
	return mType, name, value, nil
}

func (s *Server) serveUpdate(w http.ResponseWriter, r *http.Request) {
	// Метод
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writePlain(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Content-Type
	if err := validateContentType(r); err != nil {
		writePlain(w, http.StatusUnsupportedMediaType, "unsupported media type: expected text/plain")
		return
	}

	// Путь
	mType, name, rawVal, err := parsePath(r.URL.Path)
	if err != nil {
		if err.Error() == "missing name" {
			writePlain(w, http.StatusNotFound, "metric name is required")
			return
		}
		writePlain(w, http.StatusNotFound, "not found")
		return
	}

	code, status := s.mHandler.HandleUpdate(mType, name, rawVal)

	writePlain(w, code, status)
}

// rootHandler — единая точка входа без шаблонов ServeMux, чтобы исключить редиректы.
func (s *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	// Обрабатываем только /update/... остальное — 404
	if strings.HasPrefix(r.URL.Path, "/update/") || r.URL.Path == "/update" || r.URL.Path == "/update/" {
		s.serveUpdate(w, r)
		return
	}
	writePlain(w, http.StatusNotFound, "not found")
}
