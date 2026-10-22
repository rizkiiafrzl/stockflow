/**
 * API Client — single module that owns all fetch calls and error parsing.
 * Components NEVER build URLs or parse error envelopes directly.
 */
import type {
  AdjustStockRequest,
  CreateOrderRequest,
  CreateProductRequest,
  MovementsResponse,
  ProductsResponse,
  SalesOrder,
  StockMovement,
  UpdateProductRequest,
  Product,
  OrdersResponse,
} from '../types'
import { APIError } from '../types'

const BASE = '/api'

async function request<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })

  if (res.status === 204) return undefined as T

  const json = await res.json().catch(() => null)

  if (!res.ok) {
    const detail = json?.error ?? { code: 'UNKNOWN', message: res.statusText }
    throw new APIError(res.status, detail)
  }

  return json as T
}

// ── Products ──────────────────────────────────────────────────────────────

export function getProducts(params: {
  search?: string
  page?: number
  limit?: number
}): Promise<ProductsResponse> {
  const q = new URLSearchParams()
  if (params.search) q.set('search', params.search)
  if (params.page) q.set('page', String(params.page))
  if (params.limit) q.set('limit', String(params.limit))
  const qs = q.toString() ? `?${q.toString()}` : ''
  return request<ProductsResponse>(`/products${qs}`)
}

export function createProduct(data: CreateProductRequest): Promise<Product> {
  return request<Product>('/products', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function updateProduct(id: number, data: UpdateProductRequest): Promise<Product> {
  return request<Product>(`/products/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function deleteProduct(id: number): Promise<void> {
  return request<void>(`/products/${id}`, { method: 'DELETE' })
}

export function adjustStock(id: number, data: AdjustStockRequest): Promise<StockMovement> {
  return request<StockMovement>(`/products/${id}/stock`, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function getMovements(id: number): Promise<MovementsResponse> {
  return request<MovementsResponse>(`/products/${id}/movements`)
}

// ── Orders ────────────────────────────────────────────────────────────────

export function getOrders(params?: { page?: number; limit?: number }): Promise<OrdersResponse> {
  const q = new URLSearchParams()
  if (params?.page) q.set('page', String(params.page))
  if (params?.limit) q.set('limit', String(params.limit))
  const qs = q.toString() ? `?${q.toString()}` : ''
  return request<OrdersResponse>(`/orders${qs}`)
}

export function createOrder(data: CreateOrderRequest): Promise<SalesOrder> {
  return request<SalesOrder>('/orders', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function getOrder(id: number): Promise<SalesOrder> {
  return request<SalesOrder>(`/orders/${id}`)
}

export function cancelOrder(id: number): Promise<SalesOrder> {
  return request<SalesOrder>(`/orders/${id}/cancel`, { method: 'POST' })
}
