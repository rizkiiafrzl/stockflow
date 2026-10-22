package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"stockflow/backend/internal/model"
)

// OrderRepository handles all order-related DB operations.
type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// CreateOrder validates, creates an order, and decrements stock atomically.
// Rules:
//  - All validation before any write
//  - unit_price from products table, total computed server-side
//  - Stock decremented with UPDATE ... WHERE stock >= qty (no SELECT-then-UPDATE)
//  - Single transaction: either everything commits or nothing changes
func (r *OrderRepository) CreateOrder(ctx context.Context, req model.CreateOrderRequest) (*model.SalesOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// --- Phase 1: Validation (before any write) ---

	// 1a. Empty items
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("validation: items cannot be empty")
	}

	// 1b. Duplicate SKUs check
	skuSet := map[string]bool{}
	for _, item := range req.Items {
		if item.Qty <= 0 {
			return nil, fmt.Errorf("validation: qty must be > 0 for sku %q", item.SKU)
		}
		if item.SKU == "" {
			return nil, fmt.Errorf("validation: sku is required")
		}
		if skuSet[item.SKU] {
			return nil, fmt.Errorf("validation: duplicate sku %q in order", item.SKU)
		}
		skuSet[item.SKU] = true
	}

	// 1c. Lock product rows for all SKUs (SELECT FOR UPDATE to prevent concurrent overdraw)
	//     This also validates that the SKUs exist.
	type productInfo struct {
		id    int
		price int
		stock int
	}
	productMap := map[string]productInfo{}

	for _, item := range req.Items {
		var info productInfo
		err := tx.QueryRowContext(ctx,
			`SELECT id, price, stock FROM products WHERE sku = $1 FOR UPDATE`,
			item.SKU,
		).Scan(&info.id, &info.price, &info.stock)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("unknown_sku: %s", item.SKU)
		}
		if err != nil {
			return nil, fmt.Errorf("fetch product %s: %w", item.SKU, err)
		}
		productMap[item.SKU] = info
	}

	// 1d. Check stock sufficiency for ALL items before writing anything
	for _, item := range req.Items {
		info := productMap[item.SKU]
		if info.stock < item.Qty {
			return nil, fmt.Errorf("insufficient_stock: sku=%s available=%d", item.SKU, info.stock)
		}
	}

	// --- Phase 2: Compute total ---
	total := 0
	for _, item := range req.Items {
		total += productMap[item.SKU].price * item.Qty
	}

	// --- Phase 3: Write order ---
	var order model.SalesOrder
	err = tx.QueryRowContext(ctx,
		`INSERT INTO sales_orders (customer, total) VALUES ($1, $2)
		 RETURNING id, customer, status, total, created_at, cancelled_at`,
		req.Customer, total,
	).Scan(&order.ID, &order.Customer, &order.Status, &order.Total, &order.CreatedAt, &order.CancelledAt)
	if err != nil {
		return nil, fmt.Errorf("insert order: %w", err)
	}

	// --- Phase 4: Write items + decrement stock ---
	for _, item := range req.Items {
		info := productMap[item.SKU]

		// Insert order item
		var oi model.SalesOrderItem
		err = tx.QueryRowContext(ctx,
			`INSERT INTO sales_order_items (order_id, sku, qty, unit_price) VALUES ($1, $2, $3, $4)
			 RETURNING id, order_id, sku, qty, unit_price`,
			order.ID, item.SKU, item.Qty, info.price,
		).Scan(&oi.ID, &oi.OrderID, &oi.SKU, &oi.Qty, &oi.UnitPrice)
		if err != nil {
			return nil, fmt.Errorf("insert order item: %w", err)
		}
		order.Items = append(order.Items, oi)

		// Decrement stock atomically
		result, err := tx.ExecContext(ctx,
			`UPDATE products SET stock = stock - $1 WHERE id = $2 AND stock >= $1`,
			item.Qty, info.id,
		)
		if err != nil {
			return nil, fmt.Errorf("decrement stock for %s: %w", item.SKU, err)
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			// Race condition caught: another request took the stock between our check and update
			return nil, fmt.Errorf("insufficient_stock: sku=%s available=0", item.SKU)
		}

		// Write stock_movement for each item
		_, err = tx.ExecContext(ctx,
			`INSERT INTO stock_movements (product_id, type, qty, note) VALUES ($1, 'out', $2, $3)`,
			info.id, item.Qty, fmt.Sprintf("Sale order #%d", order.ID),
		)
		if err != nil {
			return nil, fmt.Errorf("insert movement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &order, nil
}

// GetOrder returns a single order with its items.
func (r *OrderRepository) GetOrder(ctx context.Context, id int) (*model.SalesOrder, error) {
	var order model.SalesOrder
	err := r.db.QueryRowContext(ctx,
		`SELECT id, customer, status, total, created_at, cancelled_at FROM sales_orders WHERE id = $1`, id,
	).Scan(&order.ID, &order.Customer, &order.Status, &order.Total, &order.CreatedAt, &order.CancelledAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, order_id, sku, qty, unit_price FROM sales_order_items WHERE order_id = $1`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("get order items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item model.SalesOrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.SKU, &item.Qty, &item.UnitPrice); err != nil {
			return nil, err
		}
		order.Items = append(order.Items, item)
	}
	if order.Items == nil {
		order.Items = []model.SalesOrderItem{}
	}
	return &order, rows.Err()
}

// CancelOrder cancels an order and restores stock atomically.
func (r *OrderRepository) CancelOrder(ctx context.Context, id int) (*model.SalesOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Lock the order row
	var order model.SalesOrder
	err = tx.QueryRowContext(ctx,
		`SELECT id, customer, status, total, created_at, cancelled_at FROM sales_orders WHERE id = $1 FOR UPDATE`, id,
	).Scan(&order.ID, &order.Customer, &order.Status, &order.Total, &order.CreatedAt, &order.CancelledAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("not_found")
	}
	if err != nil {
		return nil, fmt.Errorf("get order for cancel: %w", err)
	}
	if order.Status == "CANCELLED" {
		return nil, fmt.Errorf("already_cancelled")
	}

	// Get order items
	rows, err := tx.QueryContext(ctx,
		`SELECT id, order_id, sku, qty, unit_price FROM sales_order_items WHERE order_id = $1`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("get items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item model.SalesOrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.SKU, &item.Qty, &item.UnitPrice); err != nil {
			return nil, err
		}
		order.Items = append(order.Items, item)
	}
	rows.Close()

	// Restore stock for each item + write movement
	for _, item := range order.Items {
		_, err = tx.ExecContext(ctx,
			`UPDATE products SET stock = stock + $1 WHERE sku = $2`, item.Qty, item.SKU,
		)
		if err != nil {
			return nil, fmt.Errorf("restore stock for %s: %w", item.SKU, err)
		}

		var productID int
		_ = tx.QueryRowContext(ctx, `SELECT id FROM products WHERE sku = $1`, item.SKU).Scan(&productID)
		_, err = tx.ExecContext(ctx,
			`INSERT INTO stock_movements (product_id, type, qty, note) VALUES ($1, 'in', $2, $3)`,
			productID, item.Qty, fmt.Sprintf("Cancel order #%d", id),
		)
		if err != nil {
			return nil, fmt.Errorf("insert movement: %w", err)
		}
	}

	// Mark order cancelled
	err = tx.QueryRowContext(ctx,
		`UPDATE sales_orders SET status = 'CANCELLED', cancelled_at = now() WHERE id = $1
		 RETURNING id, customer, status, total, created_at, cancelled_at`, id,
	).Scan(&order.ID, &order.Customer, &order.Status, &order.Total, &order.CreatedAt, &order.CancelledAt)
	if err != nil {
		return nil, fmt.Errorf("cancel order: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &order, nil
}

// ListOrders returns paginated sales orders with their items.
func (r *OrderRepository) ListOrders(ctx context.Context, page, limit int) ([]model.SalesOrder, int, error) {
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sales_orders`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, customer, status, total, created_at, cancelled_at
		 FROM sales_orders
		 ORDER BY created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var orders []model.SalesOrder
	for rows.Next() {
		var o model.SalesOrder
		if err := rows.Scan(&o.ID, &o.Customer, &o.Status, &o.Total, &o.CreatedAt, &o.CancelledAt); err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Fetch items for each order
	for i := range orders {
		itemRows, err := r.db.QueryContext(ctx,
			`SELECT id, order_id, sku, qty, unit_price FROM sales_order_items WHERE order_id = $1`,
			orders[i].ID,
		)
		if err == nil {
			for itemRows.Next() {
				var it model.SalesOrderItem
				if err := itemRows.Scan(&it.ID, &it.OrderID, &it.SKU, &it.Qty, &it.UnitPrice); err == nil {
					orders[i].Items = append(orders[i].Items, it)
				}
			}
			itemRows.Close()
		}
		if orders[i].Items == nil {
			orders[i].Items = []model.SalesOrderItem{}
		}
	}

	if orders == nil {
		orders = []model.SalesOrder{}
	}
	return orders, total, nil
}

