<template>
  <view class="sk" :class="`sk--${variant}`">
    <template v-if="type === 'home'">
      <view class="sk__block sk__banner" />
      <view class="sk__row sk__chips"><view v-for="i in 4" :key="i" class="sk__block sk__chip" /></view>
      <template v-if="variant === 'idphoto'">
        <view v-for="i in 3" :key="i" class="sk__block sk__line" />
      </template>
      <view v-else class="sk__grid"><view v-for="i in 2" :key="i" class="sk__block sk__tall" /></view>
    </template>
    <view v-else-if="type === 'grid' || type === 'works'" class="sk__grid" :class="{ 'sk__grid--works': type === 'works' }">
      <view v-for="i in count" :key="i" class="sk__block sk__tall" />
    </view>
    <template v-else><view v-for="i in count" :key="i" class="sk__block sk__line" /></template>
  </view>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ type?: 'home' | 'grid' | 'list' | 'works'; variant?: 'idphoto' | 'pro' | 'portrait' | 'avatar'; count?: number }>(), { type: 'list', variant: 'idphoto', count: 4 })
</script>

<style lang="scss" scoped>
.sk__block { background: linear-gradient(90deg, #E9ECF1 25%, #F3F5F8 37%, #E9ECF1 63%); background-size: 400% 100%; animation: shimmer 1.4s infinite; border-radius: $radius-md; }
.sk__banner { height: 282rpx; margin-bottom: 40rpx; }
.sk--pro .sk__banner { height: 396rpx; }
.sk--portrait .sk__banner { height: 456rpx; }
.sk__row { display: flex; gap: 12rpx; margin-bottom: 32rpx; }
.sk__chip { flex: 1; height: 76rpx; }
.sk--idphoto .sk__chip { height: 208rpx; }
.sk__grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 28rpx 20rpx; }
.sk__grid--works { grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16rpx; }
.sk__tall { padding-top: 133.33%; margin-bottom: 40rpx; }
.sk__line { height: 104rpx; margin-bottom: 16rpx; }
@keyframes shimmer { 0% { background-position: 100% 0; } 100% { background-position: 0 0; } }
@media (prefers-reduced-motion: reduce) { .sk__block { animation: none; } }
</style>
