<template>
  <view class="pc">
    <view class="pc__preview-wrap"><image class="pc__preview" :src="preview" mode="aspectFit" /></view>
    <view class="pc__status" :class="passed ? 'pc__status--ok' : 'pc__status--bad'">
      <y-icon :name="passed ? 'check' : 'alert'" :size="32" :color="passed ? '#22C55E' : '#EF4444'" :stroke-width="2.2" />
      <text>{{ passed ? '照片检测通过' : '这张照片可能影响生成效果' }}</text>
    </view>
    <view v-if="!passed" class="pc__reasons">
      <view v-for="r in reasons" :key="r" class="pc__reason">· {{ reasonText(r) }}</view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { PHOTO_REASON_COPY } from '@/utils/errors'
defineProps<{ preview: string; passed: boolean; reasons?: string[] }>()
const reasonText = (r: string) => PHOTO_REASON_COPY[r] || r
</script>

<style lang="scss" scoped>
.pc { text-align: center; }
.pc__preview-wrap { width: 100%; height: 640rpx; border-radius: $radius-md; overflow: hidden; background: #fff; }
.pc__preview { width: 100%; height: 100%; }
.pc__status { display: inline-flex; align-items: center; gap: 8rpx; margin-top: 24rpx; font-size: $font-body-strong; font-weight: 600; }
.pc__status--ok { color: $color-success; }
.pc__status--bad { color: $color-danger; }
.pc__reasons { margin-top: 12rpx; font-size: $font-caption; color: $color-text-2; }
.pc__reason { line-height: 1.8; }
</style>
