import { describe, it, expect } from 'vitest'
import { validateProduct, validateStockAdjustment, validateOrder } from './validation'

describe('validateProduct', () => {
  it('should pass with valid product inputs', () => {
    const res = validateProduct('SKU-100', 'Mechanical Keyboard', 450000)
    expect(res.valid).toBe(true)
    expect(res.errors).toEqual({})
  })

  it('should require SKU for new products', () => {
    const res = validateProduct('', 'Mechanical Keyboard', 450000, false)
    expect(res.valid).toBe(false)
    expect(res.errors.sku).toBe('SKU is required')
  })

  it('should reject whitespace-only SKU', () => {
    const res = validateProduct('   ', 'Mechanical Keyboard', 450000)
    expect(res.valid).toBe(false)
    expect(res.errors.sku).toBe('SKU is required')
  })

  it('should not require SKU when editing existing product', () => {
    const res = validateProduct('', 'Updated Keyboard', 500000, true)
    expect(res.valid).toBe(true)
    expect(res.errors.sku).toBeUndefined()
  })

  it('should require product name', () => {
    const res = validateProduct('SKU-100', '  ', 450000)
    expect(res.valid).toBe(false)
    expect(res.errors.name).toBe('Name is required')
  })

  it('should reject negative price', () => {
    const res = validateProduct('SKU-100', 'Mouse', -1000)
    expect(res.valid).toBe(false)
    expect(res.errors.price).toBe('Price must be a non-negative integer')
  })

  it('should allow price of 0 (free promotional item)', () => {
    const res = validateProduct('SKU-FREE', 'Sticker Pack', 0)
    expect(res.valid).toBe(true)
  })

  it('should reject decimal / float price (integer IDR required)', () => {
    const res = validateProduct('SKU-100', 'Mouse', 15.75)
    expect(res.valid).toBe(false)
    expect(res.errors.price).toBe('Price must be an integer (no decimals)')
  })
})

describe('validateStockAdjustment', () => {
  it('should pass for valid stock in', () => {
    const res = validateStockAdjustment(10, 'in')
    expect(res.valid).toBe(true)
  })

  it('should pass for valid stock out', () => {
    const res = validateStockAdjustment('5', 'out')
    expect(res.valid).toBe(true)
  })

  it('should reject qty of 0', () => {
    const res = validateStockAdjustment(0, 'in')
    expect(res.valid).toBe(false)
    expect(res.errors.qty).toBe('Quantity must be an integer > 0')
  })

  it('should reject negative qty', () => {
    const res = validateStockAdjustment(-5, 'out')
    expect(res.valid).toBe(false)
    expect(res.errors.qty).toBe('Quantity must be an integer > 0')
  })

  it('should reject non-integer qty', () => {
    const res = validateStockAdjustment(3.5, 'in')
    expect(res.valid).toBe(false)
    expect(res.errors.qty).toBe('Quantity must be a whole integer')
  })
})

describe('validateOrder', () => {
  it('should pass with valid order data', () => {
    const res = validateOrder('PT Maju Bersama', [
      { sku: 'SKU-001', qty: 2 },
      { sku: 'SKU-002', qty: 5 },
    ])
    expect(res.valid).toBe(true)
    expect(res.errors).toEqual({})
  })

  it('should require customer name', () => {
    const res = validateOrder('', [{ sku: 'SKU-001', qty: 1 }])
    expect(res.valid).toBe(false)
    expect(res.errors.customer).toBe('Customer name is required')
  })

  it('should require at least one item', () => {
    const res = validateOrder('Customer A', [])
    expect(res.valid).toBe(false)
    expect(res.errors.items).toBe('At least one item is required')
  })

  it('should reject items with empty SKU', () => {
    const res = validateOrder('Customer A', [
      { sku: '', qty: 1 },
    ])
    expect(res.valid).toBe(false)
    expect(res.errors.sku_0).toBe('SKU is required')
  })

  it('should reject items with qty <= 0', () => {
    const res = validateOrder('Customer A', [
      { sku: 'SKU-001', qty: 0 },
      { sku: 'SKU-002', qty: -2 },
    ])
    expect(res.valid).toBe(false)
    expect(res.errors.qty_0).toBe('Qty must be > 0')
    expect(res.errors.qty_1).toBe('Qty must be > 0')
  })

  it('should reject duplicate SKUs in the same order', () => {
    const res = validateOrder('Customer A', [
      { sku: 'SKU-001', qty: 1 },
      { sku: 'SKU-001', qty: 2 },
    ])
    expect(res.valid).toBe(false)
    expect(res.errors.sku_1).toBe('Duplicate SKU "SKU-001"')
  })
})
