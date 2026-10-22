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
    { label: 'Total Products', value: total, color: 'var(--accent)' },
    { label: 'Active Products', value: active, color: 'var(--success)' },
    { label: 'Low Stock Items', value: lowStock, color: 'var(--warning)' },
    { label: 'Total Inventory Value', value: formatRupiah(totalValue), color: 'var(--info)' },
  ]

  return (
    <div className="summary-grid">
      {cards.map(card => (
        <div className="summary-card" key={card.label}>
          <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 8 }}>
            {card.label}
          </div>
          <div className="summary-value" style={{ color: card.color, fontSize: 24, fontWeight: 700 }}>
            {card.value}
          </div>
        </div>
      ))}
    </div>
  )
}
