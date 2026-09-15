<template>
  <scroll-view scroll-x class="rail" :show-scrollbar="false">
    <view class="rail__inner">
      <view v-for="t in templates" :key="t.id" class="rail__item" :style="{ width: itemWidth + 'rpx' }">
        <y-template-card :template="t" :square="square" @select="$emit('select', t)" />
      </view>
    </view>
  </scroll-view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TemplateCard } from '@/types'

const props = defineProps<{ templates: TemplateCard[]; square?: boolean }>()
defineEmits<{ (e: 'select', t: TemplateCard): void }>()
/* 露出下一张照片，保留横向滑动的视觉提示。 */
const itemWidth = computed(() => (props.square ? 240 : 264))
</script>

<style lang="scss" scoped>
.rail { width: calc(100% + 64rpx); white-space: nowrap; margin: 0 -32rpx; }
.rail__inner { display: inline-flex; gap: 20rpx; padding: 4rpx 32rpx 12rpx; }
.rail__item { display: inline-block; flex-shrink: 0; }
</style>
