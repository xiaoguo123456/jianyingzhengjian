<template>
  <view class="page px">
    <scroll-view scroll-x class="chips" :show-scrollbar="false">
      <view class="chips__inner">
        <view class="chip" :class="{ 'chip--on': !category }" @tap="select('')">全部</view>
        <view v-for="c in categories" :key="c.id" class="chip" :class="{ 'chip--on': category === c.id }" @tap="select(c.id)">{{ c.name }}</view>
      </view>
    </scroll-view>

    <y-skeleton v-if="loading" type="list" :count="5" />
    <y-empty-state v-else-if="!specs.length" icon="id-card" text="该分类暂无规格" />
    <view v-else class="list">
      <y-template-row
        v-for="s in specs" :key="s.id"
        :title="s.name + (s.note ? ' · ' + s.note : '')" :caption="mm(s.width_mm, s.height_mm)" :caption2="px(s.width_px, s.height_px)"
        :ratio="s.width_mm / s.height_mm" :glyph-color="s.bg_default" @select="pick(s)" />
    </view>
    <view class="note">规格数据来自后台配置，尺寸以办事机构要求为准。</view>
  </view>
</template>

<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { api } from '@/api'
import { track } from '@/composables/useAnalytics'
import { go } from '@/platform'
import { useFlowStore } from '@/store/flow'
import { useUserStore } from '@/store/user'
import type { Spec } from '@/types'
import { mm, px } from '@/utils/format'

const flow = useFlowStore()
const categories = ref<{ id: string; name: string }[]>([])
const specs = ref<Spec[]>([])
const category = ref('')
const loading = ref(true)

onLoad(async (q) => {
  await useUserStore().ready()
  category.value = q?.category || ''
  categories.value = await api.specCategories()
  await load()
  if (q?.spec) {
    const s = specs.value.find((x) => x.id === q.spec)
    if (s) pick(s)
  }
})
async function load() {
  loading.value = true
  try { specs.value = await api.specs({ category_id: category.value || undefined }) } finally { loading.value = false }
}
function select(id: string) { category.value = id; load() }
function pick(s: Spec) {
  track('spec_click', { spec_id: s.id, source: 'library' })
  flow.start('idphoto')
  flow.setSpec(s)
  go('/pages/upload/index?module=idphoto')
}
</script>

<style lang="scss" scoped>
.chips { white-space: nowrap; margin: 16rpx -32rpx 0; padding: 0 32rpx; }
.chips__inner { display: inline-flex; gap: 16rpx; padding: 8rpx 0 24rpx; }
.chip { padding: 0 28rpx; height: 64rpx; line-height: 64rpx; border-radius: $radius-pill; background: #fff; font-size: $font-body; color: $color-text-2; }
.chip--on { background: $color-primary; color: #fff; font-weight: 600; }
.list { display: flex; flex-direction: column; gap: 16rpx; }
.note { font-size: 22rpx; color: $color-text-3; text-align: center; padding: 40rpx 0 60rpx; }
</style>
