package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"stockflow/backend/internal/model"
)

// ProductRepository handles all product-related DB operations.
type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// ListProducts returns paginated products with optional search.
// last_movement_at is derived via LEFT JOIN (always fresh, no denormalization).
func (r *ProductRepository) ListProducts(ctx context.Context, search string, page, limit int) ([]model.Product, int, error) {
	offset := (page - 1) * limit

	// Build WHERE clause for search
	args := []any{}
	whereClause := ""
	argIdx := 1
	if search != "" {
		whereClause = fmt.Sprintf("WHERE (LOWER(p.sku) LIKE $%d OR LOWER(p.name) LIKE $%d)", argIdx, argIdx+1)
		pattern := "%" + strings.ToLower(search) + "%"
		args = append(args, pattern, pattern)
		argIdx += 2
	}

	// Count query
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM products p %s`, whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	// Main query with last_movement_at from JOIN
	query := fmt.Sprintf(`
		SELECT
			p.id, p.sku, p.name, p.price, p.stock, p.is_active, p.created_at,
			MAX(sm.created_at) AS last_movement_at
		FROM products p
		LEFT JOIN stock_movements sm ON sm.product_id = p.id
		%s
		GROUP BY p.id
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Price, &p.Stock, &p.IsActive, &p.CreatedAt, &p.LastMovementAt); err != nil {
			return nil, 0, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}
	if products == nil {
		products = []model.Product{}
	}
	return products, total, rows.Err()
}

// GetProductByID returns a single product.
func (r *ProductRepository) GetProductByID(ctx context.Context, id int) (*model.Product, error) {
	query := `
		SELECT
			p.id, p.sku, p.name, p.price, p.stock, p.is_active, p.created_at,
			MAX(sm.created_at) AS last_movement_at
		FROM products p
		LEFT JOIN stock_movements sm ON sm.product_id = p.id
		WHERE p.id = $1
		GROUP BY p.id
	`
	var p model.Product
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Price, &p.Stock, &p.IsActive, &p.CreatedAt, &p.LastMovementAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	return &p, nil
}

// CreateProduct inserts a new product and returns it.
func (r *ProductRepository) CreateProduct(ctx context.Context, req model.CreateProductRequest) (*model.Product, error) {
	query := `
		INSERT INTO products (sku, name, price)
		VALUES ($1, $2, $3)
		RETURNING id, sku, name, price, stock, is_active, created_at
	`
	var p model.Product
	err := r.db.QueryRowContext(ctx, query, req.SKU, req.Name, req.Price).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Price, &p.Stock, &p.IsActive, &p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}
	return &p, nil
}

// UpdateProduct updates name, price, is_active of a product.
func (r *ProductRepository) UpdateProduct(ctx context.Context, id int, req model.UpdateProductRequest) (*model.Product, error) {
	// Build dynamic SET clause
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if req.Name != "" {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, req.Name)
		argIdx++
	}
	// Price: allow 0, so we always update if provided (check non-negative in handler)
	setClauses = append(setClauses, fmt.Sprintf("price = $%d", argIdx))
	args = append(args, req.Price)
	argIdx++

	if req.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	if len(setClauses) == 0 {
		return r.GetProductByID(ctx, id)
	}

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE products SET %s WHERE id = $%d
		RETURNING id, sku, name, price, stock, is_active, created_at
	`, strings.Join(setClauses, ", "), argIdx)

	var p model.Product
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Price, &p.Stock, &p.IsActive, &p.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}
	return &p, nil
}

// DeleteProduct removes a product if it has no movement history.
func (r *ProductRepository) DeleteProduct(ctx context.Context, id int) (bool, bool, error) {
	// Check if product exists
	var exists bool
	if err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)", id).Scan(&exists); err != nil {
		return false, false, err
	}
	if !exists {
		return false, false, nil // product not found
	}

	// Check if has movement history
	var hasHistory bool
	if err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM stock_movements WHERE product_id = $1)", id,
	).Scan(&hasHistory); err != nil {
		return true, false, err
	}
	if hasHistory {
		return true, true, nil // exists but has history
	}

	_, err := r.db.ExecContext(ctx, "DELETE FROM products WHERE id = $1", id)
	return true, false, err
}

// AdjustStock adds or removes stock atomically in a single transaction.
// Uses UPDATE with conditional WHERE to guard against overdraw — no SELECT-then-UPDATE.
func (r *ProductRepository) AdjustStock(ctx context.Context, productID int, req model.AdjustStockRequest) (*model.StockMovement, int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var newStock int
	var updateErr error

	if req.Type == "out" {
		// Atomic decrement: only succeeds if stock >= qty
		err = tx.QueryRowContext(ctx,
			`UPDATE products SET stock = stock - $1 WHERE id = $2 AND stock >= $1 RETURNING stock`,
			req.Qty, productID,
		).Scan(&newStock)
		if errors.Is(err, sql.ErrNoRows) {
			// Get current stock for the error response
			var available int
			_ = tx.QueryRowContext(ctx, "SELECT stock FROM products WHERE id = $1", productID).Scan(&available)
			return nil, available, fmt.Errorf("insufficient stock")
		}
		updateErr = err
	} else {
		// Increment stock
		err = tx.QueryRowContext(ctx,
			`UPDATE products SET stock = stock + $1 WHERE id = $2 RETURNING stock`,
			req.Qty, productID,
		).Scan(&newStock)
		updateErr = err
	}

	if updateErr != nil {
		return nil, 0, fmt.Errorf("update stock: %w", updateErr)
	}

	// Write movement row in the same transaction
	var mv model.StockMovement
	err = tx.QueryRowContext(ctx,
		`INSERT INTO stock_movements (product_id, type, qty, note) VALUES ($1, $2, $3, $4)
		 RETURNING id, product_id, type, qty, note, created_at`,
		productID, req.Type, req.Qty, req.Note,
	).Scan(&mv.ID, &mv.ProductID, &mv.Type, &mv.Qty, &mv.Note, &mv.CreatedAt)
	if err != nil {
		return nil, 0, fmt.Errorf("insert movement: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, 0, fmt.Errorf("commit tx: %w", err)
	}
	return &mv, newStock, nil
}

// ListMovements returns stock movement history for a product (newest first).
func (r *ProductRepository) ListMovements(ctx context.Context, productID int) ([]model.StockMovement, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, product_id, type, qty, note, created_at
		 FROM stock_movements WHERE product_id = $1 ORDER BY created_at DESC`,
		productID,
	)
	if err != nil {
		return nil, fmt.Errorf("list movements: %w", err)
	}
	defer rows.Close()

	var movements []model.StockMovement
	for rows.Next() {
		var m model.StockMovement
		if err := rows.Scan(&m.ID, &m.ProductID, &m.Type, &m.Qty, &m.Note, &m.CreatedAt); err != nil {
			return nil, err
		}
		movements = append(movements, m)
	}
	if movements == nil {
		movements = []model.StockMovement{}
	}
	return movements, rows.Err()
}

// SKUExists checks if a SKU already exists (for validation).
func (r *ProductRepository) SKUExists(ctx context.Context, sku string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM products WHERE sku = $1)", sku).Scan(&exists)
	return exists, err
}
