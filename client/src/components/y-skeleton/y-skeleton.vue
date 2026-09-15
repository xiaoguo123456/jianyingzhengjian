<template>
  <view class="sk">
    <template v-if="type === 'home'">
      <view class="sk__block sk__banner" />
      <view class="sk__row"><view v-for="i in 4" :key="i" class="sk__block sk__card" /></view>
      <view class="sk__row"><view v-for="i in 4" :key="i" class="sk__block sk__card sk__card--short" /></view>
      <view class="sk__row sk__row--2"><view v-for="i in 2" :key="i" class="sk__block sk__tall" /></view>
    </template>
    <template v-else-if="type === 'grid'">
      <view class="sk__row sk__row--2"><view v-for="i in count" :key="i" class="sk__block sk__tall" /></view>
    </template>
    <template v-else>
      <view v-for="i in count" :key="i" class="sk__block sk__line" />
    </template>
  </view>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ type?: 'home' | 'grid' | 'list'; count?: number }>(), { type: 'list', count: 4 })
</script>

<style lang="scss" scoped>
.sk__block { background: linear-gradient(90deg, #E9EEF5 25%, #F3F6FA 37%, #E9EEF5 63%); background-size: 400% 100%; animation: shimmer 1.4s infinite; border-radius: $radius-md; }
.sk__banner { height: 260rpx; margin-bottom: 48rpx; }
.sk__row { display: flex; gap: 20rpx; margin-bottom: 48rpx; }
.sk__card { flex: 1; height: 220rpx; }
.sk__card--short { height: 150rpx; }
.sk__row--2 .sk__tall { flex: 1; height: 440rpx; margin-bottom: 20rpx; }
.sk__line { height: 140rpx; margin-bottom: 20rpx; }
@keyframes shimmer { 0% { background-position: 100% 0; } 100% { background-position: 0 0; } }
</style>
