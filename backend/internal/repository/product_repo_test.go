package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"stockflow/backend/internal/model"
)

func TestProductRepo_AdjustStock_SuccessAndInsufficient(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := NewProductRepository(db)

	sku := fmt.Sprintf("TEST-ADJ-%d", time.Now().UnixNano())
	prod, err := repo.CreateProduct(ctx, model.CreateProductRequest{
		SKU:   sku,
		Name:  "Stock Adjustment Test",
		Price: 15000,
	})
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	// 1. Stock In: add 20
	noteIn := "Restock delivery"
	mvIn, newStock, err := repo.AdjustStock(ctx, prod.ID, model.AdjustStockRequest{
		Type: "in",
		Qty:  20,
		Note: &noteIn,
	})
	if err != nil {
		t.Fatalf("failed to add stock: %v", err)
	}
	if newStock != 20 {
		t.Errorf("expected stock 20, got %d", newStock)
	}
	if mvIn.Type != "in" || mvIn.Qty != 20 {
		t.Errorf("unexpected movement data: %+v", mvIn)
	}

	// 2. Stock Out: remove 8 -> stock should be 12
	noteOut := "Damaged goods discarded"
	mvOut, newStock, err := repo.AdjustStock(ctx, prod.ID, model.AdjustStockRequest{
		Type: "out",
		Qty:  8,
		Note: &noteOut,
	})
	if err != nil {
		t.Fatalf("failed to deduct stock: %v", err)
	}
	if newStock != 12 {
		t.Errorf("expected stock 12, got %d", newStock)
	}
	if mvOut.Type != "out" || mvOut.Qty != 8 {
		t.Errorf("unexpected movement data: %+v", mvOut)
	}

	// 3. Stock Out: attempt to remove 15 (> 12 available) -> must fail with insufficient stock
	_, available, err := repo.AdjustStock(ctx, prod.ID, model.AdjustStockRequest{
		Type: "out",
		Qty:  15,
	})
	if err == nil {
		t.Fatalf("expected error for excessive stock deduction, got none")
	}
	if available != 12 {
		t.Errorf("expected available to report 12, got %d", available)
	}

	// Verify stock is still 12 (untouched)
	p, err := repo.GetProductByID(ctx, prod.ID)
	if err != nil {
		t.Fatalf("failed to get product: %v", err)
	}
	if p.Stock != 12 {
		t.Errorf("expected stock to remain 12, got %d", p.Stock)
	}
}

func TestProductRepo_DeleteProduct_WithAndWithoutHistory(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := NewProductRepository(db)

	// Case 1: Fresh product with no history -> can be deleted
	skuClean := fmt.Sprintf("TEST-DEL-CLEAN-%d", time.Now().UnixNano())
	prodClean, err := repo.CreateProduct(ctx, model.CreateProductRequest{
		SKU:   skuClean,
		Name:  "Clean Product",
		Price: 50000,
	})
	if err != nil {
		t.Fatalf("failed to create clean product: %v", err)
	}

	exists, hasHistory, err := repo.DeleteProduct(ctx, prodClean.ID)
	if err != nil {
		t.Fatalf("delete clean product failed: %v", err)
	}
	if !exists {
		t.Errorf("expected exists to be true")
	}
	if hasHistory {
		t.Errorf("expected hasHistory to be false for clean product")
	}

	// Verify it was really deleted
	deletedP, err := repo.GetProductByID(ctx, prodClean.ID)
	if err != nil {
		t.Fatalf("error checking deleted product: %v", err)
	}
	if deletedP != nil {
		t.Errorf("expected product to be nil after deletion")
	}

	// Case 2: Product WITH movement history -> cannot be deleted (returns hasHistory = true)
	skuWithHistory := fmt.Sprintf("TEST-DEL-HIST-%d", time.Now().UnixNano())
	prodHist, err := repo.CreateProduct(ctx, model.CreateProductRequest{
		SKU:   skuWithHistory,
		Name:  "Product With History",
		Price: 50000,
	})
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	// Add stock movement
	_, _, err = repo.AdjustStock(ctx, prodHist.ID, model.AdjustStockRequest{
		Type: "in",
		Qty:  5,
	})
	if err != nil {
		t.Fatalf("failed to adjust stock: %v", err)
	}

	exists, hasHistory, err = repo.DeleteProduct(ctx, prodHist.ID)
	if err != nil {
		t.Fatalf("delete product with history returned unexpected error: %v", err)
	}
	if !exists {
		t.Errorf("expected exists to be true")
	}
	if !hasHistory {
		t.Errorf("expected hasHistory to be true for product with movements")
	}

	// Verify product still exists in DB
	stillThere, err := repo.GetProductByID(ctx, prodHist.ID)
	if err != nil {
		t.Fatalf("error checking product: %v", err)
	}
	if stillThere == nil {
		t.Errorf("product with history should NOT have been deleted from DB")
	}
}
