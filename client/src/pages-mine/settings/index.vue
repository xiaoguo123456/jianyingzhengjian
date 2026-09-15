<template>
  <view class="page px">
    <view class="card list">
      <y-list-row icon="shield" title="隐私政策" @select="openWebview(env.privacyPolicyUrl, '隐私政策')" />
      <y-list-row icon="file-text" title="用户协议" @select="openWebview(env.userAgreementUrl, '用户协议')" />
      <y-list-row icon="info" title="关于" :value="`v${env.clientVersion}`" :chevron="false" />
      <y-list-row icon="trash" title="注销账号" caption="删除照片、作品和账号信息" @select="deleteAccount" />
    </view>

    <view v-if="env.useMock" class="card dev">
      <view class="dev__title">开发调试（仅 mock 模式可见）</view>
      <view class="dev__row"><text>模拟生成失败</text><switch :checked="mockState.simulateFailure" color="#2F7BF6" @change="(e: any) => (mockState.simulateFailure = e.detail.value)" /></view>
      <view class="dev__row"><text>下一张照片检测不通过</text><switch :checked="mockState.rejectNextPhoto" color="#2F7BF6" @change="(e: any) => (mockState.rejectNextPhoto = e.detail.value)" /></view>
      <view class="dev__row"><text>广告已开通</text><switch :checked="mockState.adsEnabled" color="#2F7BF6" @change="toggleAds" /></view>
      <view class="dev__row"><text>可用次数 {{ user.total }}</text><view class="dev__btn" @tap="resetCredits">清零</view><view class="dev__btn" @tap="addCredits">+3</view></view>
    </view>
    <view class="bottom-space" />
  </view>
</template>

<script setup lang="ts">
import { mockState } from '@/api/mock'
import { env } from '@/config'
import { openWebview, toast } from '@/platform'
import { useUserStore } from '@/store/user'

const user = useUserStore()
function deleteAccount() {
  uni.showModal({ title: '注销账号', content: '将删除你的照片、作品和账号信息，且无法恢复。', confirmColor: '#EF4444', success: (r) => { if (r.confirm) toast('已提交注销申请') } })
}
async function toggleAds(e: any) { mockState.adsEnabled = e.detail.value; await user.loadMe() }
async function resetCredits() { mockState.daily = 0; mockState.bonus = 0; await user.refreshCredits() }
async function addCredits() { mockState.bonus += 3; await user.refreshCredits() }
</script>

<style lang="scss" scoped>
.list { margin-top: 16rpx; overflow: hidden; }
.dev { margin-top: 24rpx; padding: 24rpx; border: 2rpx dashed $color-warning; box-shadow: none; }
.dev__title { font-size: $font-caption; color: $color-warning; font-weight: 600; margin-bottom: 8rpx; }
.dev__row { display: flex; align-items: center; justify-content: space-between; gap: 12rpx; font-size: $font-body; padding: 12rpx 0; }
.dev__btn { padding: 0 20rpx; height: 52rpx; line-height: 52rpx; border-radius: $radius-pill; background: $color-primary-soft; color: $color-primary; font-size: 22rpx; }
</style>
