<template>
  <view class="page px">
    <scroll-view v-if="!collection && categories.length" scroll-x class="chips" :show-scrollbar="false">
      <view class="chips__inner">
        <view class="chip" :class="{ 'chip--on': !category }" @tap="select('')">全部</view>
        <view v-for="c in categories" :key="c.id" class="chip" :class="{ 'chip--on': category === c.id }" @tap="select(c.id)">{{ c.name }}</view>
      </view>
    </scroll-view>

    <y-skeleton v-if="loading && !items.length" type="grid" :count="4" />
    <y-empty-state v-else-if="!items.length" text="暂无模板" />
    <view v-else class="grid">
      <y-template-card v-for="t in items" :key="t.id" :template="t" :square="module === 'avatar'" @select="open" />
    </view>
    <view v-if="hasMore" class="more">{{ loading ? '加载中…' : '上拉加载更多' }}</view>
    <view class="bottom-space" />
  </view>
</template>

<script setup lang="ts">
import { onLoad, onReachBottom } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { api } from '@/api'
import { track } from '@/composables/useAnalytics'
import { go } from '@/platform'
import { useUserStore } from '@/store/user'
import type { Category, Module, TemplateCard } from '@/types'

const module = ref<Module>('pro')
const category = ref('')
const collection = ref('')
const categories = ref<Category[]>([])
const items = ref<TemplateCard[]>([])
const page = ref(1)
const hasMore = ref(false)
const loading = ref(true)

onLoad(async (q) => {
  await useUserStore().ready()
  module.value = (q?.module as Module) || 'pro'
  category.value = q?.category || ''
  collection.value = q?.collection || ''
  if (q?.title) uni.setNavigationBarTitle({ title: decodeURIComponent(q.title) })
  if (!collection.value) categories.value = await api.templateCategories(module.value)
  await load(true)
})
async function load(reset = false) {
  loading.value = true
  if (reset) { page.value = 1; items.value = [] }
  try {
    const r = await api.templates({ module: module.value, category_id: category.value || undefined, collection_id: collection.value || undefined, page: page.value })
    items.value = reset ? r.items : [...items.value, ...r.items]
    hasMore.value = r.has_more
  } finally { loading.value = false }
}
onReachBottom(() => { if (hasMore.value && !loading.value) { page.value++; load() } })
function select(id: string) { category.value = id; load(true) }
function open(t: TemplateCard) {
  track('template_click', { template_id: t.id, module: module.value, source: 'list' })
  go(`/pages/template-detail/index?id=${t.id}`)
}
</script>

<style lang="scss" scoped>
.chips { white-space: nowrap; margin: 16rpx -32rpx 0; padding: 0 32rpx; }
.chips__inner { display: inline-flex; gap: 16rpx; padding: 8rpx 0 24rpx; }
.chip { padding: 0 28rpx; height: 64rpx; line-height: 64rpx; border-radius: $radius-pill; background: #fff; font-size: $font-body; color: $color-text-2; }
.chip--on { background: $color-primary; color: #fff; font-weight: 600; }
.grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 20rpx; padding-top: 16rpx; }
.more { text-align: center; font-size: $font-caption; color: $color-text-3; padding: 32rpx; }
</style>
