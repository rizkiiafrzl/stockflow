import type { Product } from '../types'
import { formatRupiah } from '../utils/format'

interface Props {
  products: Product[]
}

export default function SummaryCards({ products }: Props) {
  const total = products.length
  const active = products.filter(p => p.is_active).length
  const lowStock = products.filter(p => p.stock <= 10 && p.is_active).length
  const totalValue = products.reduce((sum, p) => sum + p.price * p.stock, 0)

  const cards = [
    { icon: '📦', label: 'Total Products', value: total, color: 'var(--accent)' },
    { icon: '✅', label: 'Active Products', value: active, color: 'var(--success)' },
    { icon: '⚠️', label: 'Low Stock', value: lowStock, color: 'var(--warning)' },
    { icon: '💰', label: 'Stock Value', value: formatRupiah(totalValue), color: 'var(--info)' },
  ]

  return (
    <div className="summary-grid">
      {cards.map(card => (
        <div className="summary-card" key={card.label}>
          <div className="summary-icon">{card.icon}</div>
          <div className="summary-value" style={{ color: card.color }}>{card.value}</div>
          <div className="summary-label">{card.label}</div>
        </div>
      ))}
    </div>
  )
}
