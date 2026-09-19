<template>
  <view class="page px">
    <view v-if="flow.hasTarget" class="target card">
      <y-icon :name="flow.kind === 'idphoto' ? 'id-card' : 'image'" :size="40" />
      <view class="target__text">
        <view class="target__name">{{ flow.targetName }}</view>
        <view v-if="flow.spec" class="target__cap num">{{ mm(flow.spec.width_mm, flow.spec.height_mm) }} · {{ px(flow.spec.width_px, flow.spec.height_px) }}</view>
        <view v-else class="target__cap">{{ MODULE_NAME[flow.module || 'pro'] }}模板</view>
      </view>
    </view>

    <template v-if="!result">
      <view class="tiles">
        <view class="tile card" hover-class="tile--hover" @tap="pick('album')">
          <view class="tile__icon"><y-icon name="album" :size="56" /></view>
          <view class="tile__name">从相册选择</view>
        </view>
        <view class="tile card" hover-class="tile--hover" @tap="pick('camera')">
          <view class="tile__icon"><y-icon name="camera" :size="56" /></view>
          <view class="tile__name">拍照</view>
        </view>
      </view>

      <view class="tips card">
        <view class="tips__title">拍摄建议</view>
        <view class="tips__row"><y-icon name="check" :size="28" color="#22C55E" :stroke-width="2.4" /><text>正脸、无遮挡、光线清晰</text></view>
        <view class="tips__row"><y-icon name="check" :size="28" color="#22C55E" :stroke-width="2.4" /><text>面部占画面三分之一以上</text></view>
        <view class="tips__row"><y-icon name="close" :size="28" color="#EF4444" :stroke-width="2.4" /><text>墨镜、口罩、极端侧脸、模糊照片</text></view>
      </view>

      <view class="privacy">照片仅用于本次生成，{{ retention }} 天后自动删除，可随时在「原始照片」中删除。</view>
      <view v-if="isDev && !uploading" class="dev-sample" @tap="useSample">使用示例照片（仅开发环境）</view>
      <view v-if="uploading" class="uploading"><view class="uploading__bar" /><text>正在上传并检测照片…</text></view>
    </template>

    <template v-else>
      <y-photo-check-result :preview="result.preview" :passed="result.passed" :reasons="result.reasons" />
      <view class="actions">
        <y-primary-button v-if="result.passed" text="下一步" @press="next" />
        <y-primary-button text="重新上传" :secondary="result.passed" @press="result = null" />
      </view>
    </template>

    <y-picker-sheet v-model:visible="pickerVisible" :title="flow.module === 'idphoto' ? '选择规格' : '选择模板'" :specs="pickSpecs" :templates="pickTemplates" @pick-spec="onPickSpec" @pick-template="onPickTemplate" />
  </view>
</template>

<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import { api } from '@/api'
import { track } from '@/composables/useAnalytics'
import { env } from '@/config'
import { pickPhoto } from '@/composables/usePickPhoto'
import { compressIfNeeded, go, toast } from '@/platform'
import { useCatalogueStore } from '@/store/catalogue'
import { useFlowStore } from '@/store/flow'
import { useUserStore } from '@/store/user'
import { MODULE_NAME, type Module, type Spec, type TemplateCard } from '@/types'
import { ApiError, messageOf } from '@/utils/errors'
import { mm, px } from '@/utils/format'

const flow = useFlowStore()
const user = useUserStore()
const catalogue = useCatalogueStore()
const uploading = ref(false)
const result = ref<{ preview: string; passed: boolean; reasons: string[] } | null>(null)
const pickerVisible = ref(false)
const pickSpecs = ref<Spec[] | undefined>()
const pickTemplates = ref<TemplateCard[] | undefined>()
const retention = computed(() => user.config?.retention_days ?? 30)
const isDev = env.appEnv === 'dev'

onLoad(async (q) => {
  const m = (q?.module as Module) || flow.module || 'idphoto'
  if (!flow.module) flow.start(m)
  await user.ready()
  if (q?.reuse && flow.photo) result.value = { preview: flow.photo.preview_url, passed: true, reasons: [] }
})

async function ensurePrivacy(): Promise<boolean> {
  if (user.user?.privacy_agreed) return true
  const ok = await new Promise<boolean>((resolve) => {
    uni.showModal({
      title: '照片使用说明',
      content: `上传的照片仅用于生成图片，人脸信息属于敏感个人信息，我们会在 ${retention.value} 天后自动删除原图。继续即表示同意《隐私政策》。`,
      confirmText: '同意并继续', cancelText: '不同意',
      success: (r) => resolve(!!r.confirm), fail: () => resolve(false),
    })
  })
  if (!ok) return false
  await user.agreePrivacy().catch(() => {})
  return true
}

async function pick(source: 'album' | 'camera') {
  if (!(await ensurePrivacy())) return
  const img = await pickPhoto(source)
  if (!img) return
  const file = await compressIfNeeded(img)
  await uploadFile(file.path, source)
}

async function uploadFile(path: string, source: string) {
  track('upload_start', { module: flow.module, source })
  uploading.value = true
  try {
    const r = await api.uploadPhoto({ filePath: path, module: flow.module || 'idphoto' })
    flow.setPhoto(r.photo)
    result.value = { preview: r.photo.preview_url, passed: r.check.passed, reasons: r.check.reasons }
    track('upload_success', { photo_id: r.photo.id })
  } catch (e) {
    if (e instanceof ApiError && e.code === 'PHOTO_REJECTED') {
      result.value = { preview: path, passed: false, reasons: e.data?.reasons || [] }
      track('photo_rejected', { reasons: e.data?.reasons })
    } else toast(messageOf(e))
  } finally { uploading.value = false }
}

/** Dev shortcut: upload a bundled sample so the flow can be exercised without picking a file. */
async function useSample() {
  if (!(await ensurePrivacy())) return
  let path = '/static/mock/sample_upload.jpg'
  // #ifdef H5
  // uni.uploadFile on H5 needs a blob/object URL, not a static path.
  const blob = await (await fetch(path)).blob()
  path = URL.createObjectURL(blob)
  // #endif
  await uploadFile(path, 'sample')
}

async function next() {
  if (!flow.hasTarget) return openPicker()
  proceed()
}
function proceed() {
  if (flow.kind === 'idphoto') go('/pages/idphoto-params/index')
  else go('/pages/confirm/index')
}
async function openPicker() {
  const m = flow.module || 'idphoto'
  if (m === 'idphoto') { pickSpecs.value = await api.specs({ hot: true }); pickTemplates.value = undefined }
  else { const h = await catalogue.home(m); pickTemplates.value = h.hot_templates || []; pickSpecs.value = undefined }
  pickerVisible.value = true
}
function onPickSpec(s: Spec) { flow.setSpec(s); pickerVisible.value = false; proceed() }
async function onPickTemplate(t: TemplateCard) {
  const full = await catalogue.template(t.id)
  flow.setTemplate(full)
  pickerVisible.value = false
  proceed()
}
</script>

<style lang="scss" scoped>
.target { display: flex; align-items: center; padding: 20rpx 24rpx; margin-top: 16rpx; }
.target__text { padding-left: 16rpx; }
.target__name { font-size: $font-body-strong; font-weight: 600; }
.target__cap { font-size: $font-caption; color: $color-text-3; }
.tiles { display: flex; gap: 20rpx; margin-top: 24rpx; }
.tile { flex: 1; padding: 48rpx 0; text-align: center; }
.tile--hover { opacity: 0.85; }
.tile__icon { width: 120rpx; height: 120rpx; border-radius: 50%; background: $color-primary-soft; display: flex; align-items: center; justify-content: center; margin: 0 auto 16rpx; }
.tile__name { font-size: $font-body-strong; font-weight: 600; }
.tips { margin-top: 24rpx; padding: 24rpx; }
.tips__title { font-size: $font-body-strong; font-weight: 600; margin-bottom: 12rpx; }
.tips__row { display: flex; align-items: center; gap: 12rpx; font-size: $font-body; color: $color-text-2; line-height: 2; }
.privacy { font-size: 22rpx; color: $color-text-3; margin-top: 24rpx; line-height: 1.6; }
.dev-sample { margin-top: 24rpx; text-align: center; font-size: $font-caption; color: $color-primary; text-decoration: underline; }
.uploading { margin-top: 32rpx; text-align: center; font-size: $font-caption; color: $color-text-2; }
.uploading__bar { height: 8rpx; border-radius: 4rpx; background: $color-primary-soft; overflow: hidden; position: relative; margin-bottom: 12rpx; }
.uploading__bar::after { content: ''; position: absolute; left: -40%; width: 40%; top: 0; bottom: 0; background: $color-primary; animation: slide 1.2s infinite; }
@keyframes slide { to { left: 100%; } }
.actions { margin-top: 32rpx; display: flex; flex-direction: column; gap: 20rpx; }
</style>
