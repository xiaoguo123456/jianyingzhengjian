<template>
  <view class="page">
    <y-nav-bar title="职业照" />
    <view class="px">
      <y-skeleton v-if="loading" type="home" variant="pro" />
      <y-error-state v-else-if="error" @retry="load()" />
      <template v-else-if="home">
        <y-banner :banner="home.banner" variant="professional" @press="startUpload('banner')" />
        <view class="section">
          <y-section-header title="精选形象" more="全部模板" @more="openMore('职业照模板')" />
          <view class="scene-list">
            <view v-for="c in home.hot_categories" :key="c.id" class="scene" hover-class="scene--hover" @tap="openCategory(c.id, c.name)">{{ c.name }}</view>
          </view>
          <view class="template-grid">
            <y-template-card v-for="t in templates" :key="t.id" :template="t" @select="openTemplate" />
          </view>
        </view>
      </template>
      <view class="tab-bottom" />
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useHome } from '@/composables/useHome'
import { homeTemplates } from '@/utils/home-layout'
const { home, loading, error, load, startUpload, openTemplate, openCategory, openMore } = useHome('pro')
const templates = computed(() => homeTemplates(home.value))
</script>

<style lang="scss" scoped>
.scene-list { display: flex; flex-wrap: wrap; gap: 12rpx; margin-bottom: 28rpx; }
.scene { flex: 1; min-width: 120rpx; text-align: center; padding: 18rpx 8rpx; border-radius: $radius-sm; background: #ECEFF3; font-size: 24rpx; color: $color-text-2; white-space: nowrap; }
.scene--hover { background: $color-primary-soft; color: $color-primary; }
</style>
