<template>
  <scroll-view scroll-x class="rail" :show-scrollbar="false">
    <view class="rail__inner">
      <view v-for="t in templates" :key="t.id" class="rail__item" :style="{ width: itemWidth + 'rpx' }">
        <y-template-card :template="t" :square="square" @tap="$emit('tap', t)" />
      </view>
    </view>
  </scroll-view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TemplateCard } from '@/types'

const props = defineProps<{ templates: TemplateCard[]; square?: boolean }>()
defineEmits<{ (e: 'tap', t: TemplateCard): void }>()
/* ~2.5 cards visible so the row reads as scrollable. */
const itemWidth = computed(() => (props.square ? 240 : 264))
</script>

<style lang="scss" scoped>
.rail { white-space: nowrap; margin: 0 -32rpx; }
.rail__inner { display: inline-flex; gap: 20rpx; padding: 4rpx 32rpx 12rpx; }
.rail__item { display: inline-block; flex-shrink: 0; }
</style>
