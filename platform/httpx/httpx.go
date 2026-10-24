// Package httpx holds the HTTP helpers shared by every service.
package httpx

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

// Error writes a JSON error. 5xx errors are logged and masked so internal
// details are not exposed to the client.
func Error(w http.ResponseWriter, status int, err error) {
	msg := err.Error()
	if status >= http.StatusInternalServerError {
		log.Printf("internal error: %v", err)
		msg = http.StatusText(status)
	}
	JSON(w, status, map[string]string{"error": msg})
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return false
	}
	return true
}

func PathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

// Transport is for service-to-service calls. Go's default keeps only 2 idle connections
// per host: under concurrent load, most requests then open a fresh TCP connection, which
// adds seconds of latency (measured: gateway p99 4.5s vs 0.25s inside the service).
func Transport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 512
	t.MaxIdleConnsPerHost = 256
	return t
}

func Health(w http.ResponseWriter, _ *http.Request) {
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
