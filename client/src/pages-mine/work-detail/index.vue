<template>
  <view class="page px">
    <y-skeleton v-if="!work" type="grid" :count="1" />
    <template v-else>
      <view class="img card" :class="{ 'img--square': work.module === 'avatar' }" @tap="preview">
        <image class="img__el" :src="work.url" mode="aspectFit" />
        <y-ai-label v-if="work.ai_label" />
      </view>
      <view class="card info">
        <y-list-row title="类型" :value="MODULE_NAME[work.module]" :chevron="false" />
        <y-list-row v-if="work.template" title="模板" :value="work.template.name" :chevron="false" />
        <y-list-row v-if="work.spec" title="规格" :value="`${work.spec.name} · ${mm(work.spec.width_mm, work.spec.height_mm)} · ${px(work.spec.width_px, work.spec.height_px)}`" :chevron="false" />
        <y-list-row v-if="work.meta.bg" title="背景" :value="bgName(work.meta.bg)" :chevron="false" />
        <y-list-row title="生成时间" :value="dateShort(work.created_at)" :chevron="false" />
      </view>
      <view class="actions">
        <y-primary-button text="保存图片" icon="download" @press="save" />
        <y-primary-button text="再次生成" secondary icon="refresh" @press="again" />
        <view class="del" @tap="remove">删除作品</view>
      </view>
      <view class="bottom-space" />
    </template>
  </view>
</template>

<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { api } from '@/api'
import { track } from '@/composables/useAnalytics'
import { back, go, saveImage, toast } from '@/platform'
import { useUserStore } from '@/store/user'
import { MODULE_NAME, type Work } from '@/types'
import { messageOf } from '@/utils/errors'
import { bgName, dateShort, mm, px } from '@/utils/format'

const work = ref<Work | null>(null)
onLoad(async (q) => { await useUserStore().ready(); try { work.value = await api.work(q?.id || '') } catch (e) { toast(messageOf(e)) } })
function preview() { if (work.value) uni.previewImage({ urls: [work.value.url] }) }
async function save() {
  if (!work.value) return
  try { const { url } = await api.downloadUrl(work.value.id); const r = await saveImage(url); track('work_saved', { work_id: work.value.id }); toast(r === 'saved' ? '已保存到相册' : '已在新窗口打开') } catch { /* handled */ }
}
function again() {
  if (!work.value) return
  if (work.value.template) go(`/pages/template-detail/index?id=${work.value.template.id}`)
  else go(`/pages/spec-library/index?spec=${work.value.spec?.id || ''}`)
}
function remove() {
  uni.showModal({ title: '删除作品', content: '删除后无法恢复，确定删除？', confirmColor: '#EF4444', success: async (r) => {
    if (!r.confirm || !work.value) return
    await api.deleteWork(work.value.id)
    toast('已删除')
    back()
  } })
}
</script>

<style lang="scss" scoped>
.img { position: relative; width: 100%; padding-top: 133.33%; margin-top: 16rpx; overflow: hidden; background: #fff; }
.img--square { padding-top: 100%; }
.img__el { position: absolute; top: 0; right: 0; bottom: 0; left: 0; width: 100%; height: 100%; }
.info { margin-top: 20rpx; overflow: hidden; }
.actions { margin-top: 24rpx; display: flex; flex-direction: column; gap: 20rpx; }
.del { text-align: center; font-size: $font-body; color: $color-danger; padding: 16rpx; }
</style>
