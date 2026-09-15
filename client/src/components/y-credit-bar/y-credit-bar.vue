<template>
  <view class="cc">
    <view class="cc__main" :hover-class="tappable ? 'cc--hover' : 'none'" @tap="$emit('open')">
      <view class="cc__info">
        <view class="cc__label">生成次数</view>
        <view class="cc__amount">
          <text class="cc__num num">{{ credits?.total ?? 0 }}</text>
          <text class="cc__unit">次</text>
        </view>
        <view class="cc__cap">{{ caption }}</view>
      </view>
      <view v-if="showAd" class="cc__btn" hover-class="cc__btn--hover" hover-stop-propagation @tap.stop="$emit('watch')">
        <y-icon name="play" :size="26" color="#FFFFFF" />
        <text>看视频 +1 次</text>
      </view>
      <y-icon v-else-if="tappable" name="chevron-right" :size="36" color="#C0C8D4" />
    </view>
    <view v-if="inviteText" class="cc__invite" hover-class="cc--hover" @tap="$emit('invite')">
      <view class="cc__invite-icon"><y-icon name="gift" :size="32" /></view>
      <text class="cc__invite-text">{{ inviteText }}</text>
      <y-icon name="chevron-right" :size="32" color="#C0C8D4" />
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Credits } from '@/types'

const props = withDefaults(defineProps<{ credits: Credits | null; tappable?: boolean; inviteText?: string }>(), { tappable: false, inviteText: '' })
defineEmits<{ (e: 'watch'): void; (e: 'open'): void; (e: 'invite'): void }>()
const showAd = computed(() => !!props.credits?.ads_enabled && (props.credits?.ad_rewards_today ?? 0) < (props.credits?.ad_reward_daily_cap ?? 0))
const caption = computed(() => {
  const c = props.credits
  if (!c) return ''
  return c.ads_enabled ? `今日免费 ${c.daily_free_remaining} · 视频获得 ${c.bonus_credits}` : `今日免费 ${c.daily_free_remaining} 次`
})
</script>

<style lang="scss" scoped>
.cc { background: $color-surface; border-radius: $radius-lg; box-shadow: 0 8rpx 32rpx rgba(31, 102, 224, 0.08); overflow: hidden; }
.cc--hover { background: $color-primary-tint; }
.cc__main { display: flex; align-items: center; padding: 28rpx 28rpx 26rpx; }
.cc__info { flex: 1; min-width: 0; }
.cc__label { font-size: $font-caption; color: $color-text-3; line-height: 1.2; }
/* number and unit share a baseline */
.cc__amount { display: flex; align-items: baseline; gap: 6rpx; margin-top: 6rpx; }
.cc__num { font-size: 64rpx; font-weight: 700; line-height: 1; color: $color-text; letter-spacing: -1rpx; }
.cc__unit { font-size: $font-body; font-weight: 600; color: $color-text-2; }
.cc__cap { font-size: 22rpx; color: $color-text-3; margin-top: 10rpx; line-height: 1.2; }
.cc__btn {
  display: flex; align-items: center; gap: 8rpx; flex-shrink: 0; height: 72rpx; padding: 0 28rpx;
  border-radius: $radius-pill; background: $gradient-primary; color: #fff; font-size: 26rpx; font-weight: 600;
  box-shadow: 0 8rpx 20rpx rgba(58, 141, 255, 0.28);
}
.cc__btn--hover { opacity: 0.85; }
.cc__invite { display: flex; align-items: center; gap: 14rpx; height: 88rpx; padding: 0 28rpx; border-top: 1px solid $color-border; }
.cc__invite-icon { width: 48rpx; height: 48rpx; border-radius: 50%; background: $color-primary-soft; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.cc__invite-text { flex: 1; min-width: 0; font-size: 26rpx; color: $color-text-2; line-height: 1; }
</style>
