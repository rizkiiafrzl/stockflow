package handler

import (
	"encoding/json"
	"net/http"
)

// respond writes a JSON response with the given status code.
func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// respondError writes a structured error JSON response.
func respondError(w http.ResponseWriter, status int, apiErr any) {
	respond(w, status, map[string]any{"error": apiErr})
}

// decode decodes JSON request body into v.
func decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
