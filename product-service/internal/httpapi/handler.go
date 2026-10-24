package httpapi

import (
	"net/http"

	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/httpx"
	"github.com/OubeidHanini/ecommerce-GO-microservices/product-service/internal/product"
)

type handler struct {
	products *product.Service
}

func NewHandler(products *product.Service) http.Handler {
	h := &handler{products: products}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", httpx.Health)
	mux.HandleFunc("POST /products", h.create)
	mux.HandleFunc("GET /products", h.list)
	mux.HandleFunc("GET /products/{id}", h.get)
	return mux
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		PriceCents int64  `json:"price_cents"`
		Stock      int    `json:"stock"`
	}
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	p, err := h.products.Create(r.Context(), req.Name, req.PriceCents, req.Stock)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	products, err := h.products.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, products)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(w, r)
	if !ok {
		return
	}

	p, err := h.products.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}
