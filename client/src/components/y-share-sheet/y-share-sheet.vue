<template>
  <y-sheet :visible="visible" title="分享" @update:visible="(v: boolean) => $emit('update:visible', v)">
    <view v-if="allowWork" class="ss__toggle">
      <view class="ss__opt" :class="{ 'ss__opt--on': mode === 'template' }" @tap="setMode('template')">仅分享模板</view>
      <view class="ss__opt" :class="{ 'ss__opt--on': mode === 'work' }" @tap="setMode('work')">分享作品</view>
    </view>
    <view class="ss__preview card">
      <view class="ss__img-wrap">
        <image v-if="share" class="ss__img" :src="share.image_url" mode="aspectFill" />
        <view v-else class="ss__img ss__img--loading" />
        <y-ai-label v-if="share && share.type !== 'template' && share.type !== 'tab'" />
      </view>
      <view class="ss__title">{{ share?.title || '准备分享内容…' }}</view>
    </view>
    <view class="ss__actions">
      <!-- #ifdef MP-WEIXIN -->
      <button class="ss__action ss__native" open-type="share" :disabled="!share">
        <view class="ss__circle"><y-icon name="chat" :size="44" /></view>
        <view class="ss__label">发送给朋友</view>
      </button>
      <!-- #endif -->
      <!-- #ifndef MP-WEIXIN -->
      <view class="ss__action" :class="{ 'ss__action--off': !share }" @tap="$emit('chat')">
        <view class="ss__circle"><y-icon name="link" :size="44" /></view>
        <view class="ss__label">复制链接</view>
      </view>
      <!-- #endif -->
      <view v-if="timeline" class="ss__action" :class="{ 'ss__action--off': !share }" @tap="$emit('timeline')">
        <view class="ss__circle"><y-icon name="moments" :size="44" /></view>
        <view class="ss__label">分享到朋友圈</view>
      </view>
      <view class="ss__action" :class="{ 'ss__action--off': !share }" @tap="$emit('poster')">
        <view class="ss__circle"><y-icon name="qr" :size="44" /></view>
        <view class="ss__label">保存海报</view>
      </view>
    </view>
    <view v-if="rewardText" class="ss__reward">{{ rewardText }}</view>
  </y-sheet>
</template>

<script setup lang="ts">
import type { Share } from '@/types'

defineProps<{ visible: boolean; share: Share | null; allowWork?: boolean; mode: 'template' | 'work'; timeline?: boolean; rewardText?: string }>()
const emit = defineEmits<{ (e: 'update:visible', v: boolean): void; (e: 'update:mode', m: 'template' | 'work'): void; (e: 'chat'): void; (e: 'timeline'): void; (e: 'poster'): void }>()
const setMode = (m: 'template' | 'work') => emit('update:mode', m)
</script>

<style lang="scss" scoped>
.ss__toggle { display: flex; background: $color-bg; border-radius: $radius-pill; padding: 6rpx; margin-bottom: 24rpx; }
.ss__opt { flex: 1; text-align: center; height: 64rpx; line-height: 64rpx; border-radius: $radius-pill; font-size: $font-body; color: $color-text-2; }
.ss__opt--on { background: #fff; color: $color-primary; font-weight: 600; box-shadow: $shadow-card; }
.ss__preview { display: flex; align-items: center; padding: 20rpx; border: 2rpx solid $color-border; box-shadow: none; }
.ss__img-wrap { position: relative; width: 200rpx; height: 160rpx; border-radius: $radius-sm; overflow: hidden; background: $color-primary-soft; flex-shrink: 0; }
.ss__img { width: 100%; height: 100%; }
.ss__img--loading { background: $color-border; }
.ss__title { flex: 1; padding-left: 20rpx; font-size: $font-body; color: $color-text; }
.ss__actions { display: flex; justify-content: space-around; margin-top: 32rpx; }
.ss__action { text-align: center; width: 180rpx; }
.ss__action--off { opacity: 0.4; }
.ss__native { background: transparent; padding: 0; margin: 0; line-height: normal; font-size: inherit; border: 0; }
.ss__native::after { border: 0; }
.ss__circle { width: 100rpx; height: 100rpx; border-radius: 50%; background: $color-primary-soft; display: flex; align-items: center; justify-content: center; margin: 0 auto 12rpx; }
.ss__label { font-size: $font-caption; color: $color-text-2; }
.ss__reward { margin-top: 28rpx; text-align: center; font-size: 22rpx; color: $color-text-3; }
</style>
