<template>
  <view class="er">
    <view class="er__icon"><y-icon name="alert" :size="64" color="#F59E0B" /></view>
    <view class="er__text">{{ text }}</view>
    <view class="er__btn" hover-class="er__btn--hover" @tap="$emit('retry')">重试</view>
  </view>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ text?: string }>(), { text: '加载失败' })
defineEmits<{ (e: 'retry'): void }>()
</script>

<style lang="scss" scoped>
.er { text-align: center; padding: 120rpx 32rpx; }
.er__icon { margin-bottom: 16rpx; display: flex; justify-content: center; }
.er__text { font-size: $font-body; color: $color-text-2; }
.er__btn { display: inline-block; margin-top: 24rpx; padding: 0 40rpx; height: 72rpx; line-height: 72rpx; border-radius: $radius-pill; background: $color-primary; color: #fff; font-size: $font-body; font-weight: 600; }
.er__btn--hover { opacity: 0.8; }
</style>
