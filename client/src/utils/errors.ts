/* All user-facing error copy in one place (docs/API.md §1, docs/GENERATION_PIPELINE.md §6). */

export class ApiError extends Error {
  code: string
  status: number
  data?: any
  constructor(code: string, message: string, status = 400, data?: any) {
    super(message)
    this.code = code
    this.status = status
    this.data = data
  }
}

export const ERROR_COPY: Record<string, string> = {
  NETWORK: '网络不太顺畅，请稍后再试',
  UNAUTHORIZED: '登录已过期，请重试',
  NO_CREDITS: '生成次数不足',
  PHOTO_REJECTED: '这张照片可能影响生成效果，请换一张清晰正脸照片。',
  AD_SESSION_INVALID: '未完整观看视频，未获得次数',
  AD_LOAD_FAILED: '视频暂时无法加载，请稍后再试。',
  GENERATION_UNAVAILABLE: '生成服务暂时不可用，请稍后再试。',
  RATE_LIMITED: '操作太频繁，请稍后再试',
  INTERNAL: '出了点问题，请稍后再试',
  TIMEOUT: '本次生成失败，生成次数已返还，请重新尝试。',
  PROVIDER_ERROR: '本次生成失败，生成次数已返还，请重新尝试。',
  CONTENT_REJECTED: '这张照片无法生成，请换一张照片。',
  IDENTITY_MISMATCH: '生成结果与本人差异较大，请换一张正脸照片。',
  NO_FACE: '未检测到清晰人脸，请换一张照片。',
  ALBUM_DENIED: '需要相册权限才能保存图片',
  CAMERA_DENIED: '需要相机权限才能拍照',
}

export const PHOTO_REASON_COPY: Record<string, string> = {
  no_face: '未检测到人脸',
  multiple_faces: '检测到多张人脸，请上传单人照片',
  face_too_small: '人脸太小，请靠近一些',
  blurry: '照片模糊',
  too_dark: '光线太暗',
  occluded: '面部有遮挡（墨镜、口罩等）',
  low_resolution: '分辨率过低',
}

export function messageOf(err: unknown): string {
  if (err instanceof ApiError) return ERROR_COPY[err.code] || err.message || ERROR_COPY.INTERNAL
  if (err && typeof err === 'object' && 'errMsg' in (err as any)) return ERROR_COPY.NETWORK
  return ERROR_COPY.INTERNAL
}
