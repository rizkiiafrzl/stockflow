import { useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAsync, useDebounce } from '../hooks/useAsync'
import { getProducts, deleteProduct } from '../api/client'
import { APIError } from '../types'
import type { Product } from '../types'
import { formatRupiah, formatRelative } from '../utils/format'
import SummaryCards from '../components/SummaryCards'
import StockModal from '../components/StockModal'
import ProductModal from '../components/ProductModal'
import CreateOrderModal from '../components/CreateOrderModal'
import type { SalesOrder } from '../types'

type Filter = 'all' | 'active' | 'inactive' | 'low'

const LIMIT = 15

export default function ProductsPage() {
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<Filter>('all')
  const [page, setPage] = useState(1)
  const debouncedSearch = useDebounce(search, 400)

  const [stockTarget, setStockTarget] = useState<Product | null>(null)
  const [editTarget, setEditTarget] = useState<Product | null | undefined>(undefined) // undefined = hidden, null = new
  const [showOrder, setShowOrder] = useState(false)
  const [createdOrder, setCreatedOrder] = useState<SalesOrder | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<Product | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const fetchFn = useCallback(
    () => getProducts({ search: debouncedSearch, page, limit: LIMIT }),
    [debouncedSearch, page]
  )

  const { data, loading, error, refetch } = useAsync(fetchFn, [debouncedSearch, page])

  // Client-side filter (on top of search)
  const filtered = (data?.data ?? []).filter(p => {
    if (filter === 'active') return p.is_active
    if (filter === 'inactive') return !p.is_active
    if (filter === 'low') return p.stock <= 10 && p.is_active
    return true
  })

  const handleDelete = async (p: Product) => {
    setDeleteError(null)
    try {
      await deleteProduct(p.id)
      setDeleteConfirm(null)
      refetch()
    } catch (e) {
      if (e instanceof APIError && e.detail.code === 'PRODUCT_HAS_HISTORY') {
        setDeleteError('Cannot delete: this product has movement history.')
      } else {
        setDeleteError('Failed to delete product.')
      }
    }
  }

  const totalPages = data ? Math.ceil(data.total / LIMIT) : 1

  return (
    <div>
      {/* Summary cards derived from ALL loaded products */}
      {data && <SummaryCards products={data.data} />}

      <div className="page-header">
        <div>
          <h1 className="page-title">Products</h1>
          <p className="page-subtitle">
            {data ? `${data.total} products total` : 'Loading...'}
          </p>
        </div>
        <div style={{ display: 'flex', gap: 10 }}>
          <button className="btn btn-ghost" onClick={() => setShowOrder(true)}>
            🛒 New Order
          </button>
          <button className="btn btn-primary" onClick={() => setEditTarget(null)}>
            + New Product
          </button>
        </div>
      </div>

      {/* Toolbar */}
      <div className="toolbar">
        <div className="search-box">
          <span className="search-icon">🔍</span>
          <input
            type="text"
            placeholder="Search by SKU or name..."
            value={search}
            onChange={e => { setSearch(e.target.value); setPage(1) }}
          />
        </div>
        <div className="filter-tabs">
          {(['all', 'active', 'inactive', 'low'] as Filter[]).map(f => (
            <button
              key={f}
              className={`filter-tab ${filter === f ? 'active' : ''}`}
              onClick={() => setFilter(f)}
            >
              {f === 'low' ? '⚠ Low Stock' : f.charAt(0).toUpperCase() + f.slice(1)}
            </button>
          ))}
        </div>
      </div>

      {/* Table */}
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>SKU</th>
              <th>Name</th>
              <th>Price</th>
              <th>Stock</th>
              <th>Status</th>
              <th>Last Movement</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              Array.from({ length: 6 }).map((_, i) => (
                <tr key={i} className="skeleton-row">
                  {Array.from({ length: 7 }).map((_, j) => (
                    <td key={j}><div className="skeleton skeleton-cell" style={{ width: j === 1 ? '80%' : '60%' }} /></td>
                  ))}
                </tr>
              ))
            )}

            {!loading && error && (
              <tr>
                <td colSpan={7}>
                  <div className="state-container">
                    <div className="state-icon">⚡</div>
                    <div className="state-title">Failed to load</div>
                    <div className="state-subtitle">{error.message}</div>
                    <button className="btn btn-primary" style={{ marginTop: 12 }} onClick={refetch}>Retry</button>
                  </div>
                </td>
              </tr>
            )}

            {!loading && !error && filtered.length === 0 && (
              <tr>
                <td colSpan={7}>
                  <div className="state-container">
                    <div className="state-icon">📭</div>
                    <div className="state-title">No products found</div>
                    <div className="state-subtitle">Try a different search or filter</div>
                  </div>
                </td>
              </tr>
            )}

            {!loading && filtered.map(p => (
              <tr key={p.id}>
                <td><span className="sku-tag">{p.sku}</span></td>
                <td style={{ fontWeight: 500 }}>{p.name}</td>
                <td>{formatRupiah(p.price)}</td>
                <td>
                  <span style={{ color: p.stock <= 10 ? 'var(--warning)' : 'var(--text)', fontWeight: 600 }}>
                    {p.stock}
                  </span>
                </td>
                <td>
                  <span className={`badge ${p.is_active ? 'badge-active' : 'badge-inactive'}`}>
                    {p.is_active ? '● Active' : '○ Inactive'}
                  </span>
                  {p.stock <= 10 && p.is_active && (
                    <span className="badge badge-low" style={{ marginLeft: 6 }}>Low</span>
                  )}
                </td>
                <td className="text-muted">{formatRelative(p.last_movement_at)}</td>
                <td>
                  <div style={{ display: 'flex', gap: 6 }}>
                    <button
                      className="btn-icon"
                      title="Adjust Stock"
                      onClick={() => setStockTarget(p)}
                    >📊</button>
                    <button
                      className="btn-icon"
                      title="Edit"
                      onClick={() => setEditTarget(p)}
                    >✏️</button>
                    <button
                      className="btn-icon"
                      title="Delete"
                      style={{ color: 'var(--danger)' }}
                      onClick={() => { setDeleteConfirm(p); setDeleteError(null) }}
                    >🗑</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {/* Pagination */}
        {!loading && data && totalPages > 1 && (
          <div className="pagination">
            <span>Showing {(page - 1) * LIMIT + 1}–{Math.min(page * LIMIT, data.total)} of {data.total}</span>
            <div className="pagination-controls">
              <button disabled={page <= 1} onClick={() => setPage(p => p - 1)}>← Prev</button>
              {Array.from({ length: totalPages }, (_, i) => (
                <button key={i} className={page === i + 1 ? 'active' : ''} onClick={() => setPage(i + 1)}>
                  {i + 1}
                </button>
              ))}
              <button disabled={page >= totalPages} onClick={() => setPage(p => p + 1)}>Next →</button>
            </div>
          </div>
        )}
      </div>

      {/* Delete confirm dialog */}
      {deleteConfirm && (
        <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && setDeleteConfirm(null)}>
          <div className="modal" style={{ maxWidth: 380 }}>
            <div className="modal-header">
              <h2>Delete Product?</h2>
              <button className="btn-icon" onClick={() => setDeleteConfirm(null)}>✕</button>
            </div>
            <div className="modal-body">
              <p>Are you sure you want to delete <strong>{deleteConfirm.name}</strong>?</p>
              {deleteError && <div className="alert alert-error"><span>⚠</span><span>{deleteError}</span></div>}
            </div>
            <div className="modal-footer">
              <button className="btn btn-ghost" onClick={() => setDeleteConfirm(null)}>Cancel</button>
              <button className="btn btn-danger" onClick={() => handleDelete(deleteConfirm)}>Delete</button>
            </div>
          </div>
        </div>
      )}

      {/* Created order success */}
      {createdOrder && (
        <div className="modal-overlay" onClick={() => setCreatedOrder(null)}>
          <div className="modal" style={{ maxWidth: 420 }}>
            <div className="modal-header">
              <h2>✅ Order Created!</h2>
              <button className="btn-icon" onClick={() => setCreatedOrder(null)}>✕</button>
            </div>
            <div className="modal-body">
              <div className="alert alert-success">
                Order #{createdOrder.id} for <strong>{createdOrder.customer}</strong> created successfully.
              </div>
              <div style={{ marginTop: 4 }}>
                <p style={{ fontSize: 13, color: 'var(--text-muted)', marginBottom: 8 }}>Items:</p>
                {createdOrder.items.map(it => (
                  <div key={it.id} style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, padding: '4px 0', borderBottom: '1px solid var(--border)' }}>
                    <span><span className="sku-tag">{it.sku}</span> × {it.qty}</span>
                    <span>{formatRupiah(it.unit_price * it.qty)}</span>
                  </div>
                ))}
                <div style={{ display: 'flex', justifyContent: 'space-between', fontWeight: 700, marginTop: 10, fontSize: 15 }}>
                  <span>Total</span>
                  <span style={{ color: 'var(--accent)' }}>{formatRupiah(createdOrder.total)}</span>
                </div>
              </div>
            </div>
            <div className="modal-footer" style={{ justifyContent: 'space-between' }}>
              <button
                className="btn btn-ghost"
                onClick={() => {
                  const targetId = createdOrder.id
                  setCreatedOrder(null)
                  navigate(`/orders/${targetId}`)
                }}
              >
                View Order #{createdOrder.id} Detail →
              </button>
              <button className="btn btn-primary" onClick={() => { setCreatedOrder(null); refetch() }}>Done</button>
            </div>
          </div>
        </div>
      )}

      {/* Modals */}
      {stockTarget && (
        <StockModal
          product={stockTarget}
          onClose={() => setStockTarget(null)}
          onSuccess={() => { setStockTarget(null); refetch() }}
        />
      )}

      {editTarget !== undefined && (
        <ProductModal
          product={editTarget}
          onClose={() => setEditTarget(undefined)}
          onSuccess={() => { setEditTarget(undefined); refetch() }}
        />
      )}

      {showOrder && (
        <CreateOrderModal
          onClose={() => setShowOrder(false)}
          onSuccess={(order) => { setShowOrder(false); setCreatedOrder(order); refetch() }}
        />
      )}
    </div>
  )
}
