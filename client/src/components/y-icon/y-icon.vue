<template>
  <view class="y-icon" :style="style" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { iconSrc } from '@/utils/icons'

const props = withDefaults(defineProps<{ name: string; size?: number; color?: string; strokeWidth?: number }>(), {
  size: 48, color: '#2864DC', strokeWidth: 1.75,
})
const style = computed(() => ({
  width: `${props.size}rpx`,
  height: `${props.size}rpx`,
  backgroundImage: `url("${iconSrc(props.name, props.color, props.strokeWidth)}")`,
}))
</script>

<style lang="scss" scoped>
.y-icon { display: inline-block; background-repeat: no-repeat; background-position: center; background-size: contain; flex-shrink: 0; }
</style>
