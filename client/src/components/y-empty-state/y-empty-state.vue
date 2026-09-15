<template>
  <view class="es">
    <view class="es__icon"><y-icon :name="icon" :size="72" color="#B9C6DA" /></view>
    <view class="es__text">{{ text }}</view>
    <view v-if="button" class="es__btn" hover-class="es__btn--hover" @tap="$emit('action')">{{ button }}</view>
  </view>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ icon?: string; text: string; button?: string }>(), { icon: 'image' })
defineEmits<{ (e: 'action'): void }>()
</script>

<style lang="scss" scoped>
.es { text-align: center; padding: 120rpx 32rpx; }
.es__icon { width: 160rpx; height: 160rpx; border-radius: 50%; background: #fff; display: flex; align-items: center; justify-content: center; margin: 0 auto 24rpx; }
.es__text { font-size: $font-body; color: $color-text-3; }
.es__btn { display: inline-block; margin-top: 32rpx; padding: 0 40rpx; height: 72rpx; line-height: 72rpx; border-radius: $radius-pill; background: $color-primary-soft; color: $color-primary; font-size: $font-body; font-weight: 600; }
.es__btn--hover { opacity: 0.8; }
</style>
