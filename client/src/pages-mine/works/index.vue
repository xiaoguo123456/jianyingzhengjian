<template>
  <view class="page px">
    <view class="tabs">
      <view v-for="t in tabs" :key="t.key" class="tab" :class="{ 'tab--on': module === t.key }" @tap="select(t.key)">{{ t.label }}</view>
    </view>
    <y-skeleton v-if="loading && !items.length" type="grid" :count="4" />
    <y-empty-state v-else-if="!items.length" text="还没有作品，去生成一张吧" button="去生成" @action="go('/pages/idphoto/index')" />
    <view v-else class="grid">
      <y-work-card v-for="w in items" :key="w.id" :work="w" @select="open" />
    </view>
    <view class="bottom-space" />
  </view>
</template>

<script setup lang="ts">
import { onLoad, onReachBottom, onShow } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { api } from '@/api'
import { go } from '@/platform'
import { useUserStore } from '@/store/user'
import type { Module, Work } from '@/types'

const tabs: { key: Module | 'all'; label: string }[] = [
  { key: 'all', label: '全部' }, { key: 'idphoto', label: '证件照' }, { key: 'pro', label: '职业照' }, { key: 'portrait', label: '写真' }, { key: 'avatar', label: '头像' },
]
const module = ref<Module | 'all'>('all')
const items = ref<Work[]>([])
const page = ref(1)
const hasMore = ref(false)
const loading = ref(true)
let loadedOnce = false

onLoad(async (q) => { module.value = (q?.module as Module) || 'all'; await useUserStore().ready(); await load(true); loadedOnce = true })
onShow(() => { if (loadedOnce) load(true) })
async function load(reset = false) {
  loading.value = true
  if (reset) { page.value = 1 }
  try {
    const r = await api.works({ module: module.value, page: page.value })
    items.value = reset ? r.items : [...items.value, ...r.items]
    hasMore.value = r.has_more
  } finally { loading.value = false }
}
onReachBottom(() => { if (hasMore.value && !loading.value) { page.value++; load() } })
function select(k: Module | 'all') { module.value = k; load(true) }
function open(w: Work) { go(`/pages-mine/work-detail/index?id=${w.id}`) }
</script>

<style lang="scss" scoped>
.tabs { display: flex; gap: 12rpx; padding: 16rpx 0 24rpx; }
.tab { flex: 1; text-align: center; height: 64rpx; line-height: 64rpx; border-radius: $radius-pill; background: #fff; font-size: $font-caption; color: $color-text-2; }
.tab--on { background: $color-primary; color: #fff; font-weight: 600; }
.grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 20rpx; }
</style>
