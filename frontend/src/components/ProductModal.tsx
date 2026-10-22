import { useState, useCallback } from 'react'
import type { Product } from '../types'
import { createProduct, updateProduct } from '../api/client'
import { APIError } from '../types'

import { validateProduct } from '../utils/validation'

interface Props {
  product?: Product | null
  onClose: () => void
  onSuccess: () => void
}

export default function ProductModal({ product, onClose, onSuccess }: Props) {
  const isEdit = !!product
  const [sku, setSku] = useState(product?.sku ?? '')
  const [name, setName] = useState(product?.name ?? '')
  const [price, setPrice] = useState(product ? String(product.price) : '')
  const [isActive, setIsActive] = useState(product?.is_active ?? true)
  const [loading, setLoading] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [globalError, setGlobalError] = useState<string | null>(null)

  const handleSubmit = useCallback(async () => {
    const res = validateProduct(sku, name, price, isEdit)
    if (!res.valid) { setErrors(res.errors); return }

    setLoading(true)
    setGlobalError(null)
    try {
      if (isEdit && product) {
        await updateProduct(product.id, { name, price: parseInt(price, 10), is_active: isActive })
      } else {
        await createProduct({ sku, name, price: parseInt(price, 10) })
      }
      onSuccess()
    } catch (err) {
      if (err instanceof APIError) {
        if (err.detail.code === 'SKU_ALREADY_EXISTS') {
          setErrors({ sku: `SKU "${sku}" already exists` })
        } else if (err.detail.code === 'VALIDATION_ERROR') {
          setGlobalError(err.detail.message ?? 'Validation failed')
        } else {
          setGlobalError(err.detail.message ?? err.detail.code)
        }
      } else {
        setGlobalError('Unexpected error. Please try again.')
      }
    } finally {
      setLoading(false)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sku, name, price, isActive, isEdit, product, onSuccess])

  return (
    <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal">
        <div className="modal-header">
          <h2>{isEdit ? 'Edit Product' : 'New Product'}</h2>
          <button className="btn-icon" onClick={onClose}>✕</button>
        </div>

        <div className="modal-body">
          {!isEdit && (
            <div className="field">
              <label>SKU *</label>
              <input
                type="text"
                placeholder="e.g. SKU-013"
                value={sku}
                onChange={(e) => { setSku(e.target.value); setErrors({}) }}
              />
              {errors.sku && <span className="field-error">{errors.sku}</span>}
            </div>
          )}

          <div className="field">
            <label>Name *</label>
            <input
              type="text"
              placeholder="Product name..."
              value={name}
              onChange={(e) => { setName(e.target.value); setErrors({}) }}
            />
            {errors.name && <span className="field-error">{errors.name}</span>}
          </div>

          <div className="field">
            <label>Price (IDR) *</label>
            <input
              type="number"
              min={0}
              placeholder="e.g. 250000"
              value={price}
              onChange={(e) => { setPrice(e.target.value); setErrors({}) }}
            />
            {errors.price && <span className="field-error">{errors.price}</span>}
          </div>

          {isEdit && (
            <div className="field">
              <label>Status</label>
              <div style={{ display: 'flex', gap: 8 }}>
                <button
                  type="button"
                  className={`filter-tab ${isActive ? 'active' : ''}`}
                  style={isActive ? { background: 'var(--success)' } : {}}
                  onClick={() => setIsActive(true)}
                >
                  Active
                </button>
                <button
                  type="button"
                  className={`filter-tab ${!isActive ? 'active' : ''}`}
                  style={!isActive ? { background: 'var(--text-muted)' } : {}}
                  onClick={() => setIsActive(false)}
                >
                  Inactive
                </button>
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
            {loading ? 'Saving...' : isEdit ? 'Save Changes' : 'Create Product'}
          </button>
        </div>
      </div>
    </div>
  )
}
