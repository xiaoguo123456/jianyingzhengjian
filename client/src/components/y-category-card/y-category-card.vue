<template>
  <view class="cat" :class="{ 'cat--icon': !cover }" hover-class="cat--hover" @tap="$emit('tap')">
    <template v-if="cover">
      <image class="cat__cover" :src="cover" mode="aspectFill" lazy-load />
      <view class="cat__scrim" />
      <view class="cat__name cat__name--overlay">{{ name }}</view>
    </template>
    <template v-else>
      <view class="cat__icon"><y-icon :name="icon || 'sparkles'" :size="44" /></view>
      <view class="cat__name">{{ name }}</view>
      <view v-if="caption" class="cat__caption">{{ caption }}</view>
    </template>
  </view>
</template>

<script setup lang="ts">
/* Photo tile with the name on a bottom scrim; icon variant only when no cover exists. */
defineProps<{ name: string; icon?: string; cover?: string; caption?: string }>()
defineEmits<{ (e: 'tap'): void }>()
</script>

<style lang="scss" scoped>
.cat { position: relative; width: 100%; padding-top: 128%; border-radius: $radius-md; overflow: hidden; background: $color-primary-soft; }
.cat--hover { opacity: 0.88; }
.cat__cover { position: absolute; top: 0; right: 0; bottom: 0; left: 0; width: 100%; height: 100%; }
.cat__scrim { position: absolute; left: 0; right: 0; bottom: 0; height: 46%; background: linear-gradient(180deg, rgba(17, 24, 39, 0) 0%, rgba(17, 24, 39, 0.62) 100%); }
.cat__name { font-size: $font-body; font-weight: 600; color: $color-text; }
.cat__name--overlay { position: absolute; left: 0; right: 0; bottom: 14rpx; text-align: center; color: #fff; text-shadow: 0 2rpx 6rpx rgba(0, 0, 0, 0.25); }
.cat--icon { padding-top: 0; background: $color-surface; box-shadow: $shadow-card; padding: 28rpx 8rpx 24rpx; text-align: center; }
.cat__icon { width: 88rpx; height: 88rpx; border-radius: 50%; background: $color-primary-soft; display: flex; align-items: center; justify-content: center; margin: 0 auto 14rpx; }
.cat__caption { font-size: 22rpx; color: $color-text-3; margin-top: 4rpx; }
</style>
