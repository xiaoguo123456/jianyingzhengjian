<template>
  <view class="si">
    <view v-for="(s, i) in stages" :key="s.key" class="si__step" :class="{ 'si__step--done': i < idx, 'si__step--active': i === idx }">
      <view class="si__dot">
        <y-icon v-if="i < idx" name="check" :size="24" color="#FFFFFF" :stroke-width="2.5" />
        <view v-else-if="i === idx" class="si__pulse" />
      </view>
      <view class="si__label">{{ s.label }}</view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TaskStage, TaskStatus } from '@/types'

const props = defineProps<{ stage: TaskStage | null; status: TaskStatus }>()
const stages = [
  { key: 'queued', label: '照片处理中' },
  { key: 'processing', label: '正在生成' },
  { key: 'finishing', label: '正在优化结果' },
]
const idx = computed(() => {
  if (props.status === 'success') return 3
  if (props.status === 'waiting') return 0
  const i = stages.findIndex((s) => s.key === props.stage)
  return i < 0 ? 0 : i
})
</script>

<style lang="scss" scoped>
.si { display: flex; flex-direction: column; gap: 28rpx; }
.si__step { display: flex; align-items: center; gap: 20rpx; color: $color-text-3; }
.si__dot { width: 40rpx; height: 40rpx; border-radius: 50%; background: $color-border; display: flex; align-items: center; justify-content: center; }
.si__step--done .si__dot { background: $color-success; }
.si__step--active .si__dot { background: $color-primary-soft; }
.si__step--active { color: $color-text; font-weight: 600; }
.si__step--done { color: $color-text-2; }
.si__pulse { width: 18rpx; height: 18rpx; border-radius: 50%; background: $color-primary; animation: pulse 1.2s ease-in-out infinite; }
.si__label { font-size: $font-body; }
@keyframes pulse { 0%, 100% { transform: scale(0.8); opacity: 0.6; } 50% { transform: scale(1.2); opacity: 1; } }
</style>
