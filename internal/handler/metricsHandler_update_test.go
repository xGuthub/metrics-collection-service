package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xGuthub/metrics-collection-service/internal/repository"
	"github.com/xGuthub/metrics-collection-service/internal/service"
)

// Helper to create handler with in-memory storage
func newTestHandler() (*MetricsHandler, *service.MetricsService) {
	storage := repository.NewMemStorage()
	svc := service.NewMetricsService(storage)
	h := NewMetricsHandler(svc)
	return h, svc
}

func TestUpdateHandler_ContentType(t *testing.T) {
	h, _ := newTestHandler()

	// Wrong content-type
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/temp/1", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.UpdateHandler(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status %d, got %d", http.StatusUnsupportedMediaType, rr.Code)
	}
	if got := rr.Body.String(); got != "unsupported media type: expected text/plain" {
		t.Fatalf("unexpected body: %q", got)
	}

	// Empty content-type is allowed
	req2 := httptest.NewRequest(http.MethodPost, "/update/gauge/temp/2", nil)
	rr2 := httptest.NewRecorder()

	h.UpdateHandler(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr2.Code)
	}
	if got := rr2.Body.String(); got != "OK" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestUpdateHandler_PathErrors(t *testing.T) {
	h, _ := newTestHandler()

	// Missing name should return 404 with specific message
	req := httptest.NewRequest(http.MethodPost, "/update/gauge//10", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	h.UpdateHandler(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
	if got := rr.Body.String(); got != "metric name is required" {
		t.Fatalf("unexpected body: %q", got)
	}

	// Not enough path parts -> generic not found
	req2 := httptest.NewRequest(http.MethodPost, "/update/gauge", nil)
	req2.Header.Set("Content-Type", "text/plain")
	rr2 := httptest.NewRecorder()
	h.UpdateHandler(rr2, req2)
	if rr2.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr2.Code)
	}
	if got := rr2.Body.String(); got != "not found" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestUpdateHandler_BadValueAndType(t *testing.T) {
	h, _ := newTestHandler()

	// Bad value for gauge
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/temp/not-a-number", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	h.UpdateHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if got := rr.Body.String(); got != "bad value" {
		t.Fatalf("unexpected body: %q", got)
	}

	// Bad metric type
	req2 := httptest.NewRequest(http.MethodPost, "/update/unknown/temp/1", nil)
	req2.Header.Set("Content-Type", "text/plain")
	rr2 := httptest.NewRecorder()
	h.UpdateHandler(rr2, req2)
	if rr2.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr2.Code)
	}
	if got := rr2.Body.String(); got != "bad metric type" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestUpdateHandler_Success(t *testing.T) {
	h, svc := newTestHandler()

	// Gauge update
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/temp/42.5", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	h.UpdateHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if got := rr.Body.String(); got != "OK" {
		t.Fatalf("unexpected body: %q", got)
	}
	gauges := svc.AllGauges()
	if v, ok := gauges["temp"]; !ok || v != 42.5 {
		t.Fatalf("gauge not updated: got (%v, %v)", v, ok)
	}

	// Counter update
	req2 := httptest.NewRequest(http.MethodPost, "/update/counter/requests/10", nil)
	req2.Header.Set("Content-Type", "text/plain")
	rr2 := httptest.NewRecorder()
	h.UpdateHandler(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr2.Code)
	}
	if got := rr2.Body.String(); got != "OK" {
		t.Fatalf("unexpected body: %q", got)
	}
	counters := svc.AllCounters()
	if v, ok := counters["requests"]; !ok || v != 10 {
		t.Fatalf("counter not updated: got (%v, %v)", v, ok)
	}
}
