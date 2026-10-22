package model

import "time"

// Product represents a warehouse product.
type Product struct {
	ID             int       `json:"id"`
	SKU            string    `json:"sku"`
	Name           string    `json:"name"`
	Price          int       `json:"price"`
	Stock          int       `json:"stock"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	LastMovementAt *time.Time `json:"last_movement_at"` // null if no movements
}

// StockMovement represents a stock in/out event.
type StockMovement struct {
	ID        int       `json:"id"`
	ProductID int       `json:"product_id"`
	Type      string    `json:"type"` // "in" or "out"
	Qty       int       `json:"qty"`
	Note      *string   `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

// SalesOrder represents a customer order.
type SalesOrder struct {
	ID          int              `json:"id"`
	Customer    string           `json:"customer"`
	Status      string           `json:"status"`
	Total       int              `json:"total"`
	CreatedAt   time.Time        `json:"created_at"`
	CancelledAt *time.Time       `json:"cancelled_at"`
	Items       []SalesOrderItem `json:"items,omitempty"`
}

// SalesOrderItem represents a line item in an order.
type SalesOrderItem struct {
	ID        int    `json:"id"`
	OrderID   int    `json:"order_id"`
	SKU       string `json:"sku"`
	Qty       int    `json:"qty"`
	UnitPrice int    `json:"unit_price"`
}

// --- Request / Response DTOs ---

type CreateProductRequest struct {
	SKU   string `json:"sku"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

type UpdateProductRequest struct {
	Name     string `json:"name"`
	Price    int    `json:"price"`
	IsActive *bool  `json:"is_active"`
}

type AdjustStockRequest struct {
	Type string  `json:"type"` // "in" or "out"
	Qty  int     `json:"qty"`
	Note *string `json:"note"`
}

type CreateOrderRequest struct {
	Customer string             `json:"customer"`
	Items    []OrderItemRequest `json:"items"`
}

type OrderItemRequest struct {
	SKU string `json:"sku"`
	Qty int    `json:"qty"`
}

type ProductsResponse struct {
	Data  []Product `json:"data"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
	Total int       `json:"total"`
}

type MovementsResponse struct {
	Data []StockMovement `json:"data"`
}

type OrdersResponse struct {
	Data  []SalesOrder `json:"data"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
	Total int          `json:"total"`
}

// ErrorCode defines machine-readable error codes.
type ErrorCode string

const (
	ErrProductHasHistory ErrorCode = "PRODUCT_HAS_HISTORY"
	ErrInsufficientStock ErrorCode = "INSUFFICIENT_STOCK"
	ErrNotFound          ErrorCode = "NOT_FOUND"
	ErrBadRequest        ErrorCode = "BAD_REQUEST"
	ErrAlreadyCancelled  ErrorCode = "ALREADY_CANCELLED"
)

type APIError struct {
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message,omitempty"`
	SKU       string    `json:"sku,omitempty"`
	Available *int      `json:"available,omitempty"`
}

type APIErrorResponse struct {
	Error APIError `json:"error"`
}
