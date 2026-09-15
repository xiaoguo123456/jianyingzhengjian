import { ref } from 'vue'
import { api } from '@/api'
import { ads, toast } from '@/platform'
import { useUserStore } from '@/store/user'
import { useTasksStore } from '@/store/tasks'
import type { Credits, Task } from '@/types'
import { ApiError, ERROR_COPY, messageOf } from '@/utils/errors'
import { track } from './useAnalytics'

/**
 * Orchestrates task creation with the rewarded-video fallback
 * (docs/FRONTEND_ARCHITECTURE.md §8). The page renders <y-ad-sheet v-model:visible="ad.visible" ...>.
 */
export function useGenerate() {
  const user = useUserStore()
  const tasks = useTasksStore()
  const submitting = ref(false)
  const adVisible = ref(false)
  const adBusy = ref(false)
  let adResolver: ((ok: boolean) => void) | null = null

  function askAd(): Promise<boolean> {
    adVisible.value = true
    track('no_credits')
    return new Promise((resolve) => { adResolver = resolve })
  }
  function closeAd(ok: boolean) {
    adVisible.value = false
    adBusy.value = false
    adResolver?.(ok)
    adResolver = null
  }

  /** Called by the ad sheet's primary button. */
  async function watchAd() {
    if (adBusy.value) return
    adBusy.value = true
    try {
      const session = await api.createAdSession()
      track('ad_show')
      const { ended } = await ads.show()
      if (!ended) {
        track('ad_abandon')
        toast(ERROR_COPY.AD_SESSION_INVALID)
        adBusy.value = false
        return
      }
      const r = await api.claimAd(session.session_id, true)
      user.setCredits(r.credits)
      track('ad_ended')
      toast('已获得 1 次生成机会', 'success')
      closeAd(true)
    } catch (e) {
      track('ad_error')
      if (e instanceof ApiError && e.code === 'AD_SESSION_INVALID') toast(ERROR_COPY.AD_SESSION_INVALID)
      else toast(e instanceof ApiError && e.code === 'AD_LOAD_FAILED' ? ERROR_COPY.AD_LOAD_FAILED : messageOf(e))
      adBusy.value = false
    }
  }
  function cancelAd() { closeAd(false) }

  /**
   * Runs `create`; on 402 opens the ad sheet and retries once the credit is granted.
   * Returns the task or null when the user backed out.
   */
  async function submit(create: () => Promise<{ task: Task; credits: Credits }>): Promise<Task | null> {
    if (submitting.value) return null
    submitting.value = true
    try {
      for (let attempt = 0; attempt < 3; attempt++) {
        try {
          const r = await create()
          user.setCredits(r.credits)
          tasks.set(r.task)
          track('task_created', { task_id: r.task.id, uses_genmodel: r.task.uses_genmodel })
          return r.task
        } catch (e) {
          if (e instanceof ApiError && e.code === 'NO_CREDITS') {
            if (e.data?.credits) user.setCredits(e.data.credits)
            const ok = await askAd()
            if (!ok) return null
            continue
          }
          toast(messageOf(e))
          return null
        }
      }
      return null
    } finally {
      submitting.value = false
    }
  }

  return { submitting, adVisible, adBusy, watchAd, cancelAd, submit }
}
