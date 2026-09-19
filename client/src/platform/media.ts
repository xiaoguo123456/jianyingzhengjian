export interface PickedImage { path: string; size: number }

export function pickImage(source: 'album' | 'camera'): Promise<PickedImage> {
  return new Promise((resolve, reject) => {
    uni.chooseImage({
      count: 1,
      sizeType: ['original'],
      sourceType: [source],
      success: (r) => {
        const f = (r.tempFiles as any[])[0]
        resolve({ path: r.tempFilePaths[0], size: f?.size || 0 })
      },
      fail: (e) => reject(e),
    })
  })
}

export type PickFailure = 'cancel' | 'privacy' | 'denied' | 'undeclared' | 'other'

/** Classifies a privacy / chooseImage failure. WeChat errno 112: the MP console privacy guideline does not
    declare the API; 103 / 104: the user refused the privacy popup. */
export function pickFailure(e: any): PickFailure {
  const msg = String(e?.errMsg || e?.message || '')
  const errno = Number(e?.errno)
  if (/cancel/i.test(msg)) return 'cancel'
  if (errno === 112 || /not declared/i.test(msg)) return 'undeclared'
  if (errno === 103 || errno === 104 || /privacy/i.test(msg)) return 'privacy'
  if (/auth|deny|denied/i.test(msg)) return 'denied'
  return 'other'
}

/** Compress above 4 MB where the platform supports it. */
export async function compressIfNeeded(img: PickedImage): Promise<PickedImage> {
  if (img.size <= 4 * 1024 * 1024) return img
  // #ifndef H5
  return new Promise((resolve) => {
    uni.compressImage({ src: img.path, quality: 80, success: (r) => resolve({ path: r.tempFilePath, size: img.size / 2 }), fail: () => resolve(img) })
  })
  // #endif
  // #ifdef H5
  return img
  // #endif
}
