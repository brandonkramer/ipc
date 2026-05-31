package ipc

import (
	"encoding/json"
	"net/http"
	"strings"
)

// GETOnly runs fn when the request method is GET.
func GETOnly(w http.ResponseWriter, r *http.Request, fn func(http.ResponseWriter, *http.Request)) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fn(w, r)
}

// WriteJSON writes an indented JSON response.
func WriteJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// HandleJSON serves GET JSON from fn.
func HandleJSON(w http.ResponseWriter, r *http.Request, fn func() any) {
	GETOnly(w, r, func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, fn())
	})
}

// HandleJSONErr serves GET JSON from fn, returning 500 on error.
func HandleJSONErr(w http.ResponseWriter, r *http.Request, fn func() (any, error)) {
	GETOnly(w, r, func(w http.ResponseWriter, _ *http.Request) {
		v, err := fn()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		WriteJSON(w, v)
	})
}

// HandlePrefix serves GET JSON for one path segment after prefix, returning 404 on error.
func HandlePrefix(w http.ResponseWriter, r *http.Request, prefix string, fn func(id string) (any, error)) {
	GETOnly(w, r, func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, prefix)
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		v, err := fn(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		WriteJSON(w, v)
	})
}
