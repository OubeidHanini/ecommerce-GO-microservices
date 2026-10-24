// Package metrics exposes Prometheus metrics: request rate, errors and latency for
// every HTTP route, plus the Go runtime metrics of the default registry.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests by method, route and status code.",
	}, []string{"method", "route", "code"})

	duration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency by method and route.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
)

// Wrap serves GET /metrics and instruments every other request handled by next.
func Wrap(next http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.Handle("/", instrument(next))
	return mux
}

func instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		// r.Pattern is the matched route ("GET /users/{id}"), set by the inner ServeMux.
		// Labelling by raw path instead would create one time series per id.
		route := r.Pattern
		if route == "" || route == "/" {
			route = "unmatched"
		}
		requests.WithLabelValues(r.Method, route, strconv.Itoa(rec.status)).Inc()
		duration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.ResponseController reach the underlying writer (Flush, used by the gateway's proxy).
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
