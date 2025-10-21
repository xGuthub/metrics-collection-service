package handler

import (
	"github.com/xGuthub/metrics-collection-service/internal/repository"
	"github.com/xGuthub/metrics-collection-service/internal/service"
	"net/http"
	"testing"
)

func TestMetricsHandler_HandleUpdate(t *testing.T) {
	type fields struct {
		metricsService *service.MetricsService
	}
	type args struct {
		mType  string
		name   string
		rawVal string
	}
	baseService := service.NewMetricsService(repository.NewMemStorage())

	tests := []struct {
		name       string
		fields     fields
		args       args
		wantCode   int
		wantStatus string
	}{
		{
			name:       "gauge ok",
			fields:     fields{metricsService: baseService},
			args:       args{mType: "gauge", name: "temperature", rawVal: "12.34"},
			wantCode:   http.StatusOK,
			wantStatus: "OK",
		},
		{
			name:       "gauge bad value non-number",
			fields:     fields{metricsService: baseService},
			args:       args{mType: "gauge", name: "temperature", rawVal: "abc"},
			wantCode:   http.StatusBadRequest,
			wantStatus: "bad gauge value",
		},
		{
			name:       "gauge bad value NaN",
			fields:     fields{metricsService: baseService},
			args:       args{mType: "gauge", name: "temperature", rawVal: "NaN"},
			wantCode:   http.StatusBadRequest,
			wantStatus: "bad gauge value",
		},
		{
			name:       "gauge bad value Inf",
			fields:     fields{metricsService: baseService},
			args:       args{mType: "gauge", name: "temperature", rawVal: "Inf"},
			wantCode:   http.StatusBadRequest,
			wantStatus: "bad gauge value",
		},
		{
			name:       "counter ok",
			fields:     fields{metricsService: baseService},
			args:       args{mType: "counter", name: "requests", rawVal: "5"},
			wantCode:   http.StatusOK,
			wantStatus: "OK",
		},
		{
			name:       "counter ok negative delta",
			fields:     fields{metricsService: baseService},
			args:       args{mType: "counter", name: "requests", rawVal: "-2"},
			wantCode:   http.StatusOK,
			wantStatus: "OK",
		},
		{
			name:       "counter bad value",
			fields:     fields{metricsService: baseService},
			args:       args{mType: "counter", name: "requests", rawVal: "x"},
			wantCode:   http.StatusBadRequest,
			wantStatus: "bad counter value",
		},
		{
			name:       "bad metric type",
			fields:     fields{metricsService: baseService},
			args:       args{mType: "timer", name: "t", rawVal: "1"},
			wantCode:   http.StatusBadRequest,
			wantStatus: "bad metric type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mh := &MetricsHandler{
				metricsService: tt.fields.metricsService,
			}
			gotCode, gotStatus := mh.HandleUpdate(tt.args.mType, tt.args.name, tt.args.rawVal)
			if gotCode != tt.wantCode {
				t.Errorf("HandleUpdate() gotCode = %v, want %v", gotCode, tt.wantCode)
			}
			if gotStatus != tt.wantStatus {
				t.Errorf("HandleUpdate() gotStatus = %v, want %v", gotStatus, tt.wantStatus)
			}
		})
	}
}

func TestMetricsHandler_HandleGet(t *testing.T) {
	type fields struct {
		metricsService *service.MetricsService
	}
	type args struct {
		mType string
		name  string
	}

	// Use a fresh base service to avoid cross-test contamination
	baseService := service.NewMetricsService(repository.NewMemStorage())

	tests := []struct {
		name      string
		fields    fields
		args      args
		setup     func(s *service.MetricsService)
		wantCode  int
		wantValue string
	}{
		{
			name:   "gauge exists",
			fields: fields{metricsService: baseService},
			args:   args{mType: "gauge", name: "temperature"},
			setup: func(s *service.MetricsService) {
				s.UpdateGauge("temperature", 12.34)
			},
			wantCode:  http.StatusOK,
			wantValue: "12.34",
		},
		{
			name:      "gauge not found",
			fields:    fields{metricsService: baseService},
			args:      args{mType: "gauge", name: "unknown_gauge"},
			setup:     nil,
			wantCode:  http.StatusNotFound,
			wantValue: "gauge not found",
		},
		{
			name:   "counter exists",
			fields: fields{metricsService: baseService},
			args:   args{mType: "counter", name: "requests"},
			setup: func(s *service.MetricsService) {
				s.IncrementCounter("requests", 5)
			},
			wantCode:  http.StatusOK,
			wantValue: "5",
		},
		{
			name:      "counter not found",
			fields:    fields{metricsService: baseService},
			args:      args{mType: "counter", name: "unknown_counter"},
			setup:     nil,
			wantCode:  http.StatusNotFound,
			wantValue: "counter not found",
		},
		{
			name:      "bad metric type",
			fields:    fields{metricsService: baseService},
			args:      args{mType: "timer", name: "t"},
			setup:     nil,
			wantCode:  http.StatusBadRequest,
			wantValue: "bad metric type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.fields.metricsService)
			}
			mh := &MetricsHandler{
				metricsService: tt.fields.metricsService,
			}
			gotCode, gotVal := mh.HandleGet(tt.args.mType, tt.args.name)
			if gotCode != tt.wantCode {
				t.Errorf("HandleGet() gotCode = %v, want %v", gotCode, tt.wantCode)
			}
			if gotVal != tt.wantValue {
				t.Errorf("HandleGet() gotVal = %v, want %v", gotVal, tt.wantValue)
			}
		})
	}
}
