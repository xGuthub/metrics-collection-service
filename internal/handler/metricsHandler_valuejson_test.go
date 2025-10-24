package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	models "github.com/xGuthub/metrics-collection-service/internal/model"
)

func TestValueJSONHandler_ContentType(t *testing.T) {
	h, _ := newTestHandler()

	// Wrong content type -> 415
	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBufferString(`{"id":"x","type":"gauge"}`))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	h.ValueJSONHandler(rr, req)
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected %d, got %d", http.StatusUnsupportedMediaType, rr.Code)
	}
	if got := rr.Body.String(); got != "unsupported media type: expected application/json" {
		t.Fatalf("unexpected body: %q", got)
	}

	// Empty content-type allowed
	req2 := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBufferString(`{"id":"x","type":"gauge"}`))
	rr2 := httptest.NewRecorder()
	h.ValueJSONHandler(rr2, req2)
	if rr2.Code != http.StatusNotFound { // x not seeded -> not found
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rr2.Code)
	}
}

func TestValueJSONHandler_BadJSON_And_MissingID_And_BadType(t *testing.T) {
	h, _ := newTestHandler()

	// Bad JSON
	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ValueJSONHandler(rr, req)
	if rr.Code != http.StatusBadRequest || rr.Body.String() != "bad value" {
		t.Fatalf("bad json: got %d %q", rr.Code, rr.Body.String())
	}

	// Missing ID
	req2 := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBufferString(`{"type":"gauge"}`))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	h.ValueJSONHandler(rr2, req2)
	if rr2.Code != http.StatusBadRequest || rr2.Body.String() != "bad value" {
		t.Fatalf("missing id: got %d %q", rr2.Code, rr2.Body.String())
	}

	// Bad metric type
	req3 := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBufferString(`{"id":"x","type":"unknown"}`))
	req3.Header.Set("Content-Type", "application/json")
	rr3 := httptest.NewRecorder()
	h.ValueJSONHandler(rr3, req3)
	if rr3.Code != http.StatusBadRequest || rr3.Body.String() != "bad metric type" {
		t.Fatalf("bad metric type: got %d %q", rr3.Code, rr3.Body.String())
	}
}

func TestValueJSONHandler_NotFound(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBufferString(`{"id":"nope","type":"gauge"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ValueJSONHandler(rr, req)
	if rr.Code != http.StatusNotFound || rr.Body.String() != "bad value" {
		t.Fatalf("expected 404 bad value, got %d %q", rr.Code, rr.Body.String())
	}
}

func TestValueJSONHandler_Success_Gauge(t *testing.T) {
	h, svc := newTestHandler()

	// Seed a gauge
	if err := svc.UpdateMetric(models.Gauge, "temp", "42.5"); err != nil {
		t.Fatalf("seed gauge: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBufferString(`{"id":"temp","type":"gauge"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ValueJSONHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("unexpected Content-Type: %q", ct)
	}
	var resp models.Metrics
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode resp: %v", err)
	}
	if resp.ID != "temp" || resp.MType != models.Gauge || resp.Value == nil || *resp.Value != 42.5 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Delta != nil {
		t.Fatalf("expected delta=nil, got %v", *resp.Delta)
	}
}

func TestValueJSONHandler_Success_Counter(t *testing.T) {
	h, svc := newTestHandler()

	// Seed a counter
	if err := svc.UpdateMetric(models.Counter, "requests", "10"); err != nil {
		t.Fatalf("seed counter: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBufferString(`{"id":"requests","type":"counter"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ValueJSONHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	var resp models.Metrics
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode resp: %v", err)
	}
	if resp.ID != "requests" || resp.MType != models.Counter || resp.Delta == nil || *resp.Delta != 10 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Value != nil {
		t.Fatalf("expected value=nil, got %v", *resp.Value)
	}
}
