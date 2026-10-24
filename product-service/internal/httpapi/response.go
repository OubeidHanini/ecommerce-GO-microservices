package httpapi

import (
	"errors"
	"net/http"

	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/httpx"
	"github.com/OubeidHanini/ecommerce-GO-microservices/product-service/internal/product"
)

// writeError maps domain errors to HTTP status codes: the domain knows nothing about HTTP.
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, product.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, product.ErrNotFound):
		status = http.StatusNotFound
	}
	httpx.Error(w, status, err)
}
