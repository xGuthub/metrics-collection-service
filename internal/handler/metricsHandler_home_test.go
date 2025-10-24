package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHomeHandler_Empty(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.HomeHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("unexpected Content-Type: %q", ct)
	}
	body := rr.Body.String()
	for _, must := range []string{
		"<h1>Metrics</h1>",
		"<h2>Gauges</h2>",
		"<h2>Counters</h2>",
		"<li><em>No gauges</em></li>",
		"<li><em>No counters</em></li>",
	} {
		if !strings.Contains(body, must) {
			t.Fatalf("response body missing %q. body=%q", must, body)
		}
	}
}

func TestHomeHandler_WithMetricsSorted(t *testing.T) {
	h, svc := newTestHandler()

	// Seed some values out-of-order
	if err := svc.UpdateMetric("gauge", "beta", "2.2"); err != nil {
		t.Fatalf("seed gauge beta: %v", err)
	}
	if err := svc.UpdateMetric("gauge", "alpha", "1.1"); err != nil {
		t.Fatalf("seed gauge alpha: %v", err)
	}
	if err := svc.UpdateMetric("counter", "zeta", "5"); err != nil {
		t.Fatalf("seed counter zeta: %v", err)
	}
	if err := svc.UpdateMetric("counter", "mu", "3"); err != nil {
		t.Fatalf("seed counter mu: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.HomeHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	body := rr.Body.String()

	// Ensure items present
	for _, must := range []string{
		"<li>alpha: 1.1</li>",
		"<li>beta: 2.2</li>",
		"<li>mu: 3</li>",
		"<li>zeta: 5</li>",
	} {
		if !strings.Contains(body, must) {
			t.Fatalf("response body missing %q. body=%q", must, body)
		}
	}

	// Check alphabetical order within each group
	giA := strings.Index(body, "<li>alpha: 1.1</li>")
	giB := strings.Index(body, "<li>beta: 2.2</li>")
	if giA == -1 || giB == -1 || giA > giB {
		t.Fatalf("gauges not sorted: alpha index=%d, beta index=%d", giA, giB)
	}

	ciM := strings.Index(body, "<li>mu: 3</li>")
	ciZ := strings.Index(body, "<li>zeta: 5</li>")
	if ciM == -1 || ciZ == -1 || ciM > ciZ {
		t.Fatalf("counters not sorted: mu index=%d, zeta index=%d", ciM, ciZ)
	}

	// When metrics exist, the "No ..." placeholders should not appear
	if strings.Contains(body, "<li><em>No gauges</em></li>") || strings.Contains(body, "<li><em>No counters</em></li>") {
		t.Fatalf("placeholders should not appear when metrics exist. body=%q", body)
	}
}
