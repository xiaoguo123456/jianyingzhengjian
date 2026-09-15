<template>
  <view class="spec" :class="{ 'spec--selected': selected }" hover-class="spec--hover" @tap="$emit('select', spec)">
    <view class="spec__glyph-wrap">
      <view class="spec__glyph" :style="glyphStyle"><view class="spec__inner" /></view>
    </view>
    <view class="spec__name">{{ spec.name }}</view>
    <view class="spec__size num">{{ mmText }}</view>
    <view v-if="!compact" class="spec__size num">{{ pxText }}</view>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Spec } from '@/types'
import { mm, px } from '@/utils/format'
const props = defineProps<{ spec: Spec; selected?: boolean; compact?: boolean }>()
defineEmits<{ (e: 'select', spec: Spec): void }>()
const mmText = computed(() => mm(props.spec.width_mm, props.spec.height_mm))
const pxText = computed(() => px(props.spec.width_px, props.spec.height_px))
const glyphStyle = computed(() => {
  // 同一比例尺下呈现相纸尺寸，让一寸、二寸的差别可见。
  const scale = Math.min(1.4, 72 / Math.max(props.spec.width_mm, props.spec.height_mm))
  return { width: `${props.spec.width_mm * scale}rpx`, height: `${props.spec.height_mm * scale}rpx` }
})
</script>

<style lang="scss" scoped>
.spec { padding: 22rpx 4rpx 20rpx; text-align: center; background: $color-surface; border: 2rpx solid transparent; border-radius: $radius-md; }
.spec--selected { border-color: $color-primary; background: $color-primary-tint; }
.spec--hover { background: $color-primary-soft; }
.spec__glyph-wrap { height: 76rpx; display: flex; align-items: center; justify-content: center; margin-bottom: 14rpx; }
.spec__glyph { padding: 5rpx; box-sizing: border-box; border: 2rpx solid #91ACD6; border-radius: 4rpx; background: #fff; }
.spec__inner { width: 100%; height: 100%; background: #E2EAF6; border-radius: 1rpx; }
.spec__name { font-size: 28rpx; font-weight: 500; color: $color-text; margin-bottom: 6rpx; }
.spec__size { font-size: 21rpx; color: $color-text-3; line-height: 1.5; white-space: nowrap; }
</style>
