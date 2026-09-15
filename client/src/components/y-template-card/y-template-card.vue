<template>
  <view class="tc" hover-class="tc--hover" @tap="$emit('select', template)">
    <view class="tc__cover-wrap" :class="{ 'tc__cover-wrap--square': square }">
      <image class="tc__cover" :src="template.cover_url" mode="aspectFill" lazy-load />
      <view v-if="showTag && template.tags.length" class="tc__tag"><y-tag :text="template.tags[0] === 'NEW' ? '上新' : template.tags[0]" variant="hot" /></view>
      <view v-if="template.is_favorited" class="tc__fav"><y-icon name="heart-fill" :size="30" color="#EF4444" /></view>
    </view>
    <view class="tc__name ellipsis">{{ template.name }}</view>
  </view>
</template>

<script setup lang="ts">
import type { TemplateCard } from '@/types'
withDefaults(defineProps<{ template: TemplateCard; square?: boolean; showTag?: boolean }>(), { showTag: false })
defineEmits<{ (e: 'select', t: TemplateCard): void }>()
</script>

<style lang="scss" scoped>
/* 照片与名称组成完整入口，不额外叠加卡片底板。 */
.tc { min-width: 0; }
.tc--hover { opacity: 0.88; }
.tc__cover-wrap { position: relative; width: 100%; padding-top: 133.33%; border-radius: $radius-md; overflow: hidden; background: $color-primary-soft; }
.tc__cover-wrap--square { padding-top: 100%; }
.tc__cover { position: absolute; top: 0; right: 0; bottom: 0; left: 0; width: 100%; height: 100%; }
.tc__tag { position: absolute; top: 12rpx; left: 12rpx; }
.tc__fav { position: absolute; top: 12rpx; right: 12rpx; }
.tc__name { padding: 14rpx 0 4rpx; font-size: $font-body; font-weight: 500; color: $color-text; }
</style>
