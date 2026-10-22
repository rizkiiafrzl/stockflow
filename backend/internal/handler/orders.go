package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"stockflow/backend/internal/model"
	"stockflow/backend/internal/repository"
)

// OrderHandler handles all order-related HTTP routes.
type OrderHandler struct {
	repo *repository.OrderRepository
}

func NewOrderHandler(repo *repository.OrderRepository) *OrderHandler {
	return &OrderHandler{repo: repo}
}

// POST /api/orders
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateOrderRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, map[string]string{"code": "BAD_REQUEST", "message": "invalid JSON"})
		return
	}

	if strings.TrimSpace(req.Customer) == "" {
		respondError(w, http.StatusBadRequest, map[string]any{"code": "VALIDATION_ERROR", "message": "customer is required"})
		return
	}

	order, err := h.repo.CreateOrder(r.Context(), req)
	if err != nil {
		msg := err.Error()

		// Validation errors (before any write)
		if strings.HasPrefix(msg, "validation:") {
			respondError(w, http.StatusBadRequest, map[string]string{
				"code":    "VALIDATION_ERROR",
				"message": strings.TrimPrefix(msg, "validation: "),
			})
			return
		}
		if strings.HasPrefix(msg, "unknown_sku:") {
			sku := strings.TrimPrefix(msg, "unknown_sku: ")
			respondError(w, http.StatusBadRequest, map[string]string{
				"code":    "UNKNOWN_SKU",
				"message": "unknown SKU: " + sku,
				"sku":     sku,
			})
			return
		}

		// Insufficient stock (409)
		if strings.HasPrefix(msg, "insufficient_stock:") {
			// parse sku and available from message: "insufficient_stock: sku=X available=N"
			parts := strings.Fields(strings.TrimPrefix(msg, "insufficient_stock: "))
			sku := ""
			available := 0
			for _, p := range parts {
				if strings.HasPrefix(p, "sku=") {
					sku = strings.TrimPrefix(p, "sku=")
				}
				if strings.HasPrefix(p, "available=") {
					strconv.Atoi(strings.TrimPrefix(p, "available="))
					n, _ := strconv.Atoi(strings.TrimPrefix(p, "available="))
					available = n
				}
			}
			respondError(w, http.StatusConflict, map[string]any{
				"code":      string(model.ErrInsufficientStock),
				"sku":       sku,
				"available": available,
			})
			return
		}

		respondError(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR", "message": msg})
		return
	}

	respond(w, http.StatusCreated, order)
}

// GET /api/orders/{id}
func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, map[string]string{"code": "BAD_REQUEST", "message": "invalid id"})
		return
	}

	order, err := h.repo.GetOrder(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	if order == nil {
		respondError(w, http.StatusNotFound, map[string]string{"code": "NOT_FOUND", "message": "order not found"})
		return
	}
	respond(w, http.StatusOK, order)
}

// POST /api/orders/{id}/cancel
func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, map[string]string{"code": "BAD_REQUEST", "message": "invalid id"})
		return
	}

	order, err := h.repo.CancelOrder(r.Context(), id)
	if err != nil {
		msg := err.Error()
		if msg == "not_found" {
			respondError(w, http.StatusNotFound, map[string]string{"code": "NOT_FOUND", "message": "order not found"})
			return
		}
		if msg == "already_cancelled" {
			respondError(w, http.StatusConflict, map[string]string{"code": string(model.ErrAlreadyCancelled)})
			return
		}
		respondError(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR", "message": msg})
		return
	}
	respond(w, http.StatusOK, order)
}

// GET /api/orders?page=&limit=
func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	orders, total, err := h.repo.ListOrders(r.Context(), page, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	respond(w, http.StatusOK, model.OrdersResponse{
		Data:  orders,
		Page:  page,
		Limit: limit,
		Total: total,
	})
}

