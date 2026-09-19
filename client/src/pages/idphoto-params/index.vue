<template>
  <view class="page px">
    <view class="preview card">
      <image v-if="flow.photo" class="preview__img" :src="flow.photo.preview_url" mode="aspectFit" />
      <view class="preview__meta">
        <view class="preview__name">{{ flow.spec?.name }}</view>
        <view v-if="flow.spec" class="preview__cap num">{{ mm(flow.spec.width_mm, flow.spec.height_mm) }} · {{ px(flow.spec.width_px, flow.spec.height_px) }}</view>
      </view>
    </view>

    <view class="section">
      <y-section-header title="背景颜色" />
      <view class="swatches">
        <view v-for="c in flow.spec?.bg_allowed || []" :key="c" class="swatch" :class="{ 'swatch--on': flow.params.bg === c }" @tap="flow.setParams({ bg: c })">
          <view class="swatch__color" :style="{ background: c, borderColor: c === '#FFFFFF' ? '#CBD5E1' : c }" />
          <view class="swatch__name">{{ bgName(c) }}</view>
        </view>
      </view>
    </view>

    <view class="section">
      <y-section-header title="服装" />
      <view class="opts">
        <view class="opt" :class="{ 'opt--on': flow.params.clothing === 'keep' }" @tap="flow.setParams({ clothing: 'keep' })">
          <view class="opt__name">保持原服装</view>
        </view>
      </view>
      <view v-for="g in groups" :key="g.key" class="group">
        <view class="group__title">{{ g.label }}</view>
        <view class="opts">
          <view v-for="o in g.items" :key="o.id" class="opt" :class="{ 'opt--on': flow.params.clothing === o.id }" @tap="flow.setParams({ clothing: o.id })">
            <view class="opt__name">{{ o.name }}</view>
          </view>
        </view>
      </view>
    </view>

    <view class="section">
      <y-section-header title="美化" />
      <view class="seg">
        <view class="seg__item" :class="{ 'seg__item--on': flow.params.beauty === 'natural' }" @tap="flow.setParams({ beauty: 'natural' })">自然<text class="seg__cap">不修饰</text></view>
        <view class="seg__item" :class="{ 'seg__item--on': flow.params.beauty === 'light' }" @tap="flow.setParams({ beauty: 'light' })">轻度<text class="seg__cap">自然修饰</text></view>
      </view>
      <view class="hint">保持本人特征，不提供传统美颜参数。</view>
    </view>

    <view class="bottom-space" />
    <y-primary-button sticky :text="`下一步 · 消耗 ${flow.creditCost} 次生成机会`" @press="go('/pages/confirm/index')" />
  </view>
</template>

<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import { api } from '@/api'
import { go, back } from '@/platform'
import { useFlowStore } from '@/store/flow'
import type { ClothingOption } from '@/types'
import { bgName, mm, px } from '@/utils/format'

const flow = useFlowStore()
const clothing = ref<ClothingOption[]>([])
onLoad(async () => {
  if (!flow.spec || !flow.photo) { back(); return }
  clothing.value = await api.clothingOptions()
})
const groups = computed(() => [
  { key: 'male', label: '男士', items: clothing.value.filter((c) => c.group === 'male') },
  { key: 'female', label: '女士', items: clothing.value.filter((c) => c.group === 'female') },
])
</script>

<style lang="scss" scoped>
.preview { display: flex; align-items: center; padding: 20rpx; margin-top: 16rpx; }
.preview__img { width: 160rpx; height: 214rpx; border-radius: $radius-sm; background: $color-bg; }
.preview__meta { padding-left: 24rpx; }
.preview__name { font-size: $font-h2; font-weight: 600; }
.preview__cap { font-size: $font-caption; color: $color-text-3; margin-top: 6rpx; }
.swatches { display: flex; gap: 24rpx; }
.swatch { text-align: center; }
.swatch__color { width: 88rpx; height: 88rpx; border-radius: 50%; border: 4rpx solid; box-sizing: border-box; margin: 0 auto 8rpx; }
.swatch--on .swatch__color { box-shadow: 0 0 0 6rpx #fff, 0 0 0 10rpx $color-primary; }
.swatch__name { font-size: $font-caption; color: $color-text-2; }
.opts { display: flex; flex-wrap: wrap; gap: 16rpx; }
.opt { padding: 16rpx 24rpx; border-radius: $radius-md; background: #fff; border: 2rpx solid transparent; min-width: 180rpx; text-align: center; }
.opt--on { border-color: $color-primary; background: $color-primary-tint; }
.opt__name { font-size: $font-body; font-weight: 600; }
.opt__cap { font-size: 22rpx; color: $color-text-3; margin-top: 2rpx; }
.group { margin-top: 20rpx; }
.group__title { font-size: $font-caption; color: $color-text-3; margin-bottom: 12rpx; }
.seg { display: flex; background: #fff; border-radius: $radius-md; padding: 8rpx; }
.seg__item { flex: 1; text-align: center; padding: 16rpx 0; border-radius: 16rpx; font-size: $font-body; font-weight: 600; color: $color-text-2; }
.seg__item--on { background: $color-primary-soft; color: $color-primary; }
.seg__cap { display: block; font-size: 22rpx; font-weight: 400; color: $color-text-3; margin-top: 2rpx; }
.hint { font-size: 22rpx; color: $color-text-3; margin-top: 12rpx; }
</style>
