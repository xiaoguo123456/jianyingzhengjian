<template>
  <view class="page px">
    <view v-if="task && task.status !== 'failed'" class="gen">
      <view class="gen__anim"><view class="gen__ring" /><y-icon name="sparkles" :size="64" /></view>
      <view class="gen__title">正在制作你的{{ MODULE_NAME[task.module] }}…</view>
      <view class="gen__cap">{{ task.target_name }}</view>
      <view class="gen__stages card"><y-stage-indicator :stage="task.stage" :status="task.status" /></view>
      <view class="gen__hint">{{ task.uses_genmodel ? '通常需要 20–60 秒，可离开页面，完成后在「我的作品」查看' : '证件照处理通常只需几秒' }}</view>
      <view v-if="polling.timedOut.value" class="gen__hint gen__hint--warn">等待时间较长，稍后可在「生成记录」中查看结果</view>
    </view>

    <view v-else-if="task" class="fail">
      <view class="fail__icon"><y-icon name="alert" :size="64" color="#EF4444" /></view>
      <view class="fail__title">{{ task.error?.message || '本次生成失败' }}</view>
      <view v-if="task.refunded" class="fail__cap">生成次数已返还</view>
      <view class="fail__actions">
        <y-primary-button text="重新尝试" :loading="gen.submitting.value" @press="retry" />
        <y-primary-button text="返回" secondary @press="back(1)" />
      </view>
    </view>

    <y-skeleton v-else type="list" :count="2" />
    <y-ad-sheet v-model:visible="gen.adVisible.value" :busy="gen.adBusy.value" @watch="gen.watchAd" @cancel="gen.cancelAd" />
  </view>
</template>

<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { api } from '@/api'
import { track } from '@/composables/useAnalytics'
import { useGenerate } from '@/composables/useGenerate'
import { usePolling } from '@/composables/usePolling'
import { back, replace } from '@/platform'
import { useFlowStore } from '@/store/flow'
import { useTasksStore } from '@/store/tasks'
import { useUserStore } from '@/store/user'
import { MODULE_NAME, type Task } from '@/types'
import { uuid } from '@/utils/uuid'

const tasks = useTasksStore()
const flow = useFlowStore()
const user = useUserStore()
const gen = useGenerate()
const task = ref<Task | null>(null)
let taskId = ''

const polling = usePolling(async () => {
  const t = await api.task(taskId)
  task.value = t
  tasks.set(t)
  if (t.status === 'success' && t.work) {
    track('task_success', { task_id: t.id })
    replace(`/pages/result/index?work_id=${t.work.id}&task_id=${t.id}`)
    return true
  }
  if (t.status === 'failed') {
    track('task_failed', { task_id: t.id, code: t.error?.code })
    user.refreshCredits().catch(() => {})
    return true
  }
  return false
})

onLoad(async (q) => {
  taskId = q?.task_id || ''
  await user.ready()
  task.value = tasks.get(taskId) || null
  polling.start()
})

async function retry() {
  if (!task.value) return
  const t = await gen.submit(() => api.regenerate(task.value!.id, uuid()))
  if (t) { flow.newAttempt(task.value.id); taskId = t.id; task.value = t; polling.start() }
}
</script>

<style lang="scss" scoped>
.gen { text-align: center; padding-top: 80rpx; }
.gen__anim { position: relative; width: 200rpx; height: 200rpx; margin: 0 auto 32rpx; display: flex; align-items: center; justify-content: center; }
.gen__ring { position: absolute; top: 0; right: 0; bottom: 0; left: 0; border-radius: 50%; border: 6rpx solid $color-primary-soft; border-top-color: $color-primary; animation: spin 1.2s linear infinite; }
.gen__title { font-size: $font-h2; font-weight: 600; }
.gen__cap { font-size: $font-body; color: $color-text-2; margin-top: 8rpx; }
.gen__stages { margin: 40rpx 0 24rpx; padding: 32rpx; text-align: left; }
.gen__hint { font-size: $font-caption; color: $color-text-3; line-height: 1.6; }
.gen__hint--warn { color: $color-warning; margin-top: 12rpx; }
.fail { text-align: center; padding-top: 100rpx; }
.fail__icon { margin-bottom: 20rpx; display: flex; justify-content: center; }
.fail__title { font-size: $font-body-strong; font-weight: 600; padding: 0 40rpx; }
.fail__cap { font-size: $font-caption; color: $color-success; margin-top: 8rpx; }
.fail__actions { margin-top: 48rpx; display: flex; flex-direction: column; gap: 20rpx; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
