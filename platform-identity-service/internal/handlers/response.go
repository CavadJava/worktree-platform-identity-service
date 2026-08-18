package handlers

import (
	"encoding/json"
	"net/http"
)

type envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}

type errorEnvelope struct {
	Success bool      `json:"success"`
	Error   errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(envelope{Success: true, Data: data})
}

func writeError(w http.ResponseWriter, status int, message string) {
	code := "internal_error"
	switch status {
	case http.StatusBadRequest:
		code = "bad_request"
	case http.StatusUnauthorized:
		code = "unauthorized"
	case http.StatusForbidden:
		code = "forbidden"
	case http.StatusNotFound:
		code = "not_found"
	case http.StatusConflict:
		code = "conflict"
	case http.StatusServiceUnavailable:
		code = "service_unavailable"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorEnvelope{Success: false, Error: errorBody{Code: code, Message: message}})
}
