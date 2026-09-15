<template>
  <view class="hero" :class="`hero--${variant}`" hover-class="hero--hover" @tap="$emit('press')">
    <image v-if="variant === 'editorial'" class="hero__backdrop" :src="banner.image_url" mode="aspectFill" />
    <view v-if="variant === 'editorial'" class="hero__scrim" />
    <view class="hero__text">
      <view class="hero__title">
        <view v-for="(line, i) in lines" :key="i">{{ line }}</view>
      </view>
      <view class="hero__cta">
        <y-icon name="upload" :size="30" :color="variant === 'editorial' ? '#111827' : '#FFFFFF'" :stroke-width="1.8" />
        <text>{{ cta }}</text>
      </view>
    </view>
    <view v-if="variant !== 'editorial'" class="hero__figure">
      <image class="hero__img" :src="banner.image_url" mode="aspectFill" />
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Banner } from '@/types'

const props = withDefaults(defineProps<{ banner: Banner; cta?: string; variant?: 'compact' | 'professional' | 'editorial' }>(), {
  cta: '上传照片', variant: 'compact',
})
defineEmits<{ (e: 'press'): void }>()
const lines = computed(() => props.banner.title.split('\n'))
</script>

<style lang="scss" scoped>
.hero { position: relative; display: flex; align-items: center; justify-content: space-between; overflow: hidden; background: #EDF0F5; border-radius: $radius-lg; padding: 32rpx; box-sizing: border-box; }
.hero--hover { opacity: 0.92; }
.hero__text { flex: 1; min-width: 0; padding-right: 20rpx; position: relative; z-index: 1; }
.hero__title { font-size: 42rpx; font-weight: 600; line-height: 1.4; color: $color-text; }
.hero__cta { display: inline-flex; align-items: center; justify-content: center; gap: 10rpx; margin-top: 28rpx; min-height: 80rpx; padding: 0 26rpx; border-radius: 20rpx; background: $color-primary; color: #fff; font-size: 26rpx; font-weight: 500; white-space: nowrap; }
.hero__figure { width: 164rpx; height: 218rpx; border-radius: 12rpx; overflow: hidden; background: #DBE3EC; flex-shrink: 0; }
.hero__img { width: 100%; height: 100%; }
.hero--professional { height: 396rpx; padding: 40rpx 32rpx; background: #E9EBEF; align-items: center; }
.hero--professional .hero__text { flex: none; width: 50%; box-sizing: border-box; }
.hero--professional .hero__figure { position: absolute; right: 0; top: 0; width: 46%; height: 100%; border-radius: 0; }
.hero--professional .hero__title { font-size: 44rpx; }
.hero--editorial { height: 456rpx; padding: 32rpx; align-items: flex-end; background: #4B626D; }
.hero__backdrop, .hero__scrim { position: absolute; inset: 0; width: 100%; height: 100%; }
.hero__scrim { background: linear-gradient(90deg, rgba(14, 26, 34, 0.55), rgba(14, 26, 34, 0) 75%); }
.hero--editorial .hero__title { color: #fff; font-size: 46rpx; line-height: 1.3; }
.hero--editorial .hero__cta { background: #fff; color: $color-text; margin-top: 24rpx; min-height: 76rpx; }
</style>
