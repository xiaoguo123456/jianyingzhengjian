<template>
  <view v-if="visible" class="sheet">
    <view class="sheet__mask" @tap="close" />
    <view class="sheet__panel">
      <view v-if="title" class="sheet__head">
        <text class="sheet__title">{{ title }}</text>
        <view class="sheet__close" @tap="close"><y-icon name="close" :size="36" color="#697386" /></view>
      </view>
      <scroll-view scroll-y class="sheet__body"><slot /></scroll-view>
    </view>
  </view>
</template>

<script setup lang="ts">
const props = defineProps<{ visible: boolean; title?: string; closable?: boolean }>()
const emit = defineEmits<{ (e: 'update:visible', v: boolean): void; (e: 'close'): void }>()
const close = () => { if (props.closable === false) return; emit('update:visible', false); emit('close') }
</script>

<style lang="scss" scoped>
.sheet { position: fixed; top: 0; right: 0; bottom: 0; left: 0; z-index: var(--z-overlay, 1100); }
.sheet__mask { position: absolute; top: 0; right: 0; bottom: 0; left: 0; background: $color-mask; }
.sheet__panel { position: absolute; left: 0; right: 0; bottom: 0; background: $color-surface; border-radius: $radius-lg $radius-lg 0 0; padding-bottom: env(safe-area-inset-bottom); animation: up 240ms ease-out; max-height: 80vh; display: flex; flex-direction: column; }
.sheet__head { display: flex; align-items: center; justify-content: space-between; padding: 28rpx 32rpx 8rpx; }
.sheet__title { font-size: $font-h2; font-weight: 600; }
.sheet__close { padding: 8rpx; }
.sheet__body { padding: 16rpx 32rpx 32rpx; max-height: 70vh; box-sizing: border-box; }
@keyframes up { from { transform: translateY(100%); } to { transform: translateY(0); } }
@media (prefers-reduced-motion: reduce) { .sheet__panel { animation: none; } }
</style>
