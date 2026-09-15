<template>
  <view class="tag" :class="`tag--${variant}`">{{ text }}</view>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ text: string; variant?: 'primary' | 'hot' | 'muted' }>(), { variant: 'primary' })
</script>

<style lang="scss" scoped>
.tag { display: inline-block; font-size: 20rpx; line-height: 1; padding: 8rpx 12rpx; border-radius: $radius-pill; font-weight: 600; }
.tag--primary { background: $color-primary; color: #fff; }
.tag--hot { background: rgba(255, 255, 255, 0.92); color: $color-text; }
.tag--muted { background: $color-primary-soft; color: $color-primary; }
</style>
