<template>
  <view class="page">
    <y-nav-bar title="证件照" />
    <view class="px">
      <y-skeleton v-if="loading" type="home" />
      <y-error-state v-else-if="error" @retry="load()" />
      <template v-else-if="home">
        <y-banner :banner="home.banner" @press="startUpload('banner')" />
        <view class="section">
          <y-section-header title="常用规格" more="全部规格" @more="go('/pages/spec-library/index')" />
          <view class="grid4">
            <y-spec-card v-for="s in home.hot_specs" :key="s.id" :spec="s" compact @select="pickSpec" />
          </view>
        </view>
        <view class="section">
          <y-section-header title="按用途选择" more="全部用途" @more="go('/pages/spec-library/index')" />
          <view class="usage-list">
            <y-template-row v-for="s in home.more_specs" :key="s.id" compact
              :title="s.name" :caption="mm(s.width_mm, s.height_mm)" @select="pickSpec(s)" />
          </view>
        </view>
      </template>
      <view class="tab-bottom" />
    </view>
  </view>
</template>

<script setup lang="ts">
import { useHome } from '@/composables/useHome'
import { go } from '@/platform'
import { mm } from '@/utils/format'
const { home, loading, error, load, startUpload, pickSpec } = useHome('idphoto')
</script>

<style lang="scss" scoped>
.grid4 { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12rpx; }
.usage-list { border-radius: $radius-md; overflow: hidden; background: $color-surface; }
</style>
