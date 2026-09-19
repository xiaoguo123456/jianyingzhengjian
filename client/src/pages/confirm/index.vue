<template>
  <view class="page px">
    <view class="sum card">
      <image v-if="flow.photo" class="sum__img" :src="flow.photo.preview_url" mode="aspectFill" />
      <view class="sum__body">
        <view class="sum__name">{{ flow.targetName }}</view>
        <view class="sum__cap">{{ MODULE_NAME[flow.module || 'idphoto'] }}</view>
        <template v-if="flow.kind === 'idphoto'">
          <view class="sum__cap">背景 {{ bgName(flow.params.bg) }} · 服装 {{ clothingName }} · 美化 {{ flow.params.beauty === 'light' ? '轻度' : '自然' }}</view>
        </template>
      </view>
    </view>

    <view class="cost card">
      <view class="cost__row">
        <text class="cost__label">本次消耗</text>
        <text class="cost__value">{{ flow.creditCost }} 次生成机会</text>
      </view>
      <view class="cost__row">
        <text class="cost__label">可用次数</text>
        <text class="cost__value num">{{ user.total }}</text>
      </view>
      <view v-if="user.total < flow.creditCost" class="cost__hint">
        次数不足，点击开始后{{ user.adsEnabled ? '观看一段视频即可获得 1 次' : '请明天再试' }}
      </view>
    </view>

    <view class="notify card">
      <view class="notify__text">
        <view class="notify__title">生成完成后通知我</view>
        <view class="notify__cap">可离开页面，完成后通过服务通知提醒</view>
      </view>
      <switch :checked="notify" color="#2F7BF6" @change="(e: any) => (notify = e.detail.value)" />
    </view>

    <view class="bottom-space" />
    <y-primary-button sticky text="开始生成" icon="sparkles" :loading="gen.submitting.value" @press="start" />
    <y-ad-sheet v-model:visible="gen.adVisible.value" :busy="gen.adBusy.value" @watch="gen.watchAd" @cancel="gen.cancelAd" />
  </view>
</template>

<script setup lang="ts">
import { onLoad, onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import { api } from '@/api'
import { track } from '@/composables/useAnalytics'
import { useGenerate } from '@/composables/useGenerate'
import { ads, back, replace, subscribe } from '@/platform'
import { useFlowStore } from '@/store/flow'
import { useUserStore } from '@/store/user'
import { MODULE_NAME, type ClothingOption } from '@/types'
import { bgName } from '@/utils/format'

const flow = useFlowStore()
const user = useUserStore()
const gen = useGenerate()
const notify = ref(true)
const clothing = ref<ClothingOption[]>([])

onLoad(async () => {
  if (!flow.photo || !flow.hasTarget) { back(); return }
  track('confirm_view', { module: flow.module })
  if (flow.kind === 'idphoto') clothing.value = await api.clothingOptions()
  if (user.adsEnabled && user.total === 0) ads.preload()
})
onShow(() => { user.refreshCredits().catch(() => {}) })
const clothingName = computed(() => clothing.value.find((c) => c.id === flow.params.clothing)?.name || '保持原服装')

async function start() {
  track('generate_click', { module: flow.module })
  let notifyOk = false
  if (notify.value) notifyOk = await subscribe.request(user.config?.subscribe_template_ids.task_finished || '')
  flow.setNotify(notifyOk)
  const task = await gen.submit(() => api.createTask({
    kind: flow.kind!, spec_id: flow.specId || undefined, template_id: flow.templateId || undefined,
    photo_id: flow.photo!.id, params: flow.kind === 'idphoto' ? { ...flow.params } : {},
    notify: notifyOk, parent_task_id: flow.parentTaskId, idempotency_key: flow.idempotencyKey,
  }))
  if (task) replace(`/pages/generating/index?task_id=${task.id}`)
}
</script>

<style lang="scss" scoped>
.sum { display: flex; align-items: center; padding: 20rpx; margin-top: 16rpx; }
.sum__img { width: 140rpx; height: 186rpx; border-radius: $radius-sm; background: $color-bg; flex-shrink: 0; }
.sum__body { padding-left: 24rpx; min-width: 0; }
.sum__name { font-size: $font-h2; font-weight: 600; }
.sum__cap { font-size: $font-caption; color: $color-text-3; margin-top: 6rpx; }
.cost { margin-top: 20rpx; padding: 8rpx 24rpx; }
.cost__row { display: flex; justify-content: space-between; padding: 20rpx 0; border-bottom: 1px solid $color-border; }
.cost__row:last-of-type { border-bottom: 0; }
.cost__label { font-size: $font-body; color: $color-text-2; }
.cost__value { font-size: $font-body-strong; font-weight: 600; color: $color-primary; }
.cost__hint { font-size: $font-caption; color: $color-warning; padding: 0 0 20rpx; }
.notify { margin-top: 20rpx; padding: 20rpx 24rpx; display: flex; align-items: center; justify-content: space-between; }
.notify__title { font-size: $font-body; font-weight: 600; }
.notify__cap { font-size: 22rpx; color: $color-text-3; margin-top: 2rpx; }
</style>
