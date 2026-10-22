package repository

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"stockflow/backend/internal/model"
)

func TestOrderRepo_CreateOrder_SufficientStock(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	productRepo := NewProductRepository(db)
	orderRepo := NewOrderRepository(db)

	// Create test product with 10 stock
	sku := fmt.Sprintf("TEST-ORD-%d", time.Now().UnixNano())
	prod, err := productRepo.CreateProduct(ctx, model.CreateProductRequest{
		SKU:   sku,
		Name:  "Test Order Product",
		Price: 100000,
	})
	if err != nil {
		t.Fatalf("failed to create test product: %v", err)
	}

	// Add stock 10
	_, _, err = productRepo.AdjustStock(ctx, prod.ID, model.AdjustStockRequest{
		Type: "in",
		Qty:  10,
	})
	if err != nil {
		t.Fatalf("failed to add initial stock: %v", err)
	}

	// Create order for 3 items
	order, err := orderRepo.CreateOrder(ctx, model.CreateOrderRequest{
		Customer: "Budi Test",
		Items: []model.OrderItemRequest{
			{SKU: sku, Qty: 3},
		},
	})
	if err != nil {
		t.Fatalf("expected order to succeed, got error: %v", err)
	}

	if order.Status != "CREATED" {
		t.Errorf("expected status 'CREATED', got %q", order.Status)
	}
	expectedTotal := 3 * 100000
	if order.Total != expectedTotal {
		t.Errorf("expected total %d, got %d", expectedTotal, order.Total)
	}

	// Verify stock is now 7
	updatedProd, err := productRepo.GetProductByID(ctx, prod.ID)
	if err != nil {
		t.Fatalf("failed to fetch product: %v", err)
	}
	if updatedProd.Stock != 7 {
		t.Errorf("expected stock to be 7, got %d", updatedProd.Stock)
	}

	// Verify movement was created
	movements, err := productRepo.ListMovements(ctx, prod.ID)
	if err != nil {
		t.Fatalf("failed to fetch movements: %v", err)
	}
	var foundOut bool
	for _, m := range movements {
		if m.Type == "out" && m.Qty == 3 {
			foundOut = true
			break
		}
	}
	if !foundOut {
		t.Errorf("expected to find 'out' movement of qty 3, but not found")
	}
}

func TestOrderRepo_CreateOrder_InsufficientStock_Rollback(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	productRepo := NewProductRepository(db)
	orderRepo := NewOrderRepository(db)

	sku := fmt.Sprintf("TEST-NOSTK-%d", time.Now().UnixNano())
	prod, err := productRepo.CreateProduct(ctx, model.CreateProductRequest{
		SKU:   sku,
		Name:  "Test Low Stock Product",
		Price: 50000,
	})
	if err != nil {
		t.Fatalf("failed to create test product: %v", err)
	}

	// Add stock 2
	_, _, err = productRepo.AdjustStock(ctx, prod.ID, model.AdjustStockRequest{
		Type: "in",
		Qty:  2,
	})
	if err != nil {
		t.Fatalf("failed to add initial stock: %v", err)
	}

	// Attempt to order 5 (more than available 2)
	order, err := orderRepo.CreateOrder(ctx, model.CreateOrderRequest{
		Customer: "Greedy Customer",
		Items: []model.OrderItemRequest{
			{SKU: sku, Qty: 5},
		},
	})
	if err == nil {
		t.Fatalf("expected error for insufficient stock, but got order: %+v", order)
	}

	// CRITICAL: Verify stock remains exactly 2 (no partial deduction, full rollback)
	updatedProd, err := productRepo.GetProductByID(ctx, prod.ID)
	if err != nil {
		t.Fatalf("failed to fetch product: %v", err)
	}
	if updatedProd.Stock != 2 {
		t.Errorf("stock should remain 2 after failed order, got %d", updatedProd.Stock)
	}

	// Verify no order was inserted
	var count int
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sales_orders WHERE customer = 'Greedy Customer'").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 orders for customer, got %d", count)
	}
}

func TestOrderRepo_CreateOrder_Concurrency(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	productRepo := NewProductRepository(db)
	orderRepo := NewOrderRepository(db)

	sku := fmt.Sprintf("TEST-CONCUR-%d", time.Now().UnixNano())
	prod, err := productRepo.CreateProduct(ctx, model.CreateProductRequest{
		SKU:   sku,
		Name:  "Limited Edition Item",
		Price: 200000,
	})
	if err != nil {
		t.Fatalf("failed to create test product: %v", err)
	}

	// Initial stock: exactly 5 items
	initialStock := 5
	_, _, err = productRepo.AdjustStock(ctx, prod.ID, model.AdjustStockRequest{
		Type: "in",
		Qty:  initialStock,
	})
	if err != nil {
		t.Fatalf("failed to add stock: %v", err)
	}

	// Concurrently launch 15 requests, each wanting 1 item
	// Expected: exactly 5 succeed, 10 fail with insufficient stock
	concurrentReqs := 15
	var wg sync.WaitGroup
	var successCount int64
	var failCount int64

	startSignal := make(chan struct{})

	for i := 0; i < concurrentReqs; i++ {
		wg.Add(1)
		go func(customerID int) {
			defer wg.Done()
			<-startSignal

			_, err := orderRepo.CreateOrder(ctx, model.CreateOrderRequest{
				Customer: fmt.Sprintf("Buyer #%d", customerID),
				Items: []model.OrderItemRequest{
					{SKU: sku, Qty: 1},
				},
			})
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failCount, 1)
			}
		}(i)
	}

	// Unleash all goroutines at once
	close(startSignal)
	wg.Wait()

	if successCount != int64(initialStock) {
		t.Errorf("expected exactly %d successful orders, got %d", initialStock, successCount)
	}
	if failCount != int64(concurrentReqs-initialStock) {
		t.Errorf("expected %d failed orders, got %d", concurrentReqs-initialStock, failCount)
	}

	// Verify remaining stock is strictly 0 and NEVER negative
	updatedProd, err := productRepo.GetProductByID(ctx, prod.ID)
	if err != nil {
		t.Fatalf("failed to fetch product: %v", err)
	}
	if updatedProd.Stock != 0 {
		t.Errorf("expected final stock to be exactly 0, got %d", updatedProd.Stock)
	}
}

func TestOrderRepo_CancelOrder_RestoresStock(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	productRepo := NewProductRepository(db)
	orderRepo := NewOrderRepository(db)

	sku := fmt.Sprintf("TEST-CANCEL-%d", time.Now().UnixNano())
	prod, err := productRepo.CreateProduct(ctx, model.CreateProductRequest{
		SKU:   sku,
		Name:  "Cancellable Item",
		Price: 75000,
	})
	if err != nil {
		t.Fatalf("failed to create test product: %v", err)
	}

	// Initial stock 10
	_, _, err = productRepo.AdjustStock(ctx, prod.ID, model.AdjustStockRequest{
		Type: "in",
		Qty:  10,
	})
	if err != nil {
		t.Fatalf("failed to add stock: %v", err)
	}

	// Place order for 4 items -> stock becomes 6
	order, err := orderRepo.CreateOrder(ctx, model.CreateOrderRequest{
		Customer: "Cancel Tester",
		Items: []model.OrderItemRequest{
			{SKU: sku, Qty: 4},
		},
	})
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	// Cancel order
	cancelledOrder, err := orderRepo.CancelOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to cancel order: %v", err)
	}
	if cancelledOrder.Status != "CANCELLED" {
		t.Errorf("expected status 'CANCELLED', got %q", cancelledOrder.Status)
	}
	if cancelledOrder.CancelledAt == nil {
		t.Errorf("expected CancelledAt to be set")
	}

	// Verify stock is restored back to 10
	updatedProd, err := productRepo.GetProductByID(ctx, prod.ID)
	if err != nil {
		t.Fatalf("failed to fetch product: %v", err)
	}
	if updatedProd.Stock != 10 {
		t.Errorf("expected restored stock 10, got %d", updatedProd.Stock)
	}

	// Attempting to cancel again should fail with already_cancelled
	_, err = orderRepo.CancelOrder(ctx, order.ID)
	if err == nil {
		t.Errorf("expected error when cancelling an already cancelled order")
	}
}
