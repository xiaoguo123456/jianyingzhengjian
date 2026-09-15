<template>
  <y-sheet :visible="visible" :title="title" @update:visible="(v: boolean) => $emit('update:visible', v)">
    <view v-if="specs" class="grid">
      <y-spec-card v-for="s in specs" :key="s.id" :spec="s" @select="$emit('pick-spec', s)" />
    </view>
    <view v-if="templates" class="grid grid--2">
      <y-template-card v-for="t in templates" :key="t.id" :template="t" :square="t.module === 'avatar'" @select="$emit('pick-template', t)" />
    </view>
  </y-sheet>
</template>

<script setup lang="ts">
import type { Spec, TemplateCard } from '@/types'
defineProps<{ visible: boolean; title: string; specs?: Spec[]; templates?: TemplateCard[] }>()
defineEmits<{ (e: 'update:visible', v: boolean): void; (e: 'pick-spec', s: Spec): void; (e: 'pick-template', t: TemplateCard): void }>()
</script>

<style lang="scss" scoped>
.grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 20rpx; padding-bottom: 16rpx; }
</style>
