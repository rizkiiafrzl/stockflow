import { useState, useCallback, useEffect, useRef } from 'react'
import { createOrder, getProducts } from '../api/client'
import { APIError } from '../types'
import type { SalesOrder, Product } from '../types'
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
  const [catalog, setCatalog] = useState<Product[]>([])
  const [activeDropdown, setActiveDropdown] = useState<number | null>(null)
  const [loading, setLoading] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [stockError, setStockError] = useState<{ sku: string; available: number } | null>(null)
  const [globalError, setGlobalError] = useState<string | null>(null)
  const dropdownRef = useRef<HTMLDivElement | null>(null)

  // Fetch product catalog for instant auto-suggestions
  useEffect(() => {
    getProducts({ limit: 100 })
      .then(res => setCatalog(res.data))
      .catch(() => {})
  }, [])

  // Close autocomplete dropdown on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setActiveDropdown(null)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const addItem = () => setItems(prev => [...prev, { sku: '', qty: '' }])
  const removeItem = (i: number) => {
    setItems(prev => prev.filter((_, idx) => idx !== i))
    if (activeDropdown === i) setActiveDropdown(null)
  }
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
          <button className="btn-icon" onClick={onClose}>X</button>
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
              {items.map((item, i) => {
                const searchLower = item.sku.trim().toLowerCase()
                const suggestions = catalog.filter(p =>
                  p.is_active && (
                    p.sku.toLowerCase().includes(searchLower) ||
                    p.name.toLowerCase().includes(searchLower)
                  )
                ).slice(0, 8)
                const matched = catalog.find(p => p.sku.toLowerCase() === item.sku.trim().toLowerCase())
                const qtyNum = parseInt(item.qty, 10)
                const isOverStock = matched && !isNaN(qtyNum) && qtyNum > matched.stock

                return (
                  <div key={i} className="order-item-row" style={{ flexDirection: 'column' }}>
                    <div style={{ display: 'flex', gap: 10, width: '100%', alignItems: 'flex-start' }}>
                      {/* SKU Input with autocomplete */}
                      <div className="field" style={{ flex: 2 }}>
                        <label style={{ fontSize: 11 }}>SKU / Product Name</label>
                        <div
                          className="autocomplete-wrapper"
                          ref={activeDropdown === i ? dropdownRef : undefined}
                        >
                          <input
                            type="text"
                            placeholder="Type SKU or product name..."
                            value={item.sku}
                            onChange={(e) => {
                              updateItem(i, 'sku', e.target.value)
                              setActiveDropdown(i)
                            }}
                            onFocus={() => setActiveDropdown(i)}
                            style={stockError?.sku === item.sku ? { borderColor: 'var(--danger)' } : {}}
                            autoComplete="off"
                          />
                          {activeDropdown === i && suggestions.length > 0 && (
                            <div className="autocomplete-dropdown">
                              {suggestions.map(p => (
                                <div
                                  key={p.sku}
                                  className="autocomplete-item"
                                  onMouseDown={(e) => {
                                    e.preventDefault()
                                    updateItem(i, 'sku', p.sku)
                                    setActiveDropdown(null)
                                  }}
                                >
                                  <span className="sku-tag">{p.sku}</span>
                                  <span style={{ fontWeight: 500, color: 'var(--text)' }}>{p.name}</span>
                                  <span style={{ marginLeft: 'auto', fontSize: 11, color: p.stock <= 10 ? 'var(--warning)' : 'var(--success)' }}>
                                    Stok: {p.stock} · {formatRupiah(p.price)}
                                  </span>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                        {errors[`sku_${i}`] && <span className="field-error">{errors[`sku_${i}`]}</span>}
                      </div>

                      {/* Qty Input */}
                      <div className="field" style={{ flex: 1 }}>
                        <label style={{ fontSize: 11 }}>Qty</label>
                        <input
                          type="number"
                          min={1}
                          max={matched ? matched.stock : undefined}
                          placeholder={matched ? `1 - ${matched.stock}` : "1"}
                          value={item.qty}
                          onChange={(e) => updateItem(i, 'qty', e.target.value)}
                        />
                        {errors[`qty_${i}`] && <span className="field-error">{errors[`qty_${i}`]}</span>}
                      </div>

                      {items.length > 1 && (
                        <button className="remove-item-btn" onClick={() => removeItem(i)}>X</button>
                      )}
                    </div>

                    {/* Matched product preview info */}
                    {matched && (
                      <div className="product-hint">
                        <strong style={{ color: 'var(--text)' }}>{matched.name}</strong>
                        <span>·</span>
                        <span style={{ color: 'var(--accent)' }}>{formatRupiah(matched.price)}</span>
                        <span>·</span>
                        <span>Tersedia: <strong style={{ color: matched.stock <= 10 ? 'var(--warning)' : 'var(--success)' }}>{matched.stock} unit</strong></span>
                        {isOverStock && (
                          <span style={{ color: 'var(--danger)', fontWeight: 600, marginLeft: 6 }}>
                            [Qty melebihi stok tersedia: {matched.stock}]
                          </span>
                        )}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
            <button
              type="button"
              className="btn btn-ghost btn-sm"
              style={{ marginTop: 8, alignSelf: 'flex-start' }}
              onClick={addItem}
            >
              Add Item
            </button>
          </div>

          {/* Insufficient stock error — highlights exact SKU */}
          {stockError && (
            <div className="alert alert-error">
              <div>
                <strong>Insufficient Stock</strong>: SKU <code style={{ background: 'rgba(0,0,0,0.2)', padding: '1px 4px', borderRadius: 3 }}>{stockError.sku}</code> only has{' '}
                <strong>{formatRupiah(stockError.available)} units</strong> available.
              </div>
            </div>
          )}

          {globalError && (
            <div className="alert alert-error">
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
