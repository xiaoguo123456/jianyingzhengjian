/** Saves a remote image to the album. Returns 'saved' | 'opened' (H5 fallback). */
export async function saveImage(url: string): Promise<'saved' | 'opened'> {
  // #ifdef H5
  window.open(url, '_blank')
  return 'opened'
  // #endif
  // #ifndef H5
  const filePath = url.startsWith('http')
    ? await new Promise<string>((resolve, reject) => uni.downloadFile({ url, success: (r) => resolve(r.tempFilePath), fail: reject }))
    : url
  await new Promise<void>((resolve, reject) => {
    uni.saveImageToPhotosAlbum({
      filePath,
      success: () => resolve(),
      fail: (e: any) => {
        const denied = /auth|deny|denied/i.test(e?.errMsg || '')
        if (denied) {
          uni.showModal({ title: '需要相册权限', content: '请在设置中允许保存到相册', confirmText: '去设置', success: (r) => { if (r.confirm) uni.openSetting({}) } })
        }
        reject(e)
      },
    })
  })
  return 'saved'
  // #endif
}
