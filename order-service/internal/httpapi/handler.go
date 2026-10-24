package httpapi

import (
	"net/http"

	"github.com/OubeidHanini/ecommerce-GO-microservices/order-service/internal/order"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/httpx"
)

type handler struct {
	orders *order.Service
}

func NewHandler(orders *order.Service) http.Handler {
	h := &handler{orders: orders}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", httpx.Health)
	mux.HandleFunc("POST /orders", h.create)
	mux.HandleFunc("GET /orders", h.list)
	mux.HandleFunc("GET /orders/{id}", h.get)
	return mux
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID int64 `json:"user_id"`
		Items  []struct {
			ProductID int64 `json:"product_id"`
			Quantity  int   `json:"quantity"`
		} `json:"items"`
	}
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	lines := make([]order.Line, len(req.Items))
	for i, it := range req.Items {
		lines[i] = order.Line{ProductID: it.ProductID, Quantity: it.Quantity}
	}

	o, err := h.orders.Place(r.Context(), req.UserID, lines)
	if err != nil {
		writeError(w, err)
		return
	}
	// 202: the order is accepted but not confirmed yet (stock reservation in progress).
	httpx.JSON(w, http.StatusAccepted, o)
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	orders, err := h.orders.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, orders)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r)
	if !ok {
		return
	}

	o, err := h.orders.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}
