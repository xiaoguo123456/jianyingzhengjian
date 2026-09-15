<template>
  <view class="hero" hover-class="hero--hover" @tap="$emit('tap')">
    <view class="hero__text">
      <view class="hero__title">
        <view v-for="(line, i) in lines" :key="i">{{ line }}</view>
      </view>
      <view class="hero__subtitle">{{ banner.subtitle }}</view>
      <view class="hero__cta">
        <y-icon name="upload" :size="32" color="#FFFFFF" :stroke-width="2.2" />
        <text>{{ cta }}</text>
      </view>
    </view>
    <view class="hero__figure">
      <image class="hero__img" :src="banner.image_url" mode="aspectFill" lazy-load />
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Banner } from '@/types'

const props = withDefaults(defineProps<{ banner: Banner; cta?: string }>(), { cta: '立即制作' })
defineEmits<{ (e: 'tap'): void }>()
const lines = computed(() => props.banner.title.split('\n'))
</script>

<style lang="scss" scoped>
/* The hero is the single call to action on a tab: whole card tappable plus an explicit button. */
.hero {
  position: relative; display: flex; align-items: center; justify-content: space-between; overflow: hidden;
  background: linear-gradient(135deg, #FFFFFF 0%, #EEF4FF 100%); border-radius: $radius-lg; padding: 40rpx 32rpx;
  box-shadow: 0 8rpx 32rpx rgba(31, 102, 224, 0.08);
}
.hero::after { content: ''; position: absolute; right: -80rpx; top: -80rpx; width: 320rpx; height: 320rpx; border-radius: 50%; background: radial-gradient(circle, rgba(58, 141, 255, 0.16) 0%, rgba(58, 141, 255, 0) 70%); }
.hero--hover { opacity: 0.92; }
.hero__text { flex: 1; padding-right: 24rpx; position: relative; z-index: 1; }
.hero__title { font-size: $font-h1; font-weight: 700; line-height: 1.28; color: $color-text; letter-spacing: -0.5rpx; }
.hero__subtitle { margin-top: 14rpx; font-size: $font-caption; color: $color-text-2; }
.hero__cta {
  display: inline-flex; align-items: center; gap: 8rpx; margin-top: 30rpx; height: 68rpx; padding: 0 30rpx;
  border-radius: $radius-pill; background: $gradient-primary; color: #fff; font-size: 26rpx; font-weight: 600;
  box-shadow: 0 8rpx 20rpx rgba(58, 141, 255, 0.28);
}
.hero__figure { position: relative; z-index: 1; width: 208rpx; height: 276rpx; border-radius: 24rpx; overflow: hidden; background: $color-primary-soft; flex-shrink: 0; box-shadow: 0 16rpx 32rpx rgba(31, 102, 224, 0.16); }
.hero__img { width: 100%; height: 100%; }
</style>
