package handlers

import (
	"encoding/json"
	"net/http"
)

// apiResponse is the standard envelope every endpoint returns: either
// {"success":true,"data":...} or {"success":false,"error":{"code":...,"message":...}}.
type apiResponse struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *apiError `json:"error,omitempty"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResponse{Success: true, Data: data})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResponse{Success: false, Error: &apiError{Code: errorCode(status), Message: message}})
}

// errorCode maps the HTTP status to a stable machine-readable code so
// clients can branch on it instead of parsing the human-readable message.
func errorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusServiceUnavailable:
		return "service_unavailable"
	case http.StatusInternalServerError:
		return "internal_error"
	default:
		return "error"
	}
}
