package models

import "time"

type LogEntry struct {
	ID         string    `json:"id"`
	Service    string    `json:"service"`
	Level      string    `json:"level"`
	Method     string    `json:"method,omitempty"`
	Path       string    `json:"path,omitempty"`
	Status     int       `json:"status,omitempty"`
	DurationMs int64     `json:"duration_ms,omitempty"`
	Message    string    `json:"message,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
