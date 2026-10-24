package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/metrics"
)

// New instruments handler and serves GET /metrics, so no service can forget its metrics.
func New(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           metrics.Wrap(handler),
		ReadHeaderTimeout: 5 * time.Second,
	}
}

// Run serves requests until ctx is cancelled, then gives in-flight
// requests time to finish (graceful shutdown).
func Run(ctx context.Context, srv *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func Getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
