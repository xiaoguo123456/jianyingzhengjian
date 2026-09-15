<template>
  <view class="page px">
    <view class="card form">
      <view class="row">
        <text class="row__label">头像</text>
        <!-- #ifdef MP-WEIXIN -->
        <button class="avatar-btn" open-type="chooseAvatar" @chooseavatar="onChooseAvatar">
          <image class="avatar" :src="avatar" mode="aspectFill" />
        </button>
        <!-- #endif -->
        <!-- #ifndef MP-WEIXIN -->
        <image class="avatar" :src="avatar" mode="aspectFill" @tap="pickAvatar" />
        <!-- #endif -->
      </view>
      <view class="row">
        <text class="row__label">昵称</text>
        <input class="row__input" type="nickname" v-model="nickname" placeholder="请输入昵称" maxlength="20" />
      </view>
    </view>
    <view class="hint">昵称和头像仅用于展示，不会用于生成。</view>
    <y-primary-button sticky text="保存" :loading="saving" @press="save" />
  </view>
</template>

<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { api } from '@/api'
import { back, pickImage, toast } from '@/platform'
import { useUserStore } from '@/store/user'
import { messageOf } from '@/utils/errors'

const user = useUserStore()
const nickname = ref('')
const avatar = ref('')
const avatarPath = ref('')
const saving = ref(false)
onLoad(async () => { await user.ready(); nickname.value = user.user?.nickname || ''; avatar.value = user.user?.avatar_url || '' })
function onChooseAvatar(e: any) { avatar.value = e.detail.avatarUrl; avatarPath.value = e.detail.avatarUrl }
async function pickAvatar() { try { const img = await pickImage('album'); avatar.value = img.path; avatarPath.value = img.path } catch { /* cancelled */ } }
async function save() {
  saving.value = true
  try { user.user = await api.updateProfile({ nickname: nickname.value.trim(), avatar_path: avatarPath.value || undefined }); toast('已保存', 'success'); back() } catch (e) { toast(messageOf(e)) } finally { saving.value = false }
}
</script>

<style lang="scss" scoped>
.form { margin-top: 16rpx; padding: 0 24rpx; }
.row { display: flex; align-items: center; justify-content: space-between; min-height: 120rpx; border-bottom: 1px solid $color-border; }
.row:last-child { border-bottom: 0; }
.row__label { font-size: $font-body; color: $color-text; }
.row__input { flex: 1; text-align: right; font-size: $font-body; }
.avatar { width: 96rpx; height: 96rpx; border-radius: 50%; background: $color-primary-soft; }
.avatar-btn { background: transparent; padding: 0; margin: 0; line-height: normal; }
.avatar-btn::after { border: 0; }
.hint { font-size: 22rpx; color: $color-text-3; margin-top: 16rpx; }
</style>
