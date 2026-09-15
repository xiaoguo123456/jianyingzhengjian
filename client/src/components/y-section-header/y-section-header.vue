<template>
  <view class="sh">
    <view class="sh__title">{{ title }}</view>
    <view v-if="more" class="sh__more" hover-class="sh__more--hover" @tap="$emit('more')">
      <text class="sh__more-text">{{ more }}</text>
      <y-icon name="chevron-right" :size="28" color="#8A94A6" :stroke-width="2" />
    </view>
  </view>
</template>

<script setup lang="ts">
defineProps<{ title: string; more?: string }>()
defineEmits<{ (e: 'more'): void }>()
</script>

<style lang="scss" scoped>
.sh { display: flex; align-items: center; justify-content: space-between; margin-bottom: $space-3; }
.sh__title { font-size: 36rpx; font-weight: 700; color: $color-text; letter-spacing: -0.3rpx; }
.sh__more { display: flex; align-items: center; gap: 2rpx; padding: 8rpx 0 8rpx 16rpx; }
.sh__more-text { font-size: $font-caption; color: $color-text-3; line-height: 1; }
.sh__more--hover { opacity: 0.6; }
</style>
