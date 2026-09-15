<template>
  <view class="page px">
    <y-skeleton v-if="loading" type="grid" :count="2" />
    <y-empty-state v-else-if="!items.length" icon="star" text="还没有收藏模板" button="去看看" @action="go('/pages/portrait/index')" />
    <view v-else class="grid">
      <y-template-card v-for="t in items" :key="t.id" :template="t" :square="t.module === 'avatar'" @select="go(`/pages/template-detail/index?id=${t.id}`)" />
    </view>
    <view class="bottom-space" />
  </view>
</template>

<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { api } from '@/api'
import { go } from '@/platform'
import { useUserStore } from '@/store/user'
import type { TemplateCard } from '@/types'

const items = ref<TemplateCard[]>([])
const loading = ref(true)
onShow(async () => { await useUserStore().ready(); try { items.value = (await api.favorites()).items } finally { loading.value = false } })
</script>

<style lang="scss" scoped>
.grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 20rpx; padding-top: 16rpx; }
</style>
