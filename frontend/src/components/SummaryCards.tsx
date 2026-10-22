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
    { label: 'Total Products', value: total, color: 'var(--accent)', tag: 'All' },
    { label: 'Active Products', value: active, color: 'var(--success)', tag: 'Active' },
    { label: 'Low Stock Items', value: lowStock, color: 'var(--warning)', tag: '<= 10' },
    { label: 'Total Inventory Value', value: formatRupiah(totalValue), color: 'var(--info)', tag: 'IDR' },
  ]

  return (
    <div className="summary-grid">
      {cards.map(card => (
        <div className="summary-card" key={card.label}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
            <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: 0.5 }}>
              {card.label}
            </span>
            <span className="sku-tag" style={{ fontSize: 10, borderColor: card.color, color: card.color }}>
              {card.tag}
            </span>
          </div>
          <div className="summary-value" style={{ color: card.color, fontSize: 22, fontWeight: 700 }}>
            {card.value}
          </div>
        </div>
      ))}
    </div>
  )
}
