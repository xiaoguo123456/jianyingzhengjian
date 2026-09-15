<template>
  <view class="page">
    <y-nav-bar title="职业照" />
    <view class="px">
      <y-skeleton v-if="loading" type="home" />
      <y-error-state v-else-if="error" @retry="load()" />
      <template v-else-if="home">
        <y-banner :banner="home.banner" cta="上传照片生成" @tap="startUpload('banner')" />

        <view class="section">
          <y-section-header title="热门场景" more="更多场景" @more="openMore('职业照模板')" />
          <view class="grid4">
            <y-category-card v-for="c in home.hot_categories" :key="c.id" :name="c.name" :icon="c.icon" :cover="c.cover_url" @tap="openCategory(c.id, c.name)" />
          </view>
        </view>

        <view class="section">
          <y-section-header title="推荐模板" more="查看更多" @more="openMore('职业照模板')" />
          <y-template-rail :templates="home.hot_templates || []" @tap="openTemplate" />
        </view>

        <view v-for="r in home.rails" :key="r.title" class="section">
          <y-section-header :title="r.title" more="更多" @more="r.category_id ? openCategory(r.category_id, r.title) : openMore(r.title)" />
          <y-template-rail :templates="r.templates" @tap="openTemplate" />
        </view>
      </template>
      <view class="tab-bottom" />
    </view>
  </view>
</template>

<script setup lang="ts">
import { useHome } from '@/composables/useHome'
const { home, loading, error, load, startUpload, openTemplate, openCategory, openMore } = useHome('pro')
</script>

<style lang="scss" scoped>
.grid4 { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16rpx; }
.tab-bottom { height: calc(48rpx + env(safe-area-inset-bottom)); }
</style>
