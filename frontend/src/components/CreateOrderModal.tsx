import { useState, useCallback } from 'react'
import { createOrder } from '../api/client'
import { APIError } from '../types'
import type { SalesOrder } from '../types'
import { formatRupiah } from '../utils/format'

import { validateOrder } from '../utils/validation'

interface Props {
  onClose: () => void
  onSuccess: (order: SalesOrder) => void
}

interface Item { sku: string; qty: string }

export default function CreateOrderModal({ onClose, onSuccess }: Props) {
  const [customer, setCustomer] = useState('')
  const [items, setItems] = useState<Item[]>([{ sku: '', qty: '' }])
  const [loading, setLoading] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [stockError, setStockError] = useState<{ sku: string; available: number } | null>(null)
  const [globalError, setGlobalError] = useState<string | null>(null)

  const addItem = () => setItems(prev => [...prev, { sku: '', qty: '' }])
  const removeItem = (i: number) => setItems(prev => prev.filter((_, idx) => idx !== i))
  const updateItem = (i: number, field: keyof Item, val: string) => {
    setItems(prev => prev.map((it, idx) => idx === i ? { ...it, [field]: val } : it))
    setErrors({})
    setStockError(null)
  }

  const validate = (): boolean => {
    const res = validateOrder(customer, items)
    setErrors(res.errors)
    return res.valid
  }

  const handleSubmit = useCallback(async () => {
    if (!validate()) return

    setLoading(true)
    setGlobalError(null)
    setStockError(null)
    try {
      const order = await createOrder({
        customer,
        items: items.map(it => ({ sku: it.sku, qty: parseInt(it.qty, 10) })),
      })
      onSuccess(order)
    } catch (e) {
      if (e instanceof APIError) {
        if (e.detail.code === 'INSUFFICIENT_STOCK') {
          setStockError({ sku: e.detail.sku!, available: e.detail.available! })
        } else if (e.detail.code === 'VALIDATION_ERROR' || e.detail.code === 'UNKNOWN_SKU') {
          setGlobalError(e.detail.message ?? e.detail.code)
        } else {
          setGlobalError(e.detail.message ?? e.detail.code)
        }
      } else {
        setGlobalError('Unexpected error. Please try again.')
      }
    } finally {
      setLoading(false)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [customer, items, onSuccess])

  return (
    <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal modal-lg">
        <div className="modal-header">
          <h2>Create Sales Order</h2>
          <button className="btn-icon" onClick={onClose}>✕</button>
        </div>

        <div className="modal-body">
          <div className="field">
            <label>Customer *</label>
            <input
              type="text"
              placeholder="Customer name..."
              value={customer}
              onChange={(e) => { setCustomer(e.target.value); setErrors({}) }}
            />
            {errors.customer && <span className="field-error">{errors.customer}</span>}
          </div>

          <div className="field">
            <label>Items</label>
            <div className="order-items-list">
              {items.map((item, i) => (
                <div key={i} className="order-item-row">
                  <div className="field" style={{ flex: 2 }}>
                    <label style={{ fontSize: 11 }}>SKU</label>
                    <input
                      type="text"
                      placeholder="e.g. SKU-001"
                      value={item.sku}
                      onChange={(e) => updateItem(i, 'sku', e.target.value)}
                      style={stockError?.sku === item.sku ? { borderColor: 'var(--danger)' } : {}}
                    />
                    {errors[`sku_${i}`] && <span className="field-error">{errors[`sku_${i}`]}</span>}
                  </div>
                  <div className="field" style={{ flex: 1 }}>
                    <label style={{ fontSize: 11 }}>Qty</label>
                    <input
                      type="number"
                      min={1}
                      placeholder="1"
                      value={item.qty}
                      onChange={(e) => updateItem(i, 'qty', e.target.value)}
                    />
                    {errors[`qty_${i}`] && <span className="field-error">{errors[`qty_${i}`]}</span>}
                  </div>
                  {items.length > 1 && (
                    <button className="remove-item-btn" onClick={() => removeItem(i)}>✕</button>
                  )}
                </div>
              ))}
            </div>
            <button
              type="button"
              className="btn btn-ghost btn-sm"
              style={{ marginTop: 8, alignSelf: 'flex-start' }}
              onClick={addItem}
            >
              + Add Item
            </button>
          </div>

          {/* Insufficient stock error — highlights exact SKU */}
          {stockError && (
            <div className="alert alert-error">
              <span>⚠</span>
              <div>
                <strong>Insufficient Stock</strong>: SKU <code style={{ background: 'rgba(0,0,0,0.2)', padding: '1px 4px', borderRadius: 3 }}>{stockError.sku}</code> only has{' '}
                <strong>{formatRupiah(stockError.available)} units</strong> available.
              </div>
            </div>
          )}

          {globalError && (
            <div className="alert alert-error">
              <span>⚠</span>
              <span>{globalError}</span>
            </div>
          )}
        </div>

        <div className="modal-footer">
          <button className="btn btn-ghost" onClick={onClose} disabled={loading}>Cancel</button>
          <button className="btn btn-primary" onClick={handleSubmit} disabled={loading}>
            {loading ? 'Creating...' : 'Create Order'}
          </button>
        </div>
      </div>
    </div>
  )
}
