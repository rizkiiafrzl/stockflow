package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"stockflow/backend/internal/model"
	"stockflow/backend/internal/repository"
)

// ProductHandler handles all product-related HTTP routes.
type ProductHandler struct {
	repo *repository.ProductRepository
}

func NewProductHandler(repo *repository.ProductRepository) *ProductHandler {
	return &ProductHandler{repo: repo}
}

// GET /api/products?search=&page=&limit=
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	products, total, err := h.repo.ListProducts(r.Context(), search, page, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	respond(w, http.StatusOK, model.ProductsResponse{
		Data:  products,
		Page:  page,
		Limit: limit,
		Total: total,
	})
}

// POST /api/products
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateProductRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, map[string]string{"code": "BAD_REQUEST", "message": "invalid JSON"})
		return
	}

	// Validation
	var errs []string
	if strings.TrimSpace(req.SKU) == "" {
		errs = append(errs, "sku is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		errs = append(errs, "name is required")
	}
	if req.Price < 0 {
		errs = append(errs, "price must be >= 0")
	}
	if len(errs) > 0 {
		respondError(w, http.StatusBadRequest, map[string]any{"code": "VALIDATION_ERROR", "messages": errs})
		return
	}

	// Check SKU uniqueness
	exists, err := h.repo.SKUExists(r.Context(), req.SKU)
	if err != nil {
		respondError(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR"})
		return
	}
	if exists {
		respondError(w, http.StatusConflict, map[string]string{"code": "SKU_ALREADY_EXISTS", "message": fmt.Sprintf("SKU %q already exists", req.SKU)})
		return
	}

	product, err := h.repo.CreateProduct(r.Context(), req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	respond(w, http.StatusCreated, product)
}

// PUT /api/products/{id}
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, map[string]string{"code": "BAD_REQUEST", "message": "invalid id"})
		return
	}

	var req model.UpdateProductRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, map[string]string{"code": "BAD_REQUEST", "message": "invalid JSON"})
		return
	}

	if req.Price < 0 {
		respondError(w, http.StatusBadRequest, map[string]string{"code": "VALIDATION_ERROR", "message": "price must be >= 0"})
		return
	}

	product, err := h.repo.UpdateProduct(r.Context(), id, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	if product == nil {
		respondError(w, http.StatusNotFound, map[string]string{"code": "NOT_FOUND", "message": "product not found"})
		return
	}
	respond(w, http.StatusOK, product)
}

// DELETE /api/products/{id}
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, map[string]string{"code": "BAD_REQUEST", "message": "invalid id"})
		return
	}

	exists, hasHistory, err := h.repo.DeleteProduct(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	if !exists {
		respondError(w, http.StatusNotFound, map[string]string{"code": "NOT_FOUND", "message": "product not found"})
		return
	}
	if hasHistory {
		respondError(w, http.StatusConflict, map[string]string{"code": string(model.ErrProductHasHistory)})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/products/{id}/stock
func (h *ProductHandler) AdjustStock(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, map[string]string{"code": "BAD_REQUEST", "message": "invalid id"})
		return
	}

	var req model.AdjustStockRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, map[string]string{"code": "BAD_REQUEST", "message": "invalid JSON"})
		return
	}

	// Validate
	if req.Type != "in" && req.Type != "out" {
		respondError(w, http.StatusBadRequest, map[string]string{"code": "VALIDATION_ERROR", "message": `type must be "in" or "out"`})
		return
	}
	if req.Qty <= 0 {
		respondError(w, http.StatusBadRequest, map[string]string{"code": "VALIDATION_ERROR", "message": "qty must be > 0"})
		return
	}

	// Check product exists
	product, err := h.repo.GetProductByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR"})
		return
	}
	if product == nil {
		respondError(w, http.StatusNotFound, map[string]string{"code": "NOT_FOUND", "message": "product not found"})
		return
	}

	movement, available, err := h.repo.AdjustStock(r.Context(), id, req)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient stock") {
			respondError(w, http.StatusConflict, map[string]any{
				"code":      string(model.ErrInsufficientStock),
				"available": available,
			})
			return
		}
		respondError(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	respond(w, http.StatusOK, movement)
}

// GET /api/products/{id}/movements
func (h *ProductHandler) ListMovements(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, map[string]string{"code": "BAD_REQUEST", "message": "invalid id"})
		return
	}

	// Check product exists
	product, err := h.repo.GetProductByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR"})
		return
	}
	if product == nil {
		respondError(w, http.StatusNotFound, map[string]string{"code": "NOT_FOUND", "message": "product not found"})
		return
	}

	movements, err := h.repo.ListMovements(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	respond(w, http.StatusOK, model.MovementsResponse{Data: movements})
}
