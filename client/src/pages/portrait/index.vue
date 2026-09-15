<template>
  <view class="page">
    <y-nav-bar title="写真" :tabs="segments" :active="seg" @change="switchSeg" />
    <view class="px">
      <!-- 写真 -->
      <view v-show="seg === 'portrait'">
        <y-skeleton v-if="pLoading" type="home" />
        <y-error-state v-else-if="pError" @retry="pLoad()" />
        <template v-else-if="pHome">
          <y-banner :banner="pHome.banner" cta="上传照片生成" @tap="pUpload('banner')" />

          <view class="section">
            <y-section-header title="热门风格" more="更多风格" @more="pOpenMore('写真模板')" />
            <view class="grid4">
              <y-category-card v-for="c in pHome.hot_categories" :key="c.id" :name="c.name" :cover="c.cover_url" @tap="pOpenCategory(c.id, c.name)" />
            </view>
          </view>

          <view class="section">
            <y-section-header title="精选模板" more="查看更多" @more="pOpenMore('写真模板')" />
            <y-template-rail :templates="pHome.hot_templates || []" @tap="pOpenTemplate" />
          </view>

          <view class="section">
            <y-section-header title="专题" />
            <view class="grid2">
              <view v-for="c in pHome.collections" :key="c.id" class="col" hover-class="col--hover" @tap="pOpenCollection(c.id, c.name)">
                <image class="col__img" :src="c.cover_url" mode="aspectFill" lazy-load />
                <view class="col__scrim" />
                <view class="col__text">
                  <view class="col__name">{{ c.name }}</view>
                  <view class="col__count">{{ c.template_count }} 个模板</view>
                </view>
              </view>
            </view>
          </view>

          <view v-for="r in pHome.rails" :key="r.title" class="section">
            <y-section-header :title="r.title" more="更多" @more="r.category_id ? pOpenCategory(r.category_id, r.title) : pOpenMore(r.title)" />
            <y-template-rail :templates="r.templates" @tap="pOpenTemplate" />
          </view>
        </template>
      </view>

      <!-- 头像 -->
      <view v-show="seg === 'avatar'">
        <y-skeleton v-if="aLoading" type="home" />
        <y-error-state v-else-if="aError" @retry="aLoad()" />
        <template v-else-if="aHome">
          <y-banner :banner="aHome.banner" cta="上传照片生成" @tap="aUpload('banner')" />

          <view class="section">
            <y-section-header title="头像类型" more="更多类型" @more="aOpenMore('头像模板')" />
            <view class="grid4">
              <y-category-card v-for="c in aHome.hot_categories" :key="c.id" :name="c.name" :icon="c.icon" :cover="c.cover_url" @tap="aOpenCategory(c.id, c.name)" />
            </view>
          </view>

          <view class="section">
            <y-section-header title="热门模板" more="查看更多" @more="aOpenMore('头像模板')" />
            <y-template-rail :templates="aHome.hot_templates || []" square @tap="aOpenTemplate" />
          </view>

          <view v-for="r in aHome.rails" :key="r.title" class="section">
            <y-section-header :title="r.title" more="更多" @more="r.category_id ? aOpenCategory(r.category_id, r.title) : aOpenMore(r.title)" />
            <y-template-rail :templates="r.templates" square @tap="aOpenTemplate" />
          </view>
        </template>
      </view>
      <view class="tab-bottom" />
    </view>
  </view>
</template>

<script setup lang="ts">
import { onLoad, onShow } from '@dcloudio/uni-app'
import { computed } from 'vue'
import { track } from '@/composables/useAnalytics'
import { useHome } from '@/composables/useHome'
import { useUiStore } from '@/store/ui'
import type { PortraitSegment } from '@/utils/routes'

const ui = useUiStore()
const segments = [
  { key: 'portrait', label: '写真' },
  { key: 'avatar', label: '头像' },
]
const seg = computed(() => ui.portraitSegment)

const {
  home: pHome, loading: pLoading, error: pError, load: pLoad, startUpload: pUpload,
  openTemplate: pOpenTemplate, openCategory: pOpenCategory, openCollection: pOpenCollection, openMore: pOpenMore,
} = useHome('portrait', { trackView: false })
const {
  home: aHome, loading: aLoading, error: aError, load: aLoad, startUpload: aUpload,
  openTemplate: aOpenTemplate, openCategory: aOpenCategory, openMore: aOpenMore,
} = useHome('avatar', { trackView: false })

onLoad((q) => { if (q?.seg === 'avatar' || q?.seg === 'portrait') ui.setPortraitSegment(q.seg) })
onShow(() => track('tab_view', { module: seg.value }))

function switchSeg(key: string) {
  const next = key as PortraitSegment
  if (next === seg.value) return
  ui.setPortraitSegment(next)
  uni.pageScrollTo({ scrollTop: 0, duration: 0 })
  track('tab_view', { module: next, source: 'segment' })
}
</script>

<style lang="scss" scoped>
.grid4 { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16rpx; }
.grid2 { display: grid; grid-template-columns: repeat(2, 1fr); gap: 20rpx; }
.col { position: relative; padding-top: 62%; border-radius: $radius-md; overflow: hidden; background: $color-primary-soft; }
.col--hover { opacity: 0.88; }
.col__img { position: absolute; top: 0; right: 0; bottom: 0; left: 0; width: 100%; height: 100%; }
.col__scrim { position: absolute; top: 0; right: 0; bottom: 0; left: 0; background: linear-gradient(180deg, rgba(17, 24, 39, 0) 30%, rgba(17, 24, 39, 0.65) 100%); }
.col__text { position: absolute; left: 20rpx; right: 20rpx; bottom: 16rpx; color: #fff; }
.col__name { font-size: $font-body-strong; font-weight: 700; text-shadow: 0 2rpx 6rpx rgba(0, 0, 0, 0.25); }
.col__count { font-size: 22rpx; opacity: 0.85; margin-top: 2rpx; }
.tab-bottom { height: calc(48rpx + env(safe-area-inset-bottom)); }
</style>
