<template>
  <view class="page">
    <y-skeleton v-if="loading" type="grid" :count="2" class="px" />
    <y-error-state v-else-if="!t" text="模板不存在或已下线" @retry="load" />
    <template v-else>
      <view v-if="landing" class="landing px">
        <view class="landing__hero card">
          <view class="landing__img-wrap"><image class="landing__img" :src="landing.preview_url" mode="aspectFill" /><y-ai-label /></view>
          <view class="landing__text">好友用「{{ t.name }}」生成了这张，试试同款</view>
        </view>
      </view>

      <view class="hero px">
        <view class="hero__img-wrap" :class="{ 'hero__img-wrap--square': t.module === 'avatar' }">
          <image class="hero__img" :src="t.cover_url" mode="aspectFill" />
          <view class="hero__fav" @tap="toggleFav">
            <y-icon :name="t.is_favorited ? 'heart-fill' : 'heart'" :size="44" :color="t.is_favorited ? '#EF4444' : '#FFFFFF'" :stroke-width="2" />
          </view>
          <view class="hero__share" @tap="openShare"><y-icon name="share" :size="40" color="#FFFFFF" :stroke-width="2" /></view>
        </view>
      </view>

      <view class="px info">
        <view class="row">
          <view class="info__name">{{ t.name }}</view>
          <y-tag v-for="tag in t.tags" :key="tag" :text="tag" :variant="tag === '热门' ? 'hot' : 'primary'" />
        </view>
        <view class="info__sub">{{ t.subtitle }}</view>
        <view class="info__meta">
          <text>输出 {{ t.output.width }}×{{ t.output.height }} px · {{ t.output.aspect }}</text>
          <text class="info__dot">·</text>
          <text>消耗 {{ t.credit_cost }} 次生成机会</text>
        </view>
      </view>

      <view v-if="t.sample_urls.length > 1" class="px section">
        <y-section-header title="示例效果" />
        <scroll-view scroll-x class="samples">
          <image v-for="(s, i) in t.sample_urls" :key="i" class="samples__img" :src="s" mode="aspectFill" />
        </scroll-view>
      </view>

      <view class="px tips">
        <view class="tips__title">拍摄建议</view>
        <view class="tips__line">正脸、无遮挡、光线清晰；避免墨镜、极端侧脸、面部过小。</view>
      </view>
      <view class="bottom-space" />
      <y-primary-button sticky text="上传照片生成同款" icon="upload" @press="start" />
    </template>

    <y-share-sheet v-model:visible="shareVisible" :share="pageShare.current.value" mode="template" :timeline="sharePlatform.timeline" :reward-text="rewardText" @chat="pageShare.shareDirect('chat')" @timeline="pageShare.shareDirect('timeline')" @poster="onPoster" />
    <y-poster-preview v-model:visible="posterVisible" :image-url="t?.cover_url || ''" :title="t?.name || ''" :subtitle="'上传照片就能生成同款'" :square="t?.module === 'avatar'" @save="savePoster" />
  </view>
</template>

<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import { api } from '@/api'
import { track } from '@/composables/useAnalytics'
import { usePageShare } from '@/composables/useShare'
import { go, saveImage, share as sharePlatform, toast } from '@/platform'
import { useCatalogueStore } from '@/store/catalogue'
import { useFlowStore } from '@/store/flow'
import { useUserStore } from '@/store/user'
import type { ShareLanding, Template } from '@/types'
import { messageOf } from '@/utils/errors'

const catalogue = useCatalogueStore()
const flow = useFlowStore()
const user = useUserStore()
const id = ref('')
const t = ref<Template | null>(null)
const landing = ref<ShareLanding | null>(null)
const loading = ref(true)
const shareVisible = ref(false)
const posterVisible = ref(false)

const pageShare = usePageShare(() => (t.value ? { type: 'template', template_id: t.value.id, surface: 'template_detail', module: t.value.module } : null))
const rewardText = computed(() => user.shareReward.enabled ? `好友通过你的分享完成首次生成，你获得 ${user.shareReward.per_reward} 次生成机会（每日最多 ${user.shareReward.daily_cap} 次）` : '')

onLoad(async (q) => {
  id.value = q?.id || ''
  await user.ready()
  await load()
  if (q?.s) api.openShare(q.s, 'page').then((l) => { if (l.preview_url) landing.value = l }).catch(() => {})
})
async function load() {
  loading.value = true
  try {
    t.value = await catalogue.template(id.value, true)
    track('template_detail_view', { template_id: id.value })
  } catch { t.value = null } finally { loading.value = false }
}
async function toggleFav() {
  if (!t.value) return
  try { await catalogue.toggleFavorite(t.value.id); toast(t.value.is_favorited ? '已收藏' : '已取消收藏') } catch (e) { toast(messageOf(e)) }
}
function start() {
  if (!t.value) return
  track('generate_click', { template_id: t.value.id, source: 'template_detail' })
  flow.start(t.value.module)
  flow.setTemplate(t.value)
  go(`/pages/upload/index?module=${t.value.module}`)
}
async function openShare() {
  shareVisible.value = true
  if (!pageShare.current.value) await pageShare.prepare()
}
async function onPoster() {
  if (!pageShare.current.value) await pageShare.prepare({ type: 'poster', template_id: id.value, surface: 'template_detail' })
  posterVisible.value = true
}
async function savePoster() {
  try {
    const r = await saveImage(pageShare.current.value?.poster_url || t.value?.cover_url || '')
    track('poster_saved', { template_id: id.value })
    toast(r === 'saved' ? '海报已保存' : '已在新窗口打开')
    posterVisible.value = false
  } catch { /* handled in platform */ }
}
</script>

<style lang="scss" scoped>
.landing { padding-top: 16rpx; }
.landing__hero { display: flex; align-items: center; padding: 16rpx; background: $color-primary-tint; }
.landing__img-wrap { position: relative; width: 120rpx; height: 160rpx; border-radius: $radius-sm; overflow: hidden; flex-shrink: 0; }
.landing__img { width: 100%; height: 100%; }
.landing__text { padding-left: 20rpx; font-size: $font-body; color: $color-text; }
.hero { padding-top: 16rpx; }
.hero__img-wrap { position: relative; width: 100%; padding-top: 120%; border-radius: $radius-lg; overflow: hidden; background: $color-primary-soft; }
.hero__img-wrap--square { padding-top: 100%; }
.hero__img { position: absolute; top: 0; right: 0; bottom: 0; left: 0; width: 100%; height: 100%; }
.hero__fav, .hero__share { position: absolute; top: 20rpx; width: 72rpx; height: 72rpx; border-radius: 50%; background: rgba(17, 24, 39, 0.35); display: flex; align-items: center; justify-content: center; }
.hero__fav { right: 20rpx; }
.hero__share { right: 108rpx; }
.info { padding-top: 24rpx; }
.info__name { font-size: 40rpx; font-weight: 700; margin-right: 12rpx; }
.info__sub { font-size: $font-body; color: $color-text-2; margin-top: 8rpx; }
.info__meta { font-size: $font-caption; color: $color-text-3; margin-top: 8rpx; }
.info__dot { margin: 0 12rpx; }
.samples { white-space: nowrap; }
.samples__img { width: 200rpx; height: 266rpx; border-radius: $radius-sm; margin-right: 16rpx; display: inline-block; }
.tips { margin-top: 40rpx; }
.tips__title { font-size: $font-body-strong; font-weight: 600; }
.tips__line { font-size: $font-caption; color: $color-text-3; margin-top: 6rpx; }
</style>
