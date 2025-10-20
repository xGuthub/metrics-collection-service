package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const (
	serverAddr     = "http://localhost:8080"
	pollInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
	httpTimeout    = 5 * time.Second
)

type metricsStore struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func newMetricsStore() *metricsStore {
	return &metricsStore{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (s *metricsStore) setGauge(name string, value float64) {
	s.mu.Lock()
	s.gauges[name] = value
	s.mu.Unlock()
}

func (s *metricsStore) incCounter(name string, delta int64) {
	s.mu.Lock()
	s.counters[name] += delta
	s.mu.Unlock()
}

func (s *metricsStore) getSnapshot() (map[string]float64, map[string]int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g := make(map[string]float64, len(s.gauges))
	for k, v := range s.gauges {
		g[k] = v
	}
	c := make(map[string]int64, len(s.counters))
	for k, v := range s.counters {
		c[k] = v
	}
	return g, c
}

func collectRuntimeMetrics(store *metricsStore) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// All listed as gauge (cast to float64).
	store.setGauge("Alloc", float64(m.Alloc))
	store.setGauge("BuckHashSys", float64(m.BuckHashSys))
	store.setGauge("Frees", float64(m.Frees))
	store.setGauge("GCCPUFraction", m.GCCPUFraction)
	store.setGauge("GCSys", float64(m.GCSys))
	store.setGauge("HeapAlloc", float64(m.HeapAlloc))
	store.setGauge("HeapIdle", float64(m.HeapIdle))
	store.setGauge("HeapInuse", float64(m.HeapInuse))
	store.setGauge("HeapObjects", float64(m.HeapObjects))
	store.setGauge("HeapReleased", float64(m.HeapReleased))
	store.setGauge("HeapSys", float64(m.HeapSys))
	store.setGauge("LastGC", float64(m.LastGC))
	store.setGauge("Lookups", float64(m.Lookups))
	store.setGauge("MCacheInuse", float64(m.MCacheInuse))
	store.setGauge("MCacheSys", float64(m.MCacheSys))
	store.setGauge("MSpanInuse", float64(m.MSpanInuse))
	store.setGauge("MSpanSys", float64(m.MSpanSys))
	store.setGauge("Mallocs", float64(m.Mallocs))
	store.setGauge("NextGC", float64(m.NextGC))
	store.setGauge("NumForcedGC", float64(m.NumForcedGC))
	store.setGauge("NumGC", float64(m.NumGC))
	store.setGauge("OtherSys", float64(m.OtherSys))
	store.setGauge("PauseTotalNs", float64(m.PauseTotalNs))
	store.setGauge("StackInuse", float64(m.StackInuse))
	store.setGauge("StackSys", float64(m.StackSys))
	store.setGauge("Sys", float64(m.Sys))
	store.setGauge("TotalAlloc", float64(m.TotalAlloc))

	// Random gauge
	store.setGauge("RandomValue", rand.Float64())
}

func reportMetrics(ctx context.Context, client *http.Client, store *metricsStore) {
	gauges, counters := store.getSnapshot()
	// Send gauges
	for name, val := range gauges {
		select {
		case <-ctx.Done():
			return
		default:
		}
		valueStr := strconv.FormatFloat(val, 'g', -1, 64)
		url := fmt.Sprintf("%s/update/gauge/%s/%s", serverAddr, name, valueStr)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, http.NoBody)
		if err != nil {
			log.Printf("build request failed for gauge %s: %v", name, err)
			continue
		}
		req.Header.Set("Content-Type", "text/plain")
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("report gauge %s failed: %v", name, err)
			continue
		}
		log.Printf("report gauge %s success: %s", name, valueStr)
		_ = resp.Body.Close()
	}

	// Send counters
	for name, val := range counters {
		select {
		case <-ctx.Done():
			return
		default:
		}
		url := fmt.Sprintf("%s/update/counter/%s/%d", serverAddr, name, val)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, http.NoBody)
		if err != nil {
			log.Printf("build request failed for counter %s: %v", name, err)
			continue
		}
		req.Header.Set("Content-Type", "text/plain")
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("report counter %s failed: %v", name, err)
			continue
		}
		log.Printf("report counter %s success: %d", name, val)
		_ = resp.Body.Close()
	}
}

func main() {
	store := newMetricsStore()
	client := &http.Client{Timeout: httpTimeout}

	// Initial collection and counters init
	collectRuntimeMetrics(store)
	store.incCounter("PollCount", 1)

	// Tickers
	pollTicker := time.NewTicker(pollInterval)
	reportTicker := time.NewTicker(reportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Optional: graceful shutdown on SIGINT/SIGTERM is handled by parent process in many setups.

	for {
		select {
		case <-pollTicker.C:
			collectRuntimeMetrics(store)
			store.incCounter("PollCount", 1)
		case <-reportTicker.C:
			reportMetrics(ctx, client, store)
		case <-ctx.Done():
			return
		}
	}
}
