<template>
  <view class="page px">
    <view class="note">上传的原始照片会在 {{ retention }} 天后自动删除；作品不受影响。</view>
    <y-skeleton v-if="loading" type="list" />
    <y-empty-state v-else-if="!items.length" icon="album" text="还没有上传过照片" />
    <view v-else class="list">
      <view v-for="p in items" :key="p.id" class="ph card">
        <image class="ph__img" :src="p.preview_url" mode="aspectFill" @tap="previewPhoto(p)" />
        <view class="ph__body">
          <view class="ph__title">{{ dateShort(p.created_at) }} 上传</view>
          <view class="ph__cap">{{ p.width }}×{{ p.height }} px · {{ daysLeft(p.expires_at) }} 天后删除</view>
          <view class="ph__actions">
            <view class="ph__btn" @tap="reuse(p)">再次使用</view>
            <view class="ph__btn ph__btn--danger" @tap="remove(p)">删除</view>
          </view>
        </view>
      </view>
    </view>
    <view class="bottom-space" />
  </view>
</template>

<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import { api } from '@/api'
import { go, toast } from '@/platform'
import { useFlowStore } from '@/store/flow'
import { useUserStore } from '@/store/user'
import type { Photo } from '@/types'
import { dateShort } from '@/utils/format'

const user = useUserStore()
const flow = useFlowStore()
const items = ref<Photo[]>([])
const loading = ref(true)
const retention = computed(() => user.config?.retention_days ?? 30)
onShow(async () => { await user.ready(); try { items.value = (await api.photos()).items } finally { loading.value = false } })
const previewPhoto = (p: Photo) => uni.previewImage({ urls: [p.preview_url] })
const daysLeft = (iso: string) => Math.max(0, Math.ceil((new Date(iso).getTime() - Date.now()) / 864e5))
function reuse(p: Photo) {
  uni.showActionSheet({ itemList: ['证件照', '职业照', '写真', '头像'], success: (r) => {
    const m = (['idphoto', 'pro', 'portrait', 'avatar'] as const)[r.tapIndex]
    flow.start(m); flow.setPhoto(p)
    go(`/pages/upload/index?module=${m}&reuse=1`)
  } })
}
function remove(p: Photo) {
  uni.showModal({ title: '删除照片', content: '删除原始照片不影响已生成的作品。', confirmColor: '#EF4444', success: async (r) => {
    if (!r.confirm) return
    await api.deletePhoto(p.id); items.value = items.value.filter((x) => x.id !== p.id); toast('已删除')
  } })
}
</script>

<style lang="scss" scoped>
.note { font-size: 22rpx; color: $color-text-3; padding: 16rpx 0; }
.list { display: flex; flex-direction: column; gap: 16rpx; }
.ph { display: flex; padding: 20rpx; }
.ph__img { width: 140rpx; height: 186rpx; border-radius: $radius-sm; background: $color-primary-soft; flex-shrink: 0; }
.ph__body { flex: 1; padding-left: 20rpx; }
.ph__title { font-size: $font-body; font-weight: 600; }
.ph__cap { font-size: 22rpx; color: $color-text-3; margin-top: 4rpx; }
.ph__actions { display: flex; gap: 16rpx; margin-top: 20rpx; }
.ph__btn { padding: 0 24rpx; height: 56rpx; line-height: 56rpx; border-radius: $radius-pill; background: $color-primary-soft; color: $color-primary; font-size: $font-caption; font-weight: 600; }
.ph__btn--danger { background: #FEF2F2; color: $color-danger; }
</style>
