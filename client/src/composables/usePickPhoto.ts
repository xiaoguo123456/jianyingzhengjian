import { env } from '@/config'
import { pickFailure, pickImage, privacy, toast, type PickedImage } from '@/platform'
import { ERROR_COPY } from '@/utils/errors'

/** Asks for privacy authorisation, then opens the album or camera. Every failure except a cancel is explained to
    the user instead of failing silently. Resolves null when nothing was picked. */
export async function pickPhoto(source: 'album' | 'camera'): Promise<PickedImage | null> {
  try {
    await privacy.ensureAuthorized()
    return await pickImage(source)
  } catch (e: any) {
    explain(e, source)
    return null
  }
}

function explain(e: any, source: 'album' | 'camera') {
  const kind = pickFailure(e)
  if (kind === 'cancel') return
  console.error('[pickPhoto]', source, kind, e)
  if (kind === 'denied') {
    uni.showModal({
      title: source === 'camera' ? ERROR_COPY.CAMERA_DENIED : ERROR_COPY.ALBUM_PICK_DENIED,
      content: '请在设置中开启权限后重试', confirmText: '去设置',
      success: (r) => { if (r.confirm) uni.openSetting({}) },
    })
    return
  }
  if (kind === 'privacy') return toast(ERROR_COPY.PRIVACY_REFUSED)
  if (env.appEnv === 'prod') return toast(ERROR_COPY.PICK_FAILED)
  // Test builds show the platform error so configuration problems are visible.
  const detail = String(e?.errMsg || e?.message || e)
  uni.showModal({
    title: ERROR_COPY.PICK_FAILED,
    content: kind === 'undeclared' ? `${ERROR_COPY.PICK_UNDECLARED_DEV}\n${detail}` : detail,
    showCancel: false,
  })
}
