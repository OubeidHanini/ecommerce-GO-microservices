package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestWrapLabelsRequestsByRoutePatternNotRawPath(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	h := Wrap(mux)

	for _, path := range []string{"/users/1", "/users/2", "/nope"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}

	if got := testutil.ToFloat64(requests.WithLabelValues("GET", "GET /users/{id}", "404")); got != 2 {
		t.Errorf("requests for route GET /users/{id} = %v, want 2", got)
	}
	if got := testutil.ToFloat64(requests.WithLabelValues("GET", "unmatched", "404")); got != 1 {
		t.Errorf("unmatched requests = %v, want 1", got)
	}
}

func TestWrapServesMetrics(t *testing.T) {
	rec := httptest.NewRecorder()
	Wrap(http.NotFoundHandler()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "go_goroutines") {
		t.Errorf("GET /metrics: status %d, runtime metrics missing", rec.Code)
	}
}
