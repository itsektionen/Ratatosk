// Package httpx has the response helpers that every handler needs
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// InternalError logs err, with any extra key-value pairs, and tells the client
// something went wrong on our end
func InternalError(w http.ResponseWriter, msg string, err error, args ...any) {
	slog.Error(msg, append(args, "err", err)...)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

// ReadJSON decodes a small JSON body into v. If that fails it responds with
// 400 and returns false
func ReadJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(v); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return false
	}
	return true
}

func WriteJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
