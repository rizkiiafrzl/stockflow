import { useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useAsync } from '../hooks/useAsync'
import { getOrder, cancelOrder } from '../api/client'
import { APIError } from '../types'
import { formatRupiah, formatRelative } from '../utils/format'

export default function OrderDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const orderId = parseInt(id ?? '0', 10)

  const [cancelling, setCancelling] = useState(false)
  const [cancelError, setCancelError] = useState<string | null>(null)
  const [showConfirm, setShowConfirm] = useState(false)

  const fetchFn = useCallback(() => getOrder(orderId), [orderId])
  const { data: order, loading, error, refetch } = useAsync(fetchFn, [orderId])

  const handleCancel = async () => {
    setCancelling(true)
    setCancelError(null)
    try {
      await cancelOrder(orderId)
      setShowConfirm(false)
      refetch()
    } catch (e) {
      if (e instanceof APIError && e.detail.code === 'ALREADY_CANCELLED') {
        setCancelError('This order is already cancelled.')
      } else {
        setCancelError('Failed to cancel order.')
      }
    } finally {
      setCancelling(false)
    }
  }

  if (loading) {
    return (
      <div className="state-container">
        <div className="state-icon" style={{ animation: 'spin 1s linear infinite' }}>⟳</div>
        <div className="state-title">Loading order...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="state-container">
        <div className="state-icon">⚡</div>
        <div className="state-title">Failed to load order</div>
        <div className="state-subtitle">{error.message}</div>
        <button className="btn btn-primary" style={{ marginTop: 12 }} onClick={refetch}>Retry</button>
      </div>
    )
  }

  if (!order) {
    return (
      <div className="state-container">
        <div className="state-icon">🔍</div>
        <div className="state-title">Order not found</div>
        <button className="btn btn-ghost" style={{ marginTop: 12 }} onClick={() => navigate('/')}>Back</button>
      </div>
    )
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <button className="btn btn-ghost btn-sm" style={{ marginBottom: 8 }} onClick={() => navigate('/')}>
            ← Back to Products
          </button>
          <h1 className="page-title">Order #{order.id}</h1>
          <p className="page-subtitle">{formatRelative(order.created_at)}</p>
        </div>
        {order.status === 'CREATED' && (
          <button className="btn btn-danger" onClick={() => setShowConfirm(true)}>
            Cancel Order
          </button>
        )}
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 300px', gap: 20 }}>
        {/* Items */}
        <div className="card">
          <h3 style={{ fontSize: 14, fontWeight: 600, marginBottom: 16 }}>Order Items</h3>
          <table>
            <thead>
              <tr>
                <th>SKU</th>
                <th>Qty</th>
                <th>Unit Price</th>
                <th>Subtotal</th>
              </tr>
            </thead>
            <tbody>
              {order.items.map(item => (
                <tr key={item.id}>
                  <td><span className="sku-tag">{item.sku}</span></td>
                  <td>{item.qty}</td>
                  <td>{formatRupiah(item.unit_price)}</td>
                  <td style={{ fontWeight: 600 }}>{formatRupiah(item.unit_price * item.qty)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Summary */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div className="card">
            <h3 style={{ fontSize: 14, fontWeight: 600, marginBottom: 16 }}>Order Summary</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}>
                <span className="text-muted">Customer</span>
                <strong>{order.customer}</strong>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}>
                <span className="text-muted">Status</span>
                <span className={`badge badge-${order.status.toLowerCase()}`}>
                  {order.status}
                </span>
              </div>
              {order.cancelled_at && (
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}>
                  <span className="text-muted">Cancelled</span>
                  <span>{formatRelative(order.cancelled_at)}</span>
                </div>
              )}
              <hr style={{ border: 'none', borderTop: '1px solid var(--border)' }} />
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span className="text-muted">Total</span>
                <strong style={{ fontSize: 18, color: 'var(--accent)' }}>{formatRupiah(order.total)}</strong>
              </div>
            </div>
          </div>

          {order.status === 'CANCELLED' && (
            <div className="alert alert-warning">
              <span>ℹ</span>
              <span>Stock has been restored to inventory.</span>
            </div>
          )}
        </div>
      </div>

      {/* Cancel confirm dialog */}
      {showConfirm && (
        <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && setShowConfirm(false)}>
          <div className="modal" style={{ maxWidth: 380 }}>
            <div className="modal-header">
              <h2>Cancel Order?</h2>
              <button className="btn-icon" onClick={() => setShowConfirm(false)}>✕</button>
            </div>
            <div className="modal-body">
              <p>Cancelling order #{order.id} will restore stock for all items. This cannot be undone.</p>
              {cancelError && <div className="alert alert-error"><span>⚠</span><span>{cancelError}</span></div>}
            </div>
            <div className="modal-footer">
              <button className="btn btn-ghost" onClick={() => setShowConfirm(false)} disabled={cancelling}>Back</button>
              <button className="btn btn-danger" onClick={handleCancel} disabled={cancelling}>
                {cancelling ? 'Cancelling...' : 'Yes, Cancel Order'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
