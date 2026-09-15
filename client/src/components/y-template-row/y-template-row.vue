<template>
  <view class="tr card" :class="{ 'tr--compact': compact }" hover-class="tr--hover" @tap="$emit('select')">
    <view v-if="!compact" class="tr__thumb-wrap" :class="{ 'tr__thumb-wrap--glyph': !image }">
      <image v-if="image" class="tr__thumb" :src="image" mode="aspectFill" lazy-load />
      <view v-else class="tr__glyph" :style="glyphStyle"><view class="tr__head" /><view class="tr__torso" /></view>
    </view>
    <view class="tr__body">
      <view class="tr__title ellipsis">{{ title }}</view>
      <view v-if="caption" class="tr__caption num">{{ caption }}</view>
      <view v-if="caption2" class="tr__caption num">{{ caption2 }}</view>
    </view>
    <y-icon v-if="chevron" name="chevron-right" :size="36" color="#C0C8D4" />
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
const props = withDefaults(defineProps<{ title: string; caption?: string; caption2?: string; image?: string; ratio?: number; glyphColor?: string; chevron?: boolean; compact?: boolean }>(), { chevron: true, ratio: 0.714, glyphColor: '#438EDB' })
defineEmits<{ (e: 'select'): void }>()
const glyphStyle = computed(() => {
  const h = 64
  return { width: `${Math.round(h * props.ratio)}rpx`, height: `${h}rpx`, background: props.glyphColor, borderColor: props.glyphColor === '#FFFFFF' ? '#CBD5E1' : props.glyphColor, '--silhouette': props.glyphColor === '#FFFFFF' ? '#91A7C4' : '#FFFFFF' }
})
</script>

<style lang="scss" scoped>
.tr { display: flex; align-items: center; padding: 20rpx; }
.tr--hover { opacity: 0.85; }
.tr__thumb-wrap { width: 120rpx; height: 160rpx; border-radius: $radius-sm; overflow: hidden; background: $color-primary-soft; flex-shrink: 0; display: flex; align-items: center; justify-content: center; }
.tr__thumb-wrap--glyph { height: 120rpx; }
.tr__thumb { width: 100%; height: 100%; }
.tr__glyph { position: relative; border-radius: 6rpx; border: 2rpx solid; box-sizing: border-box; overflow: hidden; }
.tr__head { position: absolute; left: 50%; top: 14%; width: 30%; padding-top: 30%; transform: translateX(-50%); border-radius: 50%; background: var(--silhouette, #FFFFFF); }
.tr__torso { position: absolute; left: 50%; bottom: -8%; width: 62%; padding-top: 40%; transform: translateX(-50%); border-radius: 50% 50% 0 0; background: var(--silhouette, #FFFFFF); }
.tr__body { flex: 1; min-width: 0; padding: 0 20rpx; }
.tr__title { font-size: $font-body-strong; font-weight: 600; color: $color-text; margin-bottom: 8rpx; }
.tr__caption { font-size: $font-caption; color: $color-text-3; line-height: 1.5; }
.tr--compact { min-height: 104rpx; padding: 0 24rpx; border-radius: 0; border-bottom: 1px solid $color-border; }
.tr--compact .tr__body { display: flex; align-items: center; justify-content: space-between; gap: 12rpx; padding: 0 12rpx 0 0; }
.tr--compact .tr__title { font-size: 28rpx; font-weight: 400; margin: 0; }
.tr--compact .tr__caption { flex-shrink: 0; font-size: 24rpx; }
</style>
