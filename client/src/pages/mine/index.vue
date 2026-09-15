<template>
  <view class="page">
    <view class="head" :style="{ paddingTop: top + 8 + 'px' }">
      <view class="profile px">
        <image v-if="user.user?.avatar_url" class="profile__avatar" :src="user.user.avatar_url" mode="aspectFill" />
        <view v-else class="profile__avatar profile__avatar--empty"><y-icon name="user" :size="64" color="#B9C6DA" /></view>
        <view class="profile__body">
          <view class="profile__name ellipsis">{{ user.user?.nickname || '未设置昵称' }}</view>
          <view class="profile__cap">已生成 <text class="num">{{ total }}</text> 张作品</view>
        </view>
        <view class="profile__edit" hover-class="profile__edit--hover" @tap="go('/pages-mine/profile-edit/index')">编辑资料</view>
      </view>
    </view>

    <view class="px body">
      <y-credit-bar
        :credits="user.credits" tappable :invite-text="inviteText"
        @open="go('/pages-mine/credits/index')" @invite="go('/pages-mine/credits/index')" @watch="gen.adVisible.value = true" />

      <view class="section">
        <y-section-header title="我的作品" :more="total ? `全部 ${total} 张` : ''" @more="go('/pages-mine/works/index')" />
        <view class="grid4">
          <view v-for="m in modules" :key="m" class="wk" hover-class="wk--hover" @tap="go(`/pages-mine/works/index?module=${m}`)">
            <template v-if="cover(m)">
              <image class="wk__img" :src="cover(m)" mode="aspectFill" />
              <view class="wk__scrim" />
              <view class="wk__text wk__text--light">
                <view class="wk__name">{{ MODULE_NAME[m] }}</view>
                <view class="wk__count num">{{ count(m) }} 张</view>
              </view>
            </template>
            <template v-else>
              <view class="wk__empty"><y-icon name="image" :size="44" color="#B9C6DA" /></view>
              <view class="wk__text">
                <view class="wk__name">{{ MODULE_NAME[m] }}</view>
                <view class="wk__count num">0 张</view>
              </view>
            </template>
          </view>
        </view>
      </view>

      <view class="section card svc">
        <view class="svc__item" hover-class="svc__item--hover" @tap="go('/pages-mine/records/index')">
          <view class="svc__icon"><y-icon name="file-text" :size="44" /></view>
          <text class="svc__label">生成记录</text>
        </view>
        <view class="svc__item" hover-class="svc__item--hover" @tap="go('/pages-mine/favorites/index')">
          <view class="svc__icon"><y-icon name="star" :size="44" /></view>
          <text class="svc__label">收藏模板</text>
        </view>
        <view class="svc__item" hover-class="svc__item--hover" @tap="go('/pages-mine/photos/index')">
          <view class="svc__icon"><y-icon name="album" :size="44" /></view>
          <text class="svc__label">照片管理</text>
        </view>
        <!-- #ifdef MP-WEIXIN -->
        <button class="svc__item svc__btn" hover-class="svc__item--hover" open-type="contact">
          <view class="svc__icon"><y-icon name="headset" :size="44" /></view>
          <text class="svc__label">联系客服</text>
        </button>
        <!-- #endif -->
        <!-- #ifndef MP-WEIXIN -->
        <view class="svc__item" hover-class="svc__item--hover" @tap="toast('请在微信小程序内联系客服')">
          <view class="svc__icon"><y-icon name="headset" :size="44" /></view>
          <text class="svc__label">联系客服</text>
        </view>
        <!-- #endif -->
      </view>

      <view class="card list">
        <y-list-row icon="settings" title="设置" caption="账号与隐私" @select="go('/pages-mine/settings/index')" />
      </view>

      <view class="footer">
        <text class="footer__link" @tap="openWebview(env.privacyPolicyUrl, '隐私政策')">隐私政策</text>
        <text class="footer__dot">·</text>
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
import { MODULE_NAME, type Module, type WorksSummary } from '@/types'

const user = useUserStore()
const gen = useGenerate()
const { top } = useSafeArea()
const summary = ref<WorksSummary | null>(null)
const modules: Module[] = ['idphoto', 'pro', 'portrait', 'avatar']

onShow(async () => {
  await user.ready()
  user.refreshCredits().catch(() => {})
  summary.value = await api.worksSummary().catch(() => null)
})

const total = computed(() => summary.value?.total ?? user.user?.works_count ?? 0)
const count = (m: Module) => summary.value?.by_module[m] ?? 0
const cover = (m: Module) => summary.value?.recent.find((w) => w.module === m)?.thumb_url || ''
const inviteText = computed(() => {
  const r = user.shareReward
  return r.enabled ? `邀请好友，好友首次生成后你得 ${r.per_reward} 次` : ''
})
</script>

<style lang="scss" scoped>
/* Header sits on a soft blue wash instead of a card; the credits card overlaps its lower edge. */
.head { background: linear-gradient(180deg, #DCE8FF 0%, rgba(242, 244, 248, 0) 100%); padding-bottom: 56rpx; }
.profile { display: flex; align-items: center; }
.profile__avatar { width: 128rpx; height: 128rpx; border-radius: 50%; background: $color-primary-soft; border: 6rpx solid #fff; box-sizing: border-box; box-shadow: 0 8rpx 24rpx rgba(31, 102, 224, 0.12); flex-shrink: 0; }
.profile__avatar--empty { display: flex; align-items: center; justify-content: center; }
.profile__body { flex: 1; min-width: 0; padding: 0 24rpx; }
.profile__name { font-size: 40rpx; font-weight: 700; line-height: 1.25; color: $color-text; letter-spacing: -0.3rpx; }
.profile__cap { font-size: $font-caption; color: $color-text-2; margin-top: 8rpx; line-height: 1.2; }
.profile__edit { flex-shrink: 0; height: 60rpx; line-height: 60rpx; padding: 0 26rpx; border-radius: $radius-pill; background: rgba(255, 255, 255, 0.8); color: $color-text; font-size: $font-caption; font-weight: 600; }
.profile__edit--hover { opacity: 0.7; }
.body { margin-top: -32rpx; }

.grid4 { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16rpx; }
.wk { position: relative; padding-top: 128%; border-radius: $radius-md; overflow: hidden; background: $color-surface; box-shadow: $shadow-card; }
.wk--hover { opacity: 0.88; }
.wk__img { position: absolute; top: 0; right: 0; bottom: 0; left: 0; width: 100%; height: 100%; }
.wk__scrim { position: absolute; left: 0; right: 0; bottom: 0; height: 52%; background: linear-gradient(180deg, rgba(17, 24, 39, 0) 0%, rgba(17, 24, 39, 0.66) 100%); }
.wk__empty { position: absolute; top: 0; left: 0; right: 0; bottom: 38%; display: flex; align-items: center; justify-content: center; }
.wk__text { position: absolute; left: 0; right: 0; bottom: 14rpx; text-align: center; color: $color-text; }
.wk__text--light { color: #fff; text-shadow: 0 2rpx 6rpx rgba(0, 0, 0, 0.25); }
.wk__name { font-size: 26rpx; font-weight: 600; line-height: 1.3; }
.wk__count { font-size: 22rpx; opacity: 0.85; line-height: 1.3; }
.wk__text:not(.wk__text--light) .wk__count { color: $color-text-3; opacity: 1; }

.svc { display: flex; padding: 8rpx 0; }
.svc__item { flex: 1; display: flex; flex-direction: column; align-items: center; padding: 24rpx 0 20rpx; border-radius: $radius-md; }
.svc__item--hover { background: $color-primary-tint; }
.svc__btn { background: transparent; margin: 0; line-height: normal; font-size: inherit; }
.svc__btn::after { border: 0; }
.svc__icon { width: 84rpx; height: 84rpx; border-radius: 50%; background: $color-primary-soft; display: flex; align-items: center; justify-content: center; }
.svc__label { margin-top: 12rpx; font-size: $font-caption; color: $color-text-2; line-height: 1.2; }

.list { margin-top: 24rpx; overflow: hidden; }
.footer { text-align: center; font-size: 22rpx; color: $color-text-3; margin-top: 40rpx; }
.footer__link { padding: 8rpx; }
.footer__dot { margin: 0 8rpx; }
.tab-bottom { height: calc(48rpx + env(safe-area-inset-bottom)); }
</style>
