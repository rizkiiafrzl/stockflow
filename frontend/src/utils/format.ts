import { formatDistanceToNow } from 'date-fns'

/** Format IDR price: 8500000 → "Rp 8.500.000" */
export function formatRupiah(amount: number): string {
  return 'Rp ' + amount.toLocaleString('id-ID')
}

/** Format relative time: "2h ago", "just now" */
export function formatRelative(isoString: string | null): string {
  if (!isoString) return '—'
  try {
    return formatDistanceToNow(new Date(isoString), { addSuffix: true })
  } catch {
    return '—'
  }
}
