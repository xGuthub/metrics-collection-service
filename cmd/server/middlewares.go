package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xGuthub/metrics-collection-service/internal/logger"
)

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter

		responseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size

	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func WithLogging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}
		h.ServeHTTP(&lw, r)

		duration := time.Since(start)

		logger.Log.Infoln(
			"uri", r.RequestURI,
			"method", r.Method,
			"status", responseData.status,
			"duration", duration,
			"size", responseData.size,
		)
	}

	return http.HandlerFunc(logFn)
}

// gzipResponseWriter wraps ResponseWriter and writes gzipped body when enabled.
type gzipResponseWriter struct {
	http.ResponseWriter
	gw          *gzip.Writer
	enabled     bool
	wroteHeader bool
}

func (grw *gzipResponseWriter) ensureGzip() {
	if grw.enabled && grw.gw == nil {
		grw.gw = gzip.NewWriter(grw.ResponseWriter)
		// Set header once gzip is decided
		grw.Header().Set("Content-Encoding", "gzip")
		grw.Header().Add("Vary", "Accept-Encoding")
		// Content-Length is no longer valid
		grw.Header().Del("Content-Length")
	}
}

func (grw *gzipResponseWriter) WriteHeader(statusCode int) {
	grw.wroteHeader = true
	if grw.enabled {
		// Ensure headers updated before writing
		grw.ensureGzip()
	}
	grw.ResponseWriter.WriteHeader(statusCode)
}

func (grw *gzipResponseWriter) Write(b []byte) (int, error) {
	if grw.enabled {
		grw.ensureGzip()

		return grw.gw.Write(b)
	}

	return grw.ResponseWriter.Write(b)
}

func (grw *gzipResponseWriter) Close() error {
	if grw.gw != nil {
		return grw.gw.Close()
	}

	return nil
}

// WithGzip handles gzip decoding for requests and gzip encoding for responses.
func WithGzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Request: if Content-Encoding: gzip, replace Body with a gzip reader
		if strings.Contains(strings.ToLower(r.Header.Get("Content-Encoding")), "gzip") {
			origBody := r.Body
			zr, err := gzip.NewReader(origBody)
			if err != nil {
				http.Error(w, "invalid gzip body", http.StatusBadRequest)

				return
			}
			// Ensure both gzip reader and original body are closed
			r.Body = &combinedReadCloser{Reader: zr, closers: []io.Closer{zr, origBody}}
		}

		// Response: if Accept-Encoding includes gzip, wrap writer
		acceptsGzip := strings.Contains(strings.ToLower(r.Header.Get("Accept-Encoding")), "gzip")
		grw := &gzipResponseWriter{ResponseWriter: w, enabled: acceptsGzip}
		defer func() { _ = grw.Close() }()

		next.ServeHTTP(grw, r)
	})
}

type combinedReadCloser struct {
	io.Reader

	closers []io.Closer
}

func (c *combinedReadCloser) Close() error {
	var firstErr error
	for _, cl := range c.closers {
		if err := cl.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}
