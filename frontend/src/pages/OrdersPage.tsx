import { useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAsync } from '../hooks/useAsync'
import { getOrders } from '../api/client'
import { formatRupiah, formatRelative } from '../utils/format'
import type { SalesOrder } from '../types'

type StatusFilter = 'all' | 'created' | 'cancelled'
const LIMIT = 15

export default function OrdersPage() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [filter, setFilter] = useState<StatusFilter>('all')

  const fetchFn = useCallback(
    () => getOrders({ page, limit: LIMIT }),
    [page]
  )

  const { data, loading, error, refetch } = useAsync(fetchFn, [page])

  const filteredOrders = (data?.data ?? []).filter((o: SalesOrder) => {
    if (filter === 'created') return o.status === 'CREATED'
    if (filter === 'cancelled') return o.status === 'CANCELLED'
    return true
  })

  const totalPages = data ? Math.ceil(data.total / LIMIT) : 1

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Sales Orders</h1>
          <p className="page-subtitle">
            {data ? `${data.total} orders total` : 'Loading orders...'}
          </p>
        </div>
      </div>

      {/* Filter Tabs */}
      <div className="toolbar">
        <div className="filter-tabs">
          {(['all', 'created', 'cancelled'] as StatusFilter[]).map((f) => (
            <button
              key={f}
              className={`filter-tab ${filter === f ? 'active' : ''}`}
              onClick={() => setFilter(f)}
            >
              {f.charAt(0).toUpperCase() + f.slice(1)}
            </button>
          ))}
        </div>
      </div>

      {/* Orders Table */}
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Order ID</th>
              <th>Customer</th>
              <th>Items</th>
              <th>Total</th>
              <th>Status</th>
              <th>Date</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              Array.from({ length: 6 }).map((_, i) => (
                <tr key={i} className="skeleton-row">
                  {Array.from({ length: 7 }).map((_, j) => (
                    <td key={j}><div className="skeleton skeleton-cell" style={{ width: j === 1 ? '70%' : '50%' }} /></td>
                  ))}
                </tr>
              ))
            )}

            {!loading && error && (
              <tr>
                <td colSpan={7}>
                  <div className="state-container">
                    <div className="state-title">Failed to load orders</div>
                    <div className="state-subtitle">{error.message}</div>
                    <button className="btn btn-primary" style={{ marginTop: 12 }} onClick={refetch}>Retry</button>
                  </div>
                </td>
              </tr>
            )}

            {!loading && !error && filteredOrders.length === 0 && (
              <tr>
                <td colSpan={7}>
                  <div className="state-container">
                    <div className="state-title">No orders found</div>
                    <div className="state-subtitle">There are no sales orders matching your filter.</div>
                  </div>
                </td>
              </tr>
            )}

            {!loading && filteredOrders.map((o) => (
              <tr key={o.id}>
                <td>
                  <span className="sku-tag">#{o.id}</span>
                </td>
                <td style={{ fontWeight: 600 }}>{o.customer}</td>
                <td>
                  <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>
                    {o.items?.length ? `${o.items.length} item${o.items.length > 1 ? 's' : ''}` : 'No items'}
                  </span>
                </td>
                <td style={{ fontWeight: 700, color: 'var(--accent)' }}>
                  {formatRupiah(o.total)}
                </td>
                <td>
                  <span className={`badge ${o.status === 'CREATED' ? 'badge-active' : 'badge-inactive'}`}>
                    {o.status}
                  </span>
                </td>
                <td className="text-muted">
                  {formatRelative(o.created_at)}
                </td>
                <td>
                  <button
                    className="btn btn-ghost btn-sm"
                    onClick={() => navigate(`/orders/${o.id}`)}
                  >
                    View Detail
                  </button>
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
              <button disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>Prev</button>
              {Array.from({ length: totalPages }, (_, i) => (
                <button key={i} className={page === i + 1 ? 'active' : ''} onClick={() => setPage(i + 1)}>
                  {i + 1}
                </button>
              ))}
              <button disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>Next</button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
