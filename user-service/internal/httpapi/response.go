package httpapi

import (
	"errors"
	"net/http"

	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/httpx"
	"github.com/OubeidHanini/ecommerce-GO-microservices/user-service/internal/user"
)

// writeError maps domain errors to HTTP status codes: the domain knows nothing about HTTP.
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, user.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, user.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, user.ErrEmailTaken):
		status = http.StatusConflict
	}
	httpx.Error(w, status, err)
}
