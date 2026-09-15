<template>
  <view class="page px">
    <y-skeleton v-if="loading && !items.length" type="list" />
    <y-empty-state v-else-if="!items.length" icon="file-text" text="还没有生成记录" />
    <view v-else class="list">
      <view v-for="t in items" :key="t.id" class="rec card" hover-class="rec--hover" @tap="open(t)">
        <view class="rec__thumb-wrap">
          <image v-if="t.work" class="rec__thumb" :src="t.work.thumb_url" mode="aspectFill" />
          <view v-else class="rec__thumb rec__thumb--empty"><y-icon :name="t.status === 'failed' ? 'alert' : 'hourglass'" :size="40" :color="t.status === 'failed' ? '#EF4444' : '#2F7BF6'" /></view>
        </view>
        <view class="rec__body">
          <view class="rec__title">{{ t.target_name || MODULE_NAME[t.module] }}</view>
          <view class="rec__cap">{{ MODULE_NAME[t.module] }} · {{ dateShort(t.created_at) }}</view>
          <view class="rec__cap">{{ t.uses_genmodel ? `消耗 ${t.credits_consumed} 次` : '免费' }}<text v-if="t.refunded"> · 已返还</text></view>
        </view>
        <view class="status" :class="`status--${t.status}`">{{ statusText(t.status) }}</view>
      </view>
    </view>
    <view class="bottom-space" />
  </view>
</template>

<script setup lang="ts">
import { onLoad, onReachBottom } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { api } from '@/api'
import { go } from '@/platform'
import { useUserStore } from '@/store/user'
import { MODULE_NAME, type Task, type TaskStatus } from '@/types'
import { dateShort } from '@/utils/format'

const items = ref<Task[]>([])
const page = ref(1)
const hasMore = ref(false)
const loading = ref(true)
onLoad(async () => { await useUserStore().ready(); await load(true) })
async function load(reset = false) {
  loading.value = true
  if (reset) page.value = 1
  try { const r = await api.tasks({ page: page.value }); items.value = reset ? r.items : [...items.value, ...r.items]; hasMore.value = r.has_more } finally { loading.value = false }
}
onReachBottom(() => { if (hasMore.value && !loading.value) { page.value++; load() } })
const statusText = (s: TaskStatus) => ({ waiting: '排队中', processing: '生成中', success: '成功', failed: '失败' }[s])
function open(t: Task) {
  if (t.status === 'success' && t.work) go(`/pages-mine/work-detail/index?id=${t.work.id}`)
  else if (t.status === 'waiting' || t.status === 'processing') go(`/pages/generating/index?task_id=${t.id}`)
}
</script>

<style lang="scss" scoped>
.list { display: flex; flex-direction: column; gap: 16rpx; padding-top: 16rpx; }
.rec { display: flex; align-items: center; padding: 20rpx; }
.rec--hover { opacity: 0.85; }
.rec__thumb-wrap { width: 100rpx; height: 132rpx; border-radius: $radius-sm; overflow: hidden; background: $color-primary-soft; flex-shrink: 0; }
.rec__thumb { width: 100%; height: 100%; }
.rec__thumb--empty { display: flex; align-items: center; justify-content: center; }
.rec__body { flex: 1; min-width: 0; padding: 0 20rpx; }
.rec__title { font-size: $font-body-strong; font-weight: 600; }
.rec__cap { font-size: 22rpx; color: $color-text-3; margin-top: 2rpx; }
.status { font-size: 22rpx; padding: 6rpx 14rpx; border-radius: 8rpx; font-weight: 600; }
.status--success { background: #ECFDF3; color: $color-success; }
.status--failed { background: #FEF2F2; color: $color-danger; }
.status--processing, .status--waiting { background: $color-primary-soft; color: $color-primary; }
</style>
