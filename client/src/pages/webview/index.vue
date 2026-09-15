<template>
  <web-view v-if="url" :src="url" />
</template>

<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
const url = ref('')
onLoad((q) => {
  url.value = decodeURIComponent(q?.url || '')
  if (q?.title) uni.setNavigationBarTitle({ title: decodeURIComponent(q.title) })
})
</script>
