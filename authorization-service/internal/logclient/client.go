// Package logclient sends request/error logs to the central log-service
// (fire-and-forget: never blocks or fails the caller's request).
package logclient

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	service string
	http    *http.Client
}

func New(baseURL, service string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		service: service,
		http:    &http.Client{Timeout: 2 * time.Second},
	}
}

type entry struct {
	Service    string `json:"service"`
	Level      string `json:"level"`
	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
	Status     int    `json:"status,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Message    string `json:"message,omitempty"`
}

func (c *Client) Send(level, method, path string, status int, durationMs int64, message string) {
	if c == nil || c.baseURL == "" {
		return
	}
	e := entry{Service: c.service, Level: level, Method: method, Path: path, Status: status, DurationMs: durationMs, Message: message}
	go func() {
		body, err := json.Marshal(e)
		if err != nil {
			return
		}
		resp, err := c.http.Post(c.baseURL+"/api/v1/logs", "application/json", bytes.NewReader(body))
		if err != nil {
			return
		}
		resp.Body.Close()
	}()
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// RequestLogger reports every finished request to log-service.
// /health and /swagger are skipped to avoid noise.
func RequestLogger(c *Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/swagger") {
				next.ServeHTTP(w, r)
				return
			}
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(rec, r)
			duration := time.Since(start).Milliseconds()

			level := "info"
			switch {
			case rec.status >= 500:
				level = "error"
			case rec.status >= 400:
				level = "warn"
			}
			c.Send(level, r.Method, r.URL.Path, rec.status, duration, "")
		})
	}
}
