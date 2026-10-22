export interface OrderItemInput {
  sku: string
  qty: string | number
}

export interface ValidationResult {
  valid: boolean
  errors: Record<string, string>
}

/**
 * Validates product creation and update inputs.
 */
export function validateProduct(sku: string, name: string, price: string | number, isEdit = false): ValidationResult {
  const errors: Record<string, string> = {}

  if (!isEdit && (!sku || !sku.trim())) {
    errors.sku = 'SKU is required'
  }

  if (!name || !name.trim()) {
    errors.name = 'Name is required'
  }

  const p = typeof price === 'number' ? price : parseInt(price, 10)
  if (price === '' || price === undefined || price === null || isNaN(p) || p < 0) {
    errors.price = 'Price must be a non-negative integer'
  } else if (!Number.isInteger(Number(price))) {
    errors.price = 'Price must be an integer (no decimals)'
  }

  return {
    valid: Object.keys(errors).length === 0,
    errors,
  }
}

/**
 * Validates stock adjustment input.
 */
export function validateStockAdjustment(qty: string | number, type: 'in' | 'out'): ValidationResult {
  const errors: Record<string, string> = {}

  if (type !== 'in' && type !== 'out') {
    errors.type = 'Direction must be "in" or "out"'
  }

  const q = typeof qty === 'number' ? qty : parseInt(qty, 10)
  if (!qty || isNaN(q) || q <= 0) {
    errors.qty = 'Quantity must be an integer > 0'
  } else if (!Number.isInteger(Number(qty))) {
    errors.qty = 'Quantity must be a whole integer'
  }

  return {
    valid: Object.keys(errors).length === 0,
    errors,
  }
}

/**
 * Validates sales order creation inputs.
 */
export function validateOrder(customer: string, items: OrderItemInput[]): ValidationResult {
  const errors: Record<string, string> = {}

  if (!customer || !customer.trim()) {
    errors.customer = 'Customer name is required'
  }

  if (!items || items.length === 0) {
    errors.items = 'At least one item is required'
    return { valid: false, errors }
  }

  const skuSet = new Set<string>()

  items.forEach((it, i) => {
    const trimmedSku = it.sku ? it.sku.trim() : ''
    if (!trimmedSku) {
      errors[`sku_${i}`] = 'SKU is required'
    } else if (skuSet.has(trimmedSku)) {
      errors[`sku_${i}`] = `Duplicate SKU "${trimmedSku}"`
    } else {
      skuSet.add(trimmedSku)
    }

    const q = typeof it.qty === 'number' ? it.qty : parseInt(it.qty, 10)
    if (!it.qty || isNaN(q) || q <= 0) {
      errors[`qty_${i}`] = 'Qty must be > 0'
    } else if (!Number.isInteger(Number(it.qty))) {
      errors[`qty_${i}`] = 'Qty must be an integer'
    }
  })

  return {
    valid: Object.keys(errors).length === 0,
    errors,
  }
}
