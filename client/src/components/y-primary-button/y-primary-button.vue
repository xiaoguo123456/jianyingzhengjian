<template>
  <view :class="['pb-wrap', { 'pb-wrap--sticky': sticky }]">
    <view class="pb" :class="{ 'pb--disabled': disabled || loading, 'pb--secondary': secondary }" hover-class="pb--hover" @tap="onTap">
      <view v-if="loading" class="pb__spinner" />
      <y-icon v-else-if="icon" :name="icon" :size="40" :color="secondary ? '#2F7BF6' : '#FFFFFF'" :stroke-width="2" />
      <text class="pb__text">{{ text }}</text>
    </view>
    <slot name="under" />
  </view>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{ text: string; icon?: string; loading?: boolean; disabled?: boolean; sticky?: boolean; secondary?: boolean }>(), { loading: false, disabled: false, sticky: false, secondary: false })
const emit = defineEmits<{ (e: 'press'): void }>()
const onTap = () => { if (!props.disabled && !props.loading) emit('press') }
</script>

<style lang="scss" scoped>
.pb-wrap--sticky {
  position: fixed; left: 0; right: 0; bottom: var(--window-bottom, 0); z-index: 20;
  padding: 16rpx 32rpx calc(16rpx + env(safe-area-inset-bottom));
  background: rgba(255, 255, 255, 0.96); box-shadow: $shadow-sheet;
}
.pb {
  height: 96rpx; border-radius: $radius-pill; background: $gradient-primary; color: #fff;
  display: flex; align-items: center; justify-content: center; gap: 12rpx;
  box-shadow: 0 8rpx 24rpx rgba(58, 141, 255, 0.28);
}
.pb--secondary { background: $color-primary-soft; color: $color-primary; box-shadow: none; }
.pb--hover { opacity: 0.85; }
.pb--disabled { opacity: 0.4; }
.pb__text { font-size: 32rpx; font-weight: 600; }
.pb__spinner { width: 32rpx; height: 32rpx; border: 4rpx solid rgba(255, 255, 255, 0.5); border-top-color: #fff; border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
