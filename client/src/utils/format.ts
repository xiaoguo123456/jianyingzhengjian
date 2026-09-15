export function mm(w: number, h: number): string {
  return `${trim(w)}×${trim(h)} mm`
}
export function px(w: number, h: number): string {
  return `${w}×${h} px`
}
function trim(n: number): string {
  return Number.isInteger(n) ? String(n) : n.toFixed(1)
}

export function relativeTime(iso: string): string {
  const d = new Date(iso).getTime()
  const diff = Date.now() - d
  const m = Math.floor(diff / 6e4)
  if (m < 1) return '刚刚'
  if (m < 60) return `${m} 分钟前`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h} 小时前`
  const day = Math.floor(h / 24)
  if (day < 7) return `${day} 天前`
  return dateShort(iso)
}

export function dateShort(iso: string): string {
  const d = new Date(iso)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

export const BG_NAMES: Record<string, string> = {
  '#FFFFFF': '白色',
  '#438EDB': '蓝色',
  '#FF0000': '红色',
  '#808080': '灰色',
}
export function bgName(hex: string): string {
  return BG_NAMES[hex.toUpperCase()] || '自定义'
}
