<template>
  <view class="nav" :style="{ paddingTop: statusBar + 'px' }">
    <view class="nav__row" :style="{ height: navHeight + 'px', paddingRight: capsuleRight + 'px' }">
      <view v-if="back" class="nav__back" @tap="onBack"><y-icon name="chevron-left" :size="48" color="#111827" /></view>
      <view v-if="tabs && tabs.length" class="nav__tabs">
        <view v-for="t in tabs" :key="t.key" class="nav__tab" :class="{ 'nav__tab--on': t.key === active }" @tap="$emit('change', t.key)">
          <text class="nav__tab-text">{{ t.label }}</text>
          <view class="nav__tab-bar" />
        </view>
      </view>
      <view v-else class="nav__title" :class="{ 'nav__title--large': large }">{{ title }}</view>
    </view>
    <view v-if="subtitle" class="nav__subtitle">{{ subtitle }}</view>
  </view>
</template>

<script setup lang="ts">
import { useSafeArea } from '@/composables/useSafeArea'
import { back as goBack } from '@/platform'

withDefaults(
  defineProps<{ title: string; subtitle?: string; back?: boolean; large?: boolean; tabs?: { key: string; label: string }[]; active?: string }>(),
  { subtitle: '', back: false, large: true },
)
defineEmits<{ (e: 'change', key: string): void }>()
const { statusBar, navHeight, capsuleRight } = useSafeArea()
const onBack = () => goBack()
</script>

<style lang="scss" scoped>
.nav { padding: 0 $space-4 20rpx; background: transparent; }
.nav__row { display: flex; align-items: center; }
.nav__back { margin-left: -12rpx; margin-right: 8rpx; width: 64rpx; height: 64rpx; display: flex; align-items: center; justify-content: center; }
.nav__title { font-size: $font-h2; font-weight: 600; color: $color-text; }
.nav__title--large { font-size: $font-display; font-weight: 700; line-height: 1.2; letter-spacing: -0.5rpx; }
.nav__subtitle { font-size: $font-body; color: $color-text-2; margin-top: 4rpx; }
/* Segment titles: same size for both labels so they share one baseline; only colour and weight change. */
.nav__tabs { display: flex; align-items: center; gap: 40rpx; }
.nav__tab { display: flex; flex-direction: column; align-items: center; }
.nav__tab-text { font-size: $font-display; line-height: 1.2; font-weight: 500; color: $color-text-3; }
.nav__tab-bar { width: 28rpx; height: 6rpx; border-radius: 3rpx; margin-top: 8rpx; background: $color-primary; opacity: 0; }
.nav__tab--on .nav__tab-text { font-weight: 700; color: $color-text; }
.nav__tab--on .nav__tab-bar { opacity: 1; }
</style>
