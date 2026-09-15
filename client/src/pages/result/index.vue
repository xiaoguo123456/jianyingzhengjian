<template>
  <view class="page px">
    <y-skeleton v-if="!work" type="grid" :count="1" />
    <template v-else>
      <view class="img card" :class="{ 'img--square': work.module === 'avatar' }" @tap="preview">
        <image class="img__el" :src="work.url" mode="aspectFit" />
        <y-ai-label v-if="work.ai_label" />
      </view>
      <view class="meta">
        <view class="meta__name">{{ work.template?.name || work.spec?.name }}</view>
        <view v-if="work.spec" class="meta__cap num">{{ mm(work.spec.width_mm, work.spec.height_mm) }} · {{ px(work.spec.width_px, work.spec.height_px) }}<text v-if="work.meta.bg"> · 背景 {{ bgName(work.meta.bg) }}</text></view>
        <view v-else class="meta__cap">{{ work.width }}×{{ work.height }} px</view>
        <view v-if="work.ai_label" class="meta__cap">本图片由 AI 生成</view>
      </view>

      <view class="actions">
        <view class="act" hover-class="act--hover" @tap="save"><view class="act__icon"><y-icon name="download" :size="40" /></view><text>保存图片</text></view>
        <view class="act" hover-class="act--hover" @tap="regenerate"><view class="act__icon"><y-icon name="refresh" :size="40" /></view><text>再生成一张</text></view>
        <view class="act" hover-class="act--hover" @tap="changeTemplate"><view class="act__icon"><y-icon name="layers" :size="40" /></view><text>换一个模板</text></view>
        <view class="act" hover-class="act--hover" @tap="openShare"><view class="act__icon"><y-icon name="share" :size="40" /></view><text>分享</text></view>
      </view>

      <view v-if="work.module === 'idphoto' && work.spec" class="section">
        <y-section-header title="换背景" />
        <view class="swatches card">
          <view v-for="c in bgOptions" :key="c" class="swatch" :class="{ 'swatch--on': work.meta.bg === c }" @tap="recolor(c)">
            <view class="swatch__color" :style="{ background: c, borderColor: c === '#FFFFFF' ? '#CBD5E1' : c }" />
            <view class="swatch__name">{{ bgName(c) }}</view>
          </view>
          <view class="swatch__free">免费，即时</view>
        </view>
        <view class="clothing card" hover-class="act--hover" @tap="changeClothing">
          <y-icon name="shirt" :size="40" />
          <view class="clothing__text"><view class="clothing__title">换服装</view><view class="clothing__cap">重新生成，消耗 1 次</view></view>
          <y-icon name="chevron-right" :size="36" color="#C0C8D4" />
        </view>
      </view>
      <view class="bottom-space" />
    </template>

    <y-ad-sheet v-model:visible="gen.adVisible.value" :busy="gen.adBusy.value" @watch="gen.watchAd" @cancel="gen.cancelAd" />
    <y-share-sheet v-model:visible="shareVisible" v-model:mode="shareMode" :share="pageShare.current.value" allow-work :timeline="sharePlatform.timeline && shareMode === 'template'" :reward-text="rewardText" @chat="pageShare.shareDirect('chat')" @timeline="pageShare.shareDirect('timeline')" @poster="onPoster" />
    <y-poster-preview v-model:visible="posterVisible" :image-url="work?.url || ''" :title="work?.template?.name || work?.spec?.name || ''" subtitle="上传照片就能生成同款" :square="work?.module === 'avatar'" @save="savePoster" />
  </view>
</template>

<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, ref, watch } from 'vue'
import { api } from '@/api'
import { track } from '@/composables/useAnalytics'
import { useGenerate } from '@/composables/useGenerate'
import { usePageShare } from '@/composables/useShare'
import { go, replace, saveImage, share as sharePlatform, toast } from '@/platform'
import { useFlowStore } from '@/store/flow'
import { useUserStore } from '@/store/user'
import type { Work } from '@/types'
import { messageOf } from '@/utils/errors'
import { bgName, mm, px } from '@/utils/format'
import { uuid } from '@/utils/uuid'

const flow = useFlowStore()
const user = useUserStore()
const gen = useGenerate()
const work = ref<Work | null>(null)
const taskId = ref('')
const shareVisible = ref(false)
const shareMode = ref<'template' | 'work'>('template')
const posterVisible = ref(false)
const bgOptions = ['#FFFFFF', '#438EDB', '#FF0000', '#808080']

const pageShare = usePageShare(() => {
  if (!work.value) return null
  return shareMode.value === 'work'
    ? { type: 'work', work_id: work.value.id, surface: 'result', module: work.value.module }
    : work.value.template ? { type: 'template', template_id: work.value.template.id, surface: 'result', module: work.value.module } : { type: 'work', work_id: work.value.id, surface: 'result', module: work.value.module }
})
const rewardText = computed(() => user.shareReward.enabled ? `好友通过你的分享完成首次生成，你获得 ${user.shareReward.per_reward} 次生成机会（每日最多 ${user.shareReward.daily_cap} 次）` : '')
watch(shareMode, () => { if (shareVisible.value) pageShare.prepare() })

onLoad(async (q) => {
  taskId.value = q?.task_id || ''
  await user.ready()
  try { work.value = await api.work(q?.work_id || '') } catch (e) { toast(messageOf(e)) }
  track('result_view', { work_id: q?.work_id })
})

function preview() { if (work.value) uni.previewImage({ urls: [work.value.url] }) }
async function save() {
  if (!work.value) return
  try {
    const { url } = await api.downloadUrl(work.value.id)
    const r = await saveImage(url)
    track('work_saved', { work_id: work.value.id })
    toast(r === 'saved' ? '已保存到相册' : '已在新窗口打开', r === 'saved' ? 'success' : 'none')
  } catch { /* handled */ }
}
async function regenerate() {
  if (!work.value) return
  track('regenerate_click', { work_id: work.value.id })
  const t = await gen.submit(() => api.regenerate(work.value!.task_id, uuid()))
  if (t) replace(`/pages/generating/index?task_id=${t.id}`)
}
function changeTemplate() {
  if (!work.value) return
  if (work.value.module === 'idphoto') return go('/pages/spec-library/index')
  go(`/pages/template-list/index?module=${work.value.module}&title=${encodeURIComponent('换一个模板')}`)
}
async function recolor(bg: string) {
  if (!work.value || work.value.meta.bg === bg) return
  track('recolor_click', { work_id: work.value.id, bg })
  uni.showLoading({ title: '处理中' })
  try { work.value = await api.recolor(work.value.id, bg) } catch (e) { toast(messageOf(e)) } finally { uni.hideLoading() }
}
function changeClothing() {
  if (!work.value?.spec) return
  if (!flow.photo || flow.specId !== work.value.spec.id) { toast('请重新上传照片'); return go('/pages/spec-library/index') }
  flow.newAttempt(taskId.value)
  go('/pages/idphoto-params/index')
}
async function openShare() {
  shareVisible.value = true
  if (!pageShare.current.value) await pageShare.prepare()
}
async function onPoster() {
  if (!work.value) return
  await pageShare.prepare({ type: 'poster', work_id: work.value.id, surface: 'result', module: work.value.module })
  posterVisible.value = true
}
async function savePoster() {
  try {
    const r = await saveImage(pageShare.current.value?.poster_url || work.value?.url || '')
    track('poster_saved', { work_id: work.value?.id })
    toast(r === 'saved' ? '海报已保存' : '已在新窗口打开')
    posterVisible.value = false
  } catch { /* handled */ }
}
</script>

<style lang="scss" scoped>
.img { position: relative; width: 100%; padding-top: 133.33%; margin-top: 16rpx; overflow: hidden; background: #fff; }
.img--square { padding-top: 100%; }
.img__el { position: absolute; top: 0; right: 0; bottom: 0; left: 0; width: 100%; height: 100%; }
.meta { margin-top: 20rpx; }
.meta__name { font-size: $font-h2; font-weight: 600; }
.meta__cap { font-size: $font-caption; color: $color-text-3; margin-top: 4rpx; }
.actions { display: flex; justify-content: space-between; margin-top: 28rpx; }
.act { text-align: center; font-size: $font-caption; color: $color-text-2; width: 22%; }
.act--hover { opacity: 0.7; }
.act__icon { width: 96rpx; height: 96rpx; border-radius: 50%; background: #fff; box-shadow: $shadow-card; display: flex; align-items: center; justify-content: center; margin: 0 auto 10rpx; }
.swatches { display: flex; align-items: center; gap: 24rpx; padding: 24rpx; }
.swatch { text-align: center; }
.swatch__color { width: 72rpx; height: 72rpx; border-radius: 50%; border: 4rpx solid; box-sizing: border-box; margin: 0 auto 6rpx; }
.swatch--on .swatch__color { box-shadow: 0 0 0 4rpx #fff, 0 0 0 8rpx $color-primary; }
.swatch__name { font-size: 22rpx; color: $color-text-2; }
.swatch__free { margin-left: auto; font-size: 22rpx; color: $color-success; }
.clothing { display: flex; align-items: center; padding: 20rpx 24rpx; margin-top: 16rpx; }
.clothing__text { flex: 1; padding-left: 16rpx; }
.clothing__title { font-size: $font-body; font-weight: 600; }
.clothing__cap { font-size: 22rpx; color: $color-text-3; }
</style>
