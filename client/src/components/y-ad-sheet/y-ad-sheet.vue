<template>
  <y-sheet :visible="visible" title="免费生成" @update:visible="onClose">
    <view class="ad">
      <view class="ad__icon"><y-icon name="play" :size="56" /></view>
      <view class="ad__title">观看一段视频</view>
      <view class="ad__desc">即可获得 1 次生成机会</view>
      <y-primary-button text="看视频免费生成" icon="play" :loading="busy" @press="$emit('watch')" />
      <view class="ad__cancel" @tap="onClose(false)">暂不需要</view>
    </view>
  </y-sheet>
</template>

<script setup lang="ts">
defineProps<{ visible: boolean; busy?: boolean }>()
const emit = defineEmits<{ (e: 'update:visible', v: boolean): void; (e: 'watch'): void; (e: 'cancel'): void }>()
const onClose = (v: boolean) => { if (!v) { emit('update:visible', false); emit('cancel') } }
</script>

<style lang="scss" scoped>
.ad { text-align: center; padding: 8rpx 0 16rpx; }
.ad__icon { width: 120rpx; height: 120rpx; border-radius: 50%; background: $color-primary-soft; display: flex; align-items: center; justify-content: center; margin: 0 auto 20rpx; }
.ad__title { font-size: $font-h2; font-weight: 600; }
.ad__desc { font-size: $font-body; color: $color-text-2; margin: 8rpx 0 36rpx; }
.ad__cancel { margin-top: 24rpx; font-size: $font-body; color: $color-text-3; padding: 12rpx; }
</style>
