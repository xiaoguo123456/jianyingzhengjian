<template>
  <view class="page">
    <y-nav-bar title="证件照" />
    <view class="px">
      <y-skeleton v-if="loading" type="home" />
      <y-error-state v-else-if="error" @retry="load()" />
      <template v-else-if="home">
        <y-banner :banner="home.banner" cta="上传照片制作" @tap="startUpload('banner')" />

        <view class="section">
          <y-section-header title="常用规格" more="更多规格" @more="go('/pages/spec-library/index')" />
          <view class="grid4">
            <y-spec-card v-for="s in home.hot_specs" :key="s.id" :spec="s" @tap="pickSpec" />
          </view>
        </view>

        <view class="section">
          <y-section-header title="常见用途" more="查看全部" @more="go('/pages/spec-library/index')" />
          <view class="list">
            <y-template-row
              v-for="s in home.more_specs" :key="s.id"
              :title="s.name" :caption="mm(s.width_mm, s.height_mm)" :caption2="px(s.width_px, s.height_px)"
              :ratio="s.width_mm / s.height_mm" :glyph-color="s.bg_default" @tap="pickSpec(s)" />
          </view>
        </view>

        <view class="section tips card">
          <view class="tips__title">拍摄建议</view>
          <view class="tips__line">正脸、无遮挡、光线均匀，面部占画面三分之一以上。生成后可免费更换背景颜色。</view>
        </view>
      </template>
      <view class="tab-bottom" />
    </view>
  </view>
</template>

<script setup lang="ts">
import { useHome } from '@/composables/useHome'
import { go } from '@/platform'
import { mm, px } from '@/utils/format'

const { home, loading, error, load, startUpload, pickSpec } = useHome('idphoto')
</script>

<style lang="scss" scoped>
.grid4 { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16rpx; }
.list { display: flex; flex-direction: column; gap: 16rpx; }
.tips { padding: 24rpx; }
.tips__title { font-size: $font-body-strong; font-weight: 600; }
.tips__line { font-size: $font-caption; color: $color-text-3; margin-top: 8rpx; line-height: 1.7; }
.tab-bottom { height: calc(48rpx + env(safe-area-inset-bottom)); }
</style>
