<template>
  <view v-if="visible" class="pp">
    <view class="pp__mask" @tap="$emit('update:visible', false)" />
    <view class="pp__panel">
      <view class="poster">
        <view class="poster__img-wrap" :class="{ 'poster__img-wrap--square': square }">
          <image class="poster__img" :src="imageUrl" mode="aspectFill" />
          <y-ai-label />
        </view>
        <view class="poster__body">
          <view class="poster__title">{{ title }}</view>
          <view class="poster__sub">{{ subtitle }}</view>
          <view class="poster__foot">
            <view class="poster__brand">映己证件照写真馆</view>
            <view class="poster__qr"><y-icon name="qr" :size="80" color="#111827" /></view>
          </view>
        </view>
      </view>
      <view class="pp__actions">
        <view class="pp__btn pp__btn--ghost" @tap="$emit('update:visible', false)">取消</view>
        <view class="pp__btn" @tap="$emit('save')">保存到相册</view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
defineProps<{ visible: boolean; imageUrl: string; title: string; subtitle: string; square?: boolean }>()
defineEmits<{ (e: 'update:visible', v: boolean): void; (e: 'save'): void }>()
</script>

<style lang="scss" scoped>
/* In production the poster is rendered server-side (docs/SHARING.md §5); this is the in-app preview. */
.pp { position: fixed; top: 0; right: 0; bottom: 0; left: 0; z-index: 110; display: flex; align-items: center; justify-content: center; }
.pp__mask { position: absolute; top: 0; right: 0; bottom: 0; left: 0; background: rgba(17, 24, 39, 0.7); }
.pp__panel { position: relative; width: 560rpx; }
.poster { background: #fff; border-radius: $radius-md; overflow: hidden; }
.poster__img-wrap { position: relative; width: 100%; padding-top: 133.33%; background: $color-primary-soft; }
.poster__img-wrap--square { padding-top: 100%; }
.poster__img { position: absolute; top: 0; right: 0; bottom: 0; left: 0; width: 100%; height: 100%; }
.poster__body { padding: 24rpx 28rpx 28rpx; }
.poster__title { font-size: $font-h2; font-weight: 700; }
.poster__sub { font-size: $font-caption; color: $color-text-2; margin-top: 6rpx; }
.poster__foot { display: flex; align-items: center; justify-content: space-between; margin-top: 20rpx; }
.poster__brand { font-size: $font-caption; color: $color-primary; font-weight: 600; }
.poster__qr { display: flex; }
.pp__actions { display: flex; gap: 20rpx; margin-top: 24rpx; }
.pp__btn { flex: 1; height: 88rpx; line-height: 88rpx; text-align: center; border-radius: $radius-pill; background: $gradient-primary; color: #fff; font-weight: 600; }
.pp__btn--ghost { background: rgba(255, 255, 255, 0.15); color: #fff; }
</style>
