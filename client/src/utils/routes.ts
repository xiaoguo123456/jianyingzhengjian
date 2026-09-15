import type { Module } from '@/types'

/** 头像 lives inside the 写真 tab as a segment (docs/DECISIONS.md D-24). */
export type PortraitSegment = 'portrait' | 'avatar'

/** Tab page that hosts a module. */
export function moduleTabPath(m: Module): string {
  return m === 'avatar' ? '/pages/portrait/index' : `/pages/${m}/index`
}

/** Link with query for shares and deep links. switchTab cannot carry a query; tab pages read it in onLoad on cold start. */
export function moduleLinkPath(m: Module, query: Record<string, string> = {}): string {
  const q: Record<string, string> = { ...(m === 'avatar' ? { seg: 'avatar' } : {}), ...query }
  const s = Object.keys(q).map((k) => `${k}=${encodeURIComponent(q[k])}`).join('&')
  return moduleTabPath(m) + (s ? `?${s}` : '')
}
