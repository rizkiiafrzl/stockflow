import { useState, useCallback } from 'react'
import type { Product } from '../types'
import { adjustStock } from '../api/client'
import { APIError } from '../types'

import { validateStockAdjustment } from '../utils/validation'

interface Props {
  product: Product
  onClose: () => void
  onSuccess: () => void
}

export default function StockModal({ product, onClose, onSuccess }: Props) {
  const [type, setType] = useState<'in' | 'out'>('in')
  const [qty, setQty] = useState('')
  const [note, setNote] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = useCallback(async () => {
    const res = validateStockAdjustment(qty, type)
    if (!res.valid) {
      setError(res.errors.qty ?? res.errors.type)
      return
    }
    const qtyNum = parseInt(qty, 10)

    setLoading(true)
    setError(null)
    try {
      await adjustStock(product.id, { type, qty: qtyNum, note: note || undefined })
      onSuccess()
    } catch (e) {
      if (e instanceof APIError && e.detail.code === 'INSUFFICIENT_STOCK') {
        setError(`Insufficient stock. Available: ${e.detail.available} units`)
      } else if (e instanceof APIError) {
        setError(e.detail.message ?? e.detail.code)
      } else {
        setError('Unexpected error. Please try again.')
      }
    } finally {
      setLoading(false)
    }
  }, [product.id, type, qty, note, onSuccess])

  return (
    <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal">
        <div className="modal-header">
          <div>
            <h2>Adjust Stock</h2>
            <p className="text-muted" style={{ fontSize: 12, marginTop: 2 }}>
              {product.name} · Current stock: <strong style={{ color: product.stock <= 10 ? 'var(--warning)' : 'var(--text)' }}>{product.stock}</strong>
            </p>
          </div>
          <button className="btn-icon" onClick={onClose}>✕</button>
        </div>

        <div className="modal-body">
          {/* Direction toggle */}
          <div className="field">
            <label>Direction</label>
            <div className="filter-tabs" style={{ width: 'fit-content' }}>
              <button
                className={`filter-tab ${type === 'in' ? 'active' : ''}`}
                style={type === 'in' ? { background: 'var(--success)' } : {}}
                onClick={() => setType('in')}
              >
                ↑ Stock In
              </button>
              <button
                className={`filter-tab ${type === 'out' ? 'active' : ''}`}
                style={type === 'out' ? { background: 'var(--danger)' } : {}}
                onClick={() => setType('out')}
              >
                ↓ Stock Out
              </button>
            </div>
          </div>

          <div className="field">
            <label>Quantity</label>
            <input
              type="number"
              min={1}
              placeholder="Enter quantity..."
              value={qty}
              onChange={(e) => { setQty(e.target.value); setError(null) }}
            />
          </div>

          <div className="field">
            <label>Note (optional)</label>
            <input
              type="text"
              placeholder="Reason for adjustment..."
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          </div>

          {error && (
            <div className="alert alert-error">
              <span>⚠</span>
              <span>{error}</span>
            </div>
          )}
        </div>

        <div className="modal-footer">
          <button className="btn btn-ghost" onClick={onClose} disabled={loading}>Cancel</button>
          <button className="btn btn-primary" onClick={handleSubmit} disabled={loading}>
            {loading ? 'Saving...' : `Confirm ${type === 'in' ? 'Stock In' : 'Stock Out'}`}
          </button>
        </div>
      </div>
    </div>
  )
}
