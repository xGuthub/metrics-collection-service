package main

import (
	"github.com/xGuthub/metrics-collection-service/internal/handler"
	"github.com/xGuthub/metrics-collection-service/internal/repository"
	"github.com/xGuthub/metrics-collection-service/internal/service"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServer_serveUpdate(t *testing.T) {
	type fields struct {
		mHandler *handler.MetricsHandler
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	okHandler := handler.NewMetricsHandler(service.NewMetricsService(repository.NewMemStorage()))

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
		wantBody       string
		wantAllow      string
		wantCT         string
	}{
		{
			name:           "method not allowed GET",
			fields:         fields{mHandler: nil},
			args:           args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodGet, "/update/gauge/temp/1", nil)},
			wantStatusCode: http.StatusMethodNotAllowed,
			wantBody:       "method not allowed",
			wantAllow:      http.MethodPost,
			wantCT:         "text/plain; charset=utf-8",
		},
		{
			name:   "unsupported media type",
			fields: fields{mHandler: nil},
			args: func() args {
				rr := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, "/update/gauge/temp/1", nil)
				req.Header.Set("Content-Type", "application/json")
				return args{w: rr, r: req}
			}(),
			wantStatusCode: http.StatusUnsupportedMediaType,
			wantBody:       "unsupported media type: expected text/plain",
			wantCT:         "text/plain; charset=utf-8",
		},
		{
			name:           "not found route",
			fields:         fields{mHandler: nil},
			args:           args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/unknown", nil)},
			wantStatusCode: http.StatusNotFound,
			wantBody:       "not found",
			wantCT:         "text/plain; charset=utf-8",
		},
		{
			name:           "missing metric name",
			fields:         fields{mHandler: nil},
			args:           args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/gauge//12.3", nil)},
			wantStatusCode: http.StatusNotFound,
			wantBody:       "metric name is required",
			wantCT:         "text/plain; charset=utf-8",
		},
		{
			name:           "bad metric type",
			fields:         fields{mHandler: okHandler},
			args:           args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/unknown/name/1", nil)},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "bad metric type",
			wantCT:         "text/plain; charset=utf-8",
		},
		{
			name:           "gauge ok",
			fields:         fields{mHandler: okHandler},
			args:           args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/gauge/temperature/12.34", nil)},
			wantStatusCode: http.StatusOK,
			wantBody:       "OK",
			wantCT:         "text/plain; charset=utf-8",
		},
		{
			name:           "gauge bad value",
			fields:         fields{mHandler: okHandler},
			args:           args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/gauge/temp/not-a-number", nil)},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "bad gauge value",
			wantCT:         "text/plain; charset=utf-8",
		},
		{
			name:           "gauge NaN value",
			fields:         fields{mHandler: okHandler},
			args:           args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/gauge/temp/NaN", nil)},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "bad gauge value",
			wantCT:         "text/plain; charset=utf-8",
		},
		{
			name:           "counter ok",
			fields:         fields{mHandler: okHandler},
			args:           args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/counter/requests/5", nil)},
			wantStatusCode: http.StatusOK,
			wantBody:       "OK",
			wantCT:         "text/plain; charset=utf-8",
		},
		{
			name:           "counter bad value",
			fields:         fields{mHandler: okHandler},
			args:           args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/counter/requests/bad", nil)},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "bad counter value",
			wantCT:         "text/plain; charset=utf-8",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				mHandler: tt.fields.mHandler,
			}
			s.serveUpdate(tt.args.w, tt.args.r)

			// Validate response
			rr := tt.args.w.(*httptest.ResponseRecorder)
			res := rr.Result()
			if res.StatusCode != tt.wantStatusCode {
				t.Fatalf("status = %d, want %d", res.StatusCode, tt.wantStatusCode)
			}
			if ct := res.Header.Get("Content-Type"); tt.wantCT != "" && ct != tt.wantCT {
				t.Fatalf("content-type = %q, want %q", ct, tt.wantCT)
			}
			if tt.wantAllow != "" {
				if got := res.Header.Get("Allow"); got != tt.wantAllow {
					t.Fatalf("allow header = %q, want %q", got, tt.wantAllow)
				}
			}
			body := rr.Body.String()
			if body != tt.wantBody {
				t.Fatalf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}
