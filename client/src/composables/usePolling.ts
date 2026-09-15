import { onHide, onShow, onUnload } from '@dcloudio/uni-app'
import { ref } from 'vue'

export interface PollingOptions { interval?: number; slowAfter?: number; slowInterval?: number; maxDuration?: number }

/** Calls `tick` until it returns true. 2 s, then 5 s after 30 s, stops after 5 min (docs/FRONTEND_ARCHITECTURE.md §8). */
export function usePolling(tick: () => Promise<boolean>, opts: PollingOptions = {}) {
  const interval = opts.interval ?? 2000
  const slowAfter = opts.slowAfter ?? 30_000
  const slowInterval = opts.slowInterval ?? 5000
  const maxDuration = opts.maxDuration ?? 300_000
  const active = ref(false)
  const timedOut = ref(false)
  let timer: ReturnType<typeof setTimeout> | null = null
  let startedAt = 0

  const schedule = () => {
    const elapsed = Date.now() - startedAt
    if (elapsed > maxDuration) { timedOut.value = true; stop(); return }
    timer = setTimeout(run, elapsed > slowAfter ? slowInterval : interval)
  }
  const run = async () => {
    if (!active.value) return
    let done = false
    try { done = await tick() } catch { done = false }
    if (!active.value) return
    if (done) stop(); else schedule()
  }
  const start = () => {
    if (active.value) return
    active.value = true
    timedOut.value = false
    startedAt = Date.now()
    run()
  }
  const stop = () => {
    active.value = false
    if (timer) { clearTimeout(timer); timer = null }
  }

  onHide(stop)
  onShow(() => { if (startedAt && !timedOut.value) start() })
  onUnload(stop)
  return { start, stop, active, timedOut }
}
