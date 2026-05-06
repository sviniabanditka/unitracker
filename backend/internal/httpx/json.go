package httpx

import (
	"encoding/json"
	"net/http"
)

// Stable error codes returned in the response envelope. Frontend maps these
// to localized i18n strings.
const (
	CodeUnauthorized       = "auth.unauthorized"
	CodeForbidden          = "auth.forbidden"
	CodeValidationFailed   = "validation.failed"
	CodeValidationRequired = "validation.required"
	CodeNotFound           = "not_found"
	CodeConflict           = "conflict"
	CodeMaintenance        = "maintenance"
	CodeInternal           = "internal"
)

// WriteJSON serializes v as JSON with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes a unified error envelope: {code, message, fields?}.
// fields may be nil.
func WriteError(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	body := map[string]any{
		"code":    code,
		"message": message,
		// "error" is kept as an alias of "message" so older clients that read
		// the legacy "error" field keep working during the transition.
		"error": message,
	}
	if len(fields) > 0 {
		body["fields"] = fields
	}
	WriteJSON(w, status, body)
}

// DefaultCodeForStatus picks a stable error code based on the HTTP status when
// the caller does not supply one explicitly.
func DefaultCodeForStatus(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound, http.StatusGone:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	case http.StatusServiceUnavailable:
		return CodeMaintenance
	case http.StatusInternalServerError:
		return CodeInternal
	default:
		if status >= 400 && status < 500 {
			return CodeValidationFailed
		}
		return CodeInternal
	}
}

// WriteErrorStatus is a convenience for writing an error using the default
// code for a status.
func WriteErrorStatus(w http.ResponseWriter, status int, message string) {
	WriteError(w, status, DefaultCodeForStatus(status), message, nil)
}

// WriteValidationError writes a 400 with code "validation.failed" and the
// per-field error map.
func WriteValidationError(w http.ResponseWriter, message string, fields map[string]string) {
	if message == "" {
		message = "Validation failed"
	}
	WriteError(w, http.StatusBadRequest, CodeValidationFailed, message, fields)
}
