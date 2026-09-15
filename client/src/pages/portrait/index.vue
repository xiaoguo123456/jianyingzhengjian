<template>
  <view class="page">
    <y-nav-bar title="写真" :tabs="segments" :active="seg" @change="switchSeg" />
    <view class="px">
      <view v-show="seg === 'portrait'">
        <y-skeleton v-if="pLoading" type="home" variant="portrait" />
        <y-error-state v-else-if="pError" @retry="pLoad()" />
        <template v-else-if="pHome">
          <y-banner :banner="pHome.banner" variant="editorial" @press="pUpload('banner')" />
          <view class="style-list">
            <view v-for="c in pHome.hot_categories" :key="c.id" class="style-item" hover-class="style-item--hover" @tap="pOpenCategory(c.id, c.name)">{{ c.name }}</view>
            <view class="style-item style-item--more" hover-class="style-item--hover" @tap="pOpenMore('写真模板')">全部<y-icon name="chevron-right" :size="22" color="#697386" /></view>
          </view>
          <view class="section selection">
            <y-section-header title="精选写真" more="全部模板" @more="pOpenMore('写真模板')" />
            <view class="template-grid">
              <y-template-card v-for="t in pTemplates" :key="t.id" :template="t" @select="pOpenTemplate" />
            </view>
          </view>
          <view v-if="pHome.collections?.length" class="section">
            <y-section-header title="灵感专题" />
            <view class="collections">
              <view v-for="c in pHome.collections.slice(0, 2)" :key="c.id" class="collection" hover-class="collection--hover" @tap="pOpenCollection(c.id, c.name)">
                <image class="collection__image" :src="c.cover_url" mode="aspectFill" lazy-load />
                <view class="collection__scrim" />
                <view class="collection__name">{{ c.name }}</view>
                <view class="collection__arrow"><y-icon name="chevron-right" color="#FFFFFF" :size="28" /></view>
              </view>
            </view>
          </view>
        </template>
      </view>
      <view v-show="seg === 'avatar'">
        <y-skeleton v-if="aLoading" type="home" variant="avatar" />
        <y-error-state v-else-if="aError" @retry="aLoad()" />
        <template v-else-if="aHome">
          <y-banner :banner="aHome.banner" @press="aUpload('banner')" />
          <view class="style-list">
            <view v-for="c in aHome.hot_categories" :key="c.id" class="style-item" hover-class="style-item--hover" @tap="aOpenCategory(c.id, c.name)">{{ c.name }}</view>
          </view>
          <view class="section selection">
            <y-section-header title="精选头像" more="全部模板" @more="aOpenMore('头像模板')" />
            <view class="template-grid">
              <y-template-card v-for="t in aTemplates" :key="t.id" :template="t" square @select="aOpenTemplate" />
            </view>
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
import { homeTemplates } from '@/utils/home-layout'
import type { PortraitSegment } from '@/utils/routes'
const ui = useUiStore()
const segments = [{ key: 'portrait', label: '写真' }, { key: 'avatar', label: '头像' }]
const seg = computed(() => ui.portraitSegment)
const {
  home: pHome, loading: pLoading, error: pError, load: pLoad, startUpload: pUpload,
  openTemplate: pOpenTemplate, openCategory: pOpenCategory, openCollection: pOpenCollection, openMore: pOpenMore,
} = useHome('portrait', { trackView: false })
const {
  home: aHome, loading: aLoading, error: aError, load: aLoad, startUpload: aUpload,
  openTemplate: aOpenTemplate, openCategory: aOpenCategory, openMore: aOpenMore,
} = useHome('avatar', { trackView: false })
const pTemplates = computed(() => homeTemplates(pHome.value))
const aTemplates = computed(() => homeTemplates(aHome.value))
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
.style-list { display: flex; flex-wrap: wrap; gap: 8rpx; padding-top: 24rpx; }
.style-item { display: flex; align-items: center; justify-content: center; gap: 2rpx; flex: 1; min-height: 76rpx; padding: 0 10rpx; border-radius: $radius-sm; background: #ECEFF3; color: $color-text-2; font-size: 23rpx; white-space: nowrap; }
.style-item--more { flex: 0 0 auto; background: transparent; padding-right: 0; }
.style-item--hover { background: $color-primary-soft; color: $color-primary; }
.selection { margin-top: 32rpx; }
.collections { display: flex; flex-direction: column; gap: 20rpx; }
.collection { position: relative; height: 236rpx; border-radius: $radius-md; overflow: hidden; background: #667883; }
.collection--hover { opacity: 0.88; }
.collection__image, .collection__scrim { position: absolute; inset: 0; width: 100%; height: 100%; }
.collection__scrim { background: linear-gradient(90deg, rgba(17, 24, 39, 0.55), rgba(17, 24, 39, 0) 80%); }
.collection__name { position: absolute; left: 28rpx; bottom: 28rpx; color: #fff; font-size: 32rpx; font-weight: 500; }
.collection__arrow { position: absolute; right: 24rpx; bottom: 28rpx; }
</style>
