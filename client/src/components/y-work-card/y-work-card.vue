<template>
  <view class="wc card" hover-class="wc--hover" @tap="$emit('select', work)">
    <view class="wc__img-wrap" :class="{ 'wc__img-wrap--square': work.module === 'avatar' }">
      <image class="wc__img" :src="work.thumb_url" mode="aspectFill" lazy-load />
      <y-ai-label v-if="work.ai_label" />
    </view>
    <view class="wc__meta">
      <view class="wc__title ellipsis">{{ work.template?.name || work.spec?.name || MODULE_NAME[work.module] }}</view>
      <view class="wc__caption">{{ relativeTime(work.created_at) }}</view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { MODULE_NAME, type Work } from '@/types'
import { relativeTime } from '@/utils/format'
defineProps<{ work: Work }>()
defineEmits<{ (e: 'select', w: Work): void }>()
</script>

<style lang="scss" scoped>
.wc { overflow: hidden; }
.wc--hover { opacity: 0.85; }
.wc__img-wrap { position: relative; width: 100%; padding-top: 133.33%; background: $color-primary-soft; }
.wc__img-wrap--square { padding-top: 100%; }
.wc__img { position: absolute; top: 0; right: 0; bottom: 0; left: 0; width: 100%; height: 100%; }
.wc__meta { padding: 14rpx 20rpx 18rpx; }
.wc__title { font-size: $font-body; font-weight: 600; }
.wc__caption { font-size: 22rpx; color: $color-text-3; margin-top: 2rpx; }
</style>
