export function uuid(): string {
  let s = ''
  for (let i = 0; i < 32; i++) {
    const r = (Math.random() * 16) | 0
    if (i === 8 || i === 12 || i === 16 || i === 20) s += '-'
    s += (i === 12 ? 4 : i === 16 ? (r & 3) | 8 : r).toString(16)
  }
  return s
}
export function shortId(prefix = ''): string {
  return prefix + Date.now().toString(36) + Math.random().toString(36).slice(2, 6)
}
export const sleep = (ms: number) => new Promise<void>((r) => setTimeout(r, ms))
