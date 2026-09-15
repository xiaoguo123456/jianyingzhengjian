<template>
  <view class="page px">
    <view class="mt"><y-credit-bar :credits="user.credits" @watch="gen.adVisible.value = true" /></view>

    <view class="card rules">
      <view class="rules__title">规则</view>
      <view class="rules__line">· 每天免费赠送 {{ user.adsEnabled ? 1 : 3 }} 次，当天有效</view>
      <view class="rules__line" v-if="user.adsEnabled">· 完整观看一段视频得 1 次，每天最多 {{ user.credits?.ad_reward_daily_cap ?? 10 }} 次，长期有效</view>
      <view class="rules__line">· 保持原服装且不美化的证件照免费，换背景免费</view>
      <view class="rules__line">· 生成失败自动返还</view>
    </view>

    <view v-if="rewards?.enabled" class="card invite">
      <view class="invite__head">
        <view class="invite__icon"><y-icon name="users" :size="44" /></view>
        <view class="invite__text">
          <view class="invite__title">邀请好友</view>
          <view class="invite__cap">好友通过你的分享完成首次生成，你获得 {{ rewards.per_reward }} 次（每日最多 {{ rewards.daily_cap }} 次）</view>
        </view>
      </view>
      <view class="invite__stats">
        <view class="stat"><view class="stat__num num">{{ rewards.earned_today }}</view><view class="stat__label">今日获得</view></view>
        <view class="stat"><view class="stat__num num">{{ rewards.earned_total }}</view><view class="stat__label">累计获得</view></view>
      </view>
      <y-primary-button text="邀请好友" icon="share" @press="invite" />
    </view>

    <y-ad-sheet v-model:visible="gen.adVisible.value" :busy="gen.adBusy.value" @watch="gen.watchAd" @cancel="gen.cancelAd" />
    <y-share-sheet v-model:visible="shareVisible" :share="pageShare.current.value" mode="template" :timeline="sharePlatform.timeline" @chat="pageShare.shareDirect('chat')" @timeline="pageShare.shareDirect('timeline')" @poster="toast('请在作品页生成海报')" />
  </view>
</template>

<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { api } from '@/api'
import { useGenerate } from '@/composables/useGenerate'
import { usePageShare } from '@/composables/useShare'
import { share as sharePlatform, toast } from '@/platform'
import { useUserStore } from '@/store/user'
import type { ShareRewards } from '@/types'

const user = useUserStore()
const gen = useGenerate()
const rewards = ref<ShareRewards | null>(null)
const shareVisible = ref(false)
const pageShare = usePageShare(() => ({ type: 'tab', module: 'portrait', surface: 'credits' }))
onShow(async () => { await user.ready(); user.refreshCredits().catch(() => {}); rewards.value = await api.shareRewards().catch(() => null) })
async function invite() { shareVisible.value = true; if (!pageShare.current.value) await pageShare.prepare() }
</script>

<style lang="scss" scoped>
.mt { margin-top: 16rpx; }
.rules { margin-top: 20rpx; padding: 24rpx; }
.rules__title { font-size: $font-body-strong; font-weight: 600; margin-bottom: 8rpx; }
.rules__line { font-size: $font-caption; color: $color-text-2; line-height: 1.9; }
.invite { margin-top: 20rpx; padding: 24rpx; }
.invite__head { display: flex; align-items: center; }
.invite__icon { width: 88rpx; height: 88rpx; border-radius: 50%; background: $color-primary-soft; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.invite__text { padding-left: 20rpx; }
.invite__title { font-size: $font-body-strong; font-weight: 600; }
.invite__cap { font-size: 22rpx; color: $color-text-3; margin-top: 4rpx; }
.invite__stats { display: flex; margin: 24rpx 0; }
.stat { flex: 1; text-align: center; }
.stat__num { font-size: 44rpx; font-weight: 700; color: $color-primary; }
.stat__label { font-size: 22rpx; color: $color-text-3; }
</style>
