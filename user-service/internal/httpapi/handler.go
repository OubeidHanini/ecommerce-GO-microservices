package httpapi

import (
	"net/http"

	"github.com/OubeidHanini/ecommerce-GO-microservices/user-service/internal/user"
)

type handler struct {
	users *user.Service
}

func NewHandler(users *user.Service) http.Handler {
	h := &handler{users: users}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /users", h.create)
	mux.HandleFunc("GET /users", h.list)
	mux.HandleFunc("GET /users/{id}", h.get)
	return mux
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	u, err := h.users.Register(r.Context(), req.Name, req.Email)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	u, err := h.users.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}
