// Shared API types — no `any` on domain data
export interface Product {
  id: number
  sku: string
  name: string
  price: number
  stock: number
  is_active: boolean
  created_at: string
  last_movement_at: string | null
}

export interface StockMovement {
  id: number
  product_id: number
  type: 'in' | 'out'
  qty: number
  note: string | null
  created_at: string
}

export interface SalesOrderItem {
  id: number
  order_id: number
  sku: string
  qty: number
  unit_price: number
}

export interface SalesOrder {
  id: number
  customer: string
  status: 'CREATED' | 'CANCELLED'
  total: number
  created_at: string
  cancelled_at: string | null
  items: SalesOrderItem[]
}

// Request types
export interface CreateProductRequest {
  sku: string
  name: string
  price: number
}

export interface UpdateProductRequest {
  name: string
  price: number
  is_active?: boolean
}

export interface AdjustStockRequest {
  type: 'in' | 'out'
  qty: number
  note?: string
}

export interface CreateOrderRequest {
  customer: string
  items: { sku: string; qty: number }[]
}

// Response envelope types
export interface ProductsResponse {
  data: Product[]
  page: number
  limit: number
  total: number
}

export interface MovementsResponse {
  data: StockMovement[]
}

export interface APIErrorDetail {
  code: string
  message?: string
  messages?: string[]
  sku?: string
  available?: number
}

export interface APIErrorResponse {
  error: APIErrorDetail
}

export class APIError extends Error {
  public status: number
  public detail: APIErrorDetail

  constructor(status: number, detail: APIErrorDetail) {
    super(detail.message ?? detail.code)
    this.status = status
    this.detail = detail
  }
}
