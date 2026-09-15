<template>
  <view class="spec card" :class="{ 'spec--selected': selected }" hover-class="spec--hover" @tap="$emit('tap', spec)">
    <view class="spec__glyph-wrap">
      <view class="spec__glyph" :style="glyphStyle">
        <view class="spec__head" />
        <view class="spec__body" />
      </view>
    </view>
    <view class="spec__name">{{ spec.name }}</view>
    <view class="spec__size num">{{ mmText }}</view>
    <view class="spec__size spec__size--px num">{{ pxText }}</view>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Spec } from '@/types'
import { mm, px } from '@/utils/format'

const props = defineProps<{ spec: Spec; selected?: boolean }>()
defineEmits<{ (e: 'tap', spec: Spec): void }>()
const mmText = computed(() => mm(props.spec.width_mm, props.spec.height_mm))
const pxText = computed(() => px(props.spec.width_px, props.spec.height_px))
const glyphStyle = computed(() => {
  const r = props.spec.width_mm / props.spec.height_mm
  const h = 64
  const light = props.spec.bg_default === '#FFFFFF'
  return { width: `${Math.round(h * r)}rpx`, height: `${h}rpx`, background: props.spec.bg_default, borderColor: light ? '#CBD5E1' : props.spec.bg_default }
})
</script>

<style lang="scss" scoped>
.spec { padding: 24rpx 8rpx 20rpx; text-align: center; border: 2rpx solid transparent; }
.spec--hover { opacity: 0.85; }
.spec--selected { border-color: $color-primary; background: $color-primary-tint; }
.spec__glyph-wrap { height: 72rpx; display: flex; align-items: flex-end; justify-content: center; margin-bottom: 14rpx; }
/* a tiny ID-photo glyph: frame in the spec's aspect ratio with a person silhouette */
.spec__glyph { position: relative; border-radius: 6rpx; border: 2rpx solid; box-sizing: border-box; overflow: hidden; }
.spec__head { position: absolute; left: 50%; top: 14%; width: 30%; padding-top: 30%; transform: translateX(-50%); border-radius: 50%; background: rgba(255, 255, 255, 0.92); }
.spec__body { position: absolute; left: 50%; bottom: -8%; width: 62%; padding-top: 40%; transform: translateX(-50%); border-radius: 50% 50% 0 0; background: rgba(255, 255, 255, 0.92); }
.spec__name { font-size: $font-body-strong; font-weight: 600; color: $color-text; margin-bottom: 6rpx; }
.spec__size { font-size: 22rpx; color: $color-text-2; line-height: 1.5; }
.spec__size--px { color: $color-text-3; }
</style>
