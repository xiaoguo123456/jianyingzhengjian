import { api } from '@/api'
import type { AppEvent } from '@/types'

let queue: AppEvent[] = []
let timer: ReturnType<typeof setTimeout> | null = null

function flush() {
  timer = null
  if (!queue.length) return
  const batch = queue
  queue = []
  api.events(batch).catch(() => {})
}

/** Records a funnel event twice: WeChat analytics (where available) and the batched API (docs/DECISIONS.md D-17). */
export function track(name: string, props: Record<string, any> = {}) {
  // #ifdef MP-WEIXIN
  try { (wx as any).reportEvent?.(name, props) } catch { /* ignore */ }
  // #endif
  queue.push({ name, props, ts: new Date().toISOString() })
  if (!timer) timer = setTimeout(flush, 3000)
  if (queue.length >= 20) flush()
}
