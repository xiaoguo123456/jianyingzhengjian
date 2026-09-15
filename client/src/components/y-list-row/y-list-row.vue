<template>
  <view class="lr" hover-class="lr--hover" @tap="$emit('select')">
    <view v-if="icon" class="lr__icon"><y-icon :name="icon" :size="40" :color="iconColor" /></view>
    <view class="lr__body">
      <view class="lr__title">{{ title }}</view>
      <view v-if="caption" class="lr__caption">{{ caption }}</view>
    </view>
    <slot name="right" />
    <view v-if="value" class="lr__value">{{ value }}</view>
    <y-icon v-if="chevron" name="chevron-right" :size="36" color="#C0C8D4" />
  </view>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ title: string; caption?: string; icon?: string; iconColor?: string; value?: string; chevron?: boolean }>(), { chevron: true, iconColor: '#2F7BF6' })
defineEmits<{ (e: 'select'): void }>()
</script>

<style lang="scss" scoped>
.lr { display: flex; align-items: center; min-height: 112rpx; padding: 0 24rpx; }
.lr--hover { background: $color-primary-tint; }
.lr__icon { width: 48rpx; height: 48rpx; margin-right: 20rpx; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.lr__body { flex: 1; min-width: 0; }
.lr__title { font-size: $font-body; color: $color-text; }
.lr__caption { font-size: $font-caption; color: $color-text-3; margin-top: 2rpx; }
.lr__value { font-size: $font-caption; color: $color-text-3; margin-right: 8rpx; }
</style>
