package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	models "github.com/xGuthub/metrics-collection-service/internal/model"
)

func TestUpdateJSONHandler_ContentType(t *testing.T) {
	h, _ := newTestHandler()

	// Wrong content-type
	body := bytes.NewBufferString(`{"id":"t","type":"gauge","value":1.5}`)
	req := httptest.NewRequest(http.MethodPost, "/update/", body)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	h.UpdateJSONHandler(rr, req)
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status %d, got %d", http.StatusUnsupportedMediaType, rr.Code)
	}
	if got := rr.Body.String(); got != "unsupported media type: expected application/json" {
		t.Fatalf("unexpected body: %q", got)
	}

	// Empty content-type is allowed
	body2 := bytes.NewBufferString(`{"id":"t","type":"gauge","value":2.5}`)
	req2 := httptest.NewRequest(http.MethodPost, "/update/", body2)
	rr2 := httptest.NewRecorder()
	h.UpdateJSONHandler(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr2.Code)
	}
	if ct := rr2.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("unexpected Content-Type: %q", ct)
	}
}

func TestUpdateJSONHandler_BadJSON_And_MissingFields(t *testing.T) {
	h, _ := newTestHandler()

	// Bad JSON
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.UpdateJSONHandler(rr, req)
	if rr.Code != http.StatusBadRequest || rr.Body.String() != "bad value" {
		t.Fatalf("bad json: got %d %q", rr.Code, rr.Body.String())
	}

	// Missing ID
	req2 := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(`{"type":"gauge","value":1}`))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	h.UpdateJSONHandler(rr2, req2)
	if rr2.Code != http.StatusBadRequest || rr2.Body.String() != "bad value" {
		t.Fatalf("missing id: got %d %q", rr2.Code, rr2.Body.String())
	}

	// Unknown type
	req3 := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(`{"id":"x","type":"unknown","value":1}`))
	req3.Header.Set("Content-Type", "application/json")
	rr3 := httptest.NewRecorder()
	h.UpdateJSONHandler(rr3, req3)
	if rr3.Code != http.StatusBadRequest || rr3.Body.String() != "bad metric type" {
		t.Fatalf("unknown type: got %d %q", rr3.Code, rr3.Body.String())
	}

	// Gauge without value
	req4 := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(`{"id":"g1","type":"gauge"}`))
	req4.Header.Set("Content-Type", "application/json")
	rr4 := httptest.NewRecorder()
	h.UpdateJSONHandler(rr4, req4)
	if rr4.Code != http.StatusBadRequest || rr4.Body.String() != "bad value" {
		t.Fatalf("gauge w/o value: got %d %q", rr4.Code, rr4.Body.String())
	}

	// Counter without delta
	req5 := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(`{"id":"c1","type":"counter"}`))
	req5.Header.Set("Content-Type", "application/json")
	rr5 := httptest.NewRecorder()
	h.UpdateJSONHandler(rr5, req5)
	if rr5.Code != http.StatusBadRequest || rr5.Body.String() != "bad value" {
		t.Fatalf("counter w/o delta: got %d %q", rr5.Code, rr5.Body.String())
	}
}

func TestUpdateJSONHandler_Success_Gauge(t *testing.T) {
	h, svc := newTestHandler()

	val := 42.5
	reqBody := models.Metrics{ID: "temp", MType: models.Gauge, Value: &val}
	buf, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(buf))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.UpdateJSONHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("unexpected Content-Type: %q", ct)
	}

	var resp models.Metrics
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != "temp" || resp.MType != models.Gauge || resp.Value == nil || *resp.Value != 42.5 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Delta != nil {
		t.Fatalf("expected delta to be nil, got %v", *resp.Delta)
	}

	gauges := svc.AllGauges()
	if v, ok := gauges["temp"]; !ok || v != 42.5 {
		t.Fatalf("gauge not updated: got (%v, %v)", v, ok)
	}
}

func TestUpdateJSONHandler_Success_Counter(t *testing.T) {
	h, svc := newTestHandler()

	d := int64(10)
	reqBody := models.Metrics{ID: "requests", MType: models.Counter, Delta: &d}
	buf, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(buf))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.UpdateJSONHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	var resp models.Metrics
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != "requests" || resp.MType != models.Counter || resp.Delta == nil || *resp.Delta != 10 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Value != nil {
		t.Fatalf("expected value to be nil, got %v", *resp.Value)
	}

	counters := svc.AllCounters()
	if v, ok := counters["requests"]; !ok || v != 10 {
		t.Fatalf("counter not updated: got (%v, %v)", v, ok)
	}

	// Do another update and ensure accumulation and returned delta reflect current value
	d2 := int64(5)
	reqBody2 := models.Metrics{ID: "requests", MType: models.Counter, Delta: &d2}
	buf2, _ := json.Marshal(reqBody2)
	req2 := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(buf2))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	h.UpdateJSONHandler(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr2.Code)
	}
	var resp2 models.Metrics
	if err := json.Unmarshal(rr2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp2.Delta == nil || *resp2.Delta != 15 {
		t.Fatalf("expected accumulated delta 15, got %+v", resp2)
	}
	if v := svc.AllCounters()["requests"]; v != 15 {
		t.Fatalf("expected storage counter 15, got %d", v)
	}
}
