package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/xGuthub/metrics-collection-service/internal/config"
	models "github.com/xGuthub/metrics-collection-service/internal/model"
)

const (
	httpTimeout = 5 * time.Second
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

func reportMetrics(ctx context.Context, client *resty.Client, store *metricsStore, baseURL string) {
	gauges, counters := store.getSnapshot()

	// Endpoint for JSON updates
	url := fmt.Sprintf("%s/update/", baseURL)

	// Send gauges as JSON one by one
	for name, val := range gauges {
		select {
		case <-ctx.Done():
			return
		default:
		}

		v := val // create addressable copy
		payload := models.Metrics{ID: name, MType: models.Gauge, Value: &v}

		// Marshal and gzip the payload
		body, err := gzipJSON(payload)
		if err != nil {
			log.Printf("prepare gauge %s failed: %v", name, err)
			continue
		}

		_, err = client.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			// Accept-Encoding is automatically handled by net/http, but setting explicitly is okay
			SetHeader("Accept-Encoding", "gzip").
			SetBody(body).
			Post(url)
		if err != nil {
			log.Printf("report gauge %s failed: %v", name, err)

			continue
		}
		log.Printf("report gauge %s success", name)
	}

	// Send counters as JSON one by one
	for name, val := range counters {
		select {
		case <-ctx.Done():
			return
		default:
		}

		d := val // create addressable copy
		payload := models.Metrics{ID: name, MType: models.Counter, Delta: &d}

		// Marshal and gzip the payload
		body, err := gzipJSON(payload)
		if err != nil {
			log.Printf("prepare counter %s failed: %v", name, err)
			continue
		}

		_, err = client.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("Accept-Encoding", "gzip").
			SetBody(body).
			Post(url)
		if err != nil {
			log.Printf("report counter %s failed: %v", name, err)

			continue
		}
		log.Printf("report counter %s success", name)
	}
}

// gzipJSON marshals v to JSON and gzips it.
func gzipJSON(v any) ([]byte, error) {
	// Marshal to JSON
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	// Compress
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(b); err != nil {
		zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func main() {
	// Load config from flags: -a, -r, -p
	cfg, err := config.LoadAgentConfigFromFlags()
	if err != nil {
		log.Fatalf("failed to parse flags: %v", err)
	}

	store := newMetricsStore()
	client := resty.New().SetTimeout(httpTimeout)

	// Initial collection and counters init
	collectRuntimeMetrics(store)
	store.incCounter("PollCount", 1)

	// Tickers
	pollTicker := time.NewTicker(cfg.PollInterval)
	reportTicker := time.NewTicker(cfg.ReportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	baseURL := fmt.Sprintf("http://%s", cfg.Address)

	for {
		select {
		case <-pollTicker.C:
			collectRuntimeMetrics(store)
			store.incCounter("PollCount", 1)
		case <-reportTicker.C:
			reportMetrics(ctx, client, store, baseURL)
		case <-ctx.Done():
			return
		}
	}
}
