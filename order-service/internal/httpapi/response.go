package httpapi

import (
	"errors"
	"net/http"

	"github.com/OubeidHanini/ecommerce-GO-microservices/order-service/internal/order"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/httpx"
)

// writeError maps domain errors to HTTP status codes: the domain knows nothing about HTTP.
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, order.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, order.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, order.ErrUnavailable):
		status = http.StatusServiceUnavailable
	}
	httpx.Error(w, status, err)
}
