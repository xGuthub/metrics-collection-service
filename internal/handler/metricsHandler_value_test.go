package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValueHandler_PathErrors(t *testing.T) {
	h, _ := newTestHandler()

	// Missing name -> 404 with specific message
	req := httptest.NewRequest(http.MethodGet, "/value/gauge//10", nil)
	rr := httptest.NewRecorder()
	h.ValueHandler(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
	if got := rr.Body.String(); got != "metric name is required" {
		t.Fatalf("unexpected body: %q", got)
	}

	// Not enough parts -> generic not found
	req2 := httptest.NewRequest(http.MethodGet, "/value/gauge", nil)
	rr2 := httptest.NewRecorder()
	h.ValueHandler(rr2, req2)
	if rr2.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr2.Code)
	}
	if got := rr2.Body.String(); got != "not found" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestValueHandler_BadType_And_NotFound(t *testing.T) {
	h, _ := newTestHandler()

	// Bad metric type
	req := httptest.NewRequest(http.MethodGet, "/value/unknown/temp", nil)
	rr := httptest.NewRecorder()
	h.ValueHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if got := rr.Body.String(); got != "bad metric type" {
		t.Fatalf("unexpected body: %q", got)
	}

	// Not found value -> 404 "bad value"
	req2 := httptest.NewRequest(http.MethodGet, "/value/gauge/not_exist", nil)
	rr2 := httptest.NewRecorder()
	h.ValueHandler(rr2, req2)
	if rr2.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr2.Code)
	}
	if got := rr2.Body.String(); got != "bad value" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestValueHandler_Success(t *testing.T) {
	h, svc := newTestHandler()

	// Seed values via service
	if err := svc.UpdateMetric("gauge", "temp", "42.5"); err != nil {
		t.Fatalf("seed gauge: %v", err)
	}
	if err := svc.UpdateMetric("counter", "requests", "10"); err != nil {
		t.Fatalf("seed counter: %v", err)
	}

	// Gauge success
	req := httptest.NewRequest(http.MethodGet, "/value/gauge/temp", nil)
	rr := httptest.NewRecorder()
	h.ValueHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if got := rr.Body.String(); got != "42.5" {
		t.Fatalf("unexpected body: %q", got)
	}

	// Counter success
	req2 := httptest.NewRequest(http.MethodGet, "/value/counter/requests", nil)
	rr2 := httptest.NewRecorder()
	h.ValueHandler(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr2.Code)
	}
	if got := rr2.Body.String(); got != "10" {
		t.Fatalf("unexpected body: %q", got)
	}
}
