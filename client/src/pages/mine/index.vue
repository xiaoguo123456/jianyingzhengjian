<template>
  <view class="page">
    <view class="profile px" :style="{ paddingTop: top + 16 + 'px' }">
      <image v-if="user.user?.avatar_url" class="profile__avatar" :src="user.user.avatar_url" mode="aspectFill" />
      <view v-else class="profile__avatar profile__avatar--empty"><y-icon name="user" :size="48" color="#697386" /></view>
      <view class="profile__body">
        <view class="profile__name ellipsis">{{ user.user?.nickname || '未设置昵称' }}</view>
        <view class="profile__edit" hover-class="pressed" @tap="go('/pages-mine/profile-edit/index')">编辑资料<y-icon name="chevron-right" :size="22" color="#697386" /></view>
      </view>
    </view>
    <view class="px">
      <view class="works-section">
        <y-section-header title="我的作品" :more="total ? `全部 ${total} 张` : ''" @more="go('/pages-mine/works/index')" />
        <view class="filters">
          <view v-for="m in filters" :key="m.key" class="filter" :class="{ 'filter--on': activeModule === m.key }" hover-class="pressed" @tap="selectModule(m.key)">{{ m.label }}</view>
        </view>
        <y-skeleton v-if="worksLoading" type="works" :count="3" />
        <y-error-state v-else-if="worksError" @retry="load()" />
        <view v-else-if="recentWorks.length" class="works-grid">
          <view v-for="w in recentWorks" :key="w.id" class="work" hover-class="pressed" @tap="go(`/pages-mine/work-detail/index?id=${w.id}`)">
            <view class="work__photo">
              <image class="work__image" :src="w.thumb_url" mode="aspectFill" />
              <y-ai-label v-if="w.ai_label" />
            </view>
            <view class="work__name ellipsis">{{ w.template?.name || w.spec?.name || MODULE_NAME[w.module] }}</view>
          </view>
        </view>
        <view v-else class="works-empty" @tap="createWork">
          <y-icon name="image" :size="40" color="#697386" /><text>还没有作品，去制作</text><y-icon name="chevron-right" :size="24" color="#697386" />
        </view>
      </view>
      <view class="section">
        <y-credit-bar :credits="user.credits" compact tappable :invite-text="inviteText"
          @open="go('/pages-mine/credits/index')" @invite="go('/pages-mine/credits/index')" @watch="gen.adVisible.value = true" />
      </view>
      <view class="section card services">
        <view class="service" hover-class="pressed" @tap="go('/pages-mine/records/index')">
          <y-icon name="file-text" :size="40" color="#4B5563" /><text>生成记录</text>
        </view>
        <view class="service" hover-class="pressed" @tap="go('/pages-mine/favorites/index')">
          <y-icon name="star" :size="40" color="#4B5563" /><text>收藏模板</text>
        </view>
        <view class="service" hover-class="pressed" @tap="go('/pages-mine/photos/index')">
          <y-icon name="album" :size="40" color="#4B5563" /><text>原始照片</text>
        </view>
        <!-- #ifdef MP-WEIXIN -->
        <button class="service service--button" hover-class="pressed" open-type="contact">
          <y-icon name="headset" :size="40" color="#4B5563" /><text>联系客服</text>
        </button>
        <!-- #endif -->
        <!-- #ifndef MP-WEIXIN -->
        <view class="service" hover-class="pressed" @tap="toast('请在微信小程序内联系客服')">
          <y-icon name="headset" :size="40" color="#4B5563" /><text>联系客服</text>
        </view>
        <!-- #endif -->
      </view>
      <view class="card settings"><y-list-row icon="settings" title="设置" @select="go('/pages-mine/settings/index')" /></view>
      <view class="footer">
        <text class="footer__link" @tap="openWebview(env.privacyPolicyUrl, '隐私政策')">隐私政策</text>
        <text>·</text>
        <text class="footer__link" @tap="openWebview(env.userAgreementUrl, '用户协议')">用户协议</text>
      </view>
      <view class="tab-bottom" />
    </view>
    <y-ad-sheet v-model:visible="gen.adVisible.value" :busy="gen.adBusy.value" @watch="gen.watchAd" @cancel="gen.cancelAd" />
  </view>
</template>

<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import { api } from '@/api'
import { useGenerate } from '@/composables/useGenerate'
import { useSafeArea } from '@/composables/useSafeArea'
import { env } from '@/config'
import { go, openWebview, toast } from '@/platform'
import { useUserStore } from '@/store/user'
import { useUiStore } from '@/store/ui'
import { MODULE_NAME, type Module, type Work, type WorksSummary } from '@/types'
const user = useUserStore()
const ui = useUiStore()
const gen = useGenerate()
const { top } = useSafeArea()
const summary = ref<WorksSummary | null>(null)
const activeModule = ref<Module | 'all'>('all')
const recentWorks = ref<Work[]>([])
const worksLoading = ref(true)
const worksError = ref(false)
let requestVersion = 0
const filters: { key: Module | 'all'; label: string }[] = [
  { key: 'all', label: '全部' }, { key: 'idphoto', label: '证件照' }, { key: 'pro', label: '职业照' }, { key: 'portrait', label: '写真' }, { key: 'avatar', label: '头像' },
]
onShow(load)
async function load() {
  const version = ++requestVersion
  const module = activeModule.value
  worksLoading.value = true
  worksError.value = false
  try {
    await user.ready()
    user.refreshCredits().catch(() => {})
    const [nextSummary, works] = await Promise.all([api.worksSummary(), api.works({ module, page: 1 })])
    if (version !== requestVersion) return
    summary.value = nextSummary
    recentWorks.value = works.items.slice(0, 3)
  } catch {
    if (version === requestVersion) worksError.value = true
  } finally {
    if (version === requestVersion) worksLoading.value = false
  }
}
function createWork() {
  const module = activeModule.value
  if (module === 'avatar' || module === 'portrait') {
    ui.setPortraitSegment(module)
    go('/pages/portrait/index')
  } else {
    go(`/pages/${module === 'all' ? 'idphoto' : module}/index`)
  }
}
function selectModule(module: Module | 'all') {
  if (activeModule.value === module) return
  activeModule.value = module
  load()
}
const total = computed(() => summary.value?.total ?? user.user?.works_count ?? 0)
const inviteText = computed(() => user.shareReward.enabled ? '邀请好友得次数' : '')
</script>

<style lang="scss" scoped>
.profile { display: flex; align-items: center; padding-bottom: 36rpx; }
.profile__avatar { width: 100rpx; height: 100rpx; border-radius: 50%; background: $color-primary-soft; flex-shrink: 0; }
.profile__avatar--empty { display: flex; align-items: center; justify-content: center; }
.profile__body { flex: 1; min-width: 0; padding-left: 24rpx; }
.profile__name { font-size: 36rpx; font-weight: 600; line-height: 1.35; }
.profile__edit { display: inline-flex; align-items: center; min-height: 52rpx; gap: 4rpx; color: $color-text-3; font-size: 24rpx; }
.works-section { margin-top: 12rpx; }
.filters { display: flex; gap: 12rpx; margin-bottom: 24rpx; }
.filter { flex: 1; min-width: 0; min-height: 68rpx; display: flex; align-items: center; justify-content: center; font-size: 24rpx; color: $color-text-2; border-radius: $radius-sm; }
.filter--on { color: $color-primary; background: $color-primary-soft; font-weight: 600; }
.works-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16rpx; }
.work { min-width: 0; }
.work__photo { position: relative; padding-top: 133.33%; border-radius: $radius-sm; overflow: hidden; background: #E7E9ED; }
.work__image { position: absolute; inset: 0; width: 100%; height: 100%; }
.work__name { margin-top: 12rpx; font-size: 24rpx; color: $color-text-2; }
.works-empty { min-height: 180rpx; display: flex; align-items: center; justify-content: center; gap: 16rpx; color: $color-text-3; font-size: 26rpx; border-radius: $radius-md; background: #fff; }
.services { display: flex; padding: 28rpx 8rpx; }
.service { flex: 1; display: flex; flex-direction: column; align-items: center; gap: 14rpx; padding: 8rpx 0; font-size: 24rpx; color: $color-text-2; }
.service--button { background: transparent; margin: 0; line-height: 1.5; border-radius: 0; }
.service--button::after { border: 0; }
.settings { margin-top: 20rpx; overflow: hidden; }
.footer { display: flex; align-items: center; justify-content: center; gap: 16rpx; font-size: 22rpx; color: $color-text-3; margin-top: 28rpx; }
.footer__link { padding: 12rpx 4rpx; }
.pressed { opacity: 0.7; }
</style>
