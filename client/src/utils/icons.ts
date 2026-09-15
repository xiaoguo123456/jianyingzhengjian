/* Line icon set (24×24 viewBox, stroke 1.75, round caps) rendered as SVG data URIs.
   docs/DESIGN_SYSTEM.md §2.5. Keep every path ASCII-only: the base64 encoder below is ASCII-only. */

export const ICON_PATHS: Record<string, string> = {
  'id-card': '<rect x="3" y="5" width="18" height="14" rx="3"/><circle cx="8.5" cy="11" r="2"/><path d="M13 9h5M13 13h5M5.5 16.5c.6-1.4 1.7-2 3-2s2.4.6 3 2"/>',
  briefcase: '<rect x="3" y="7" width="18" height="13" rx="3"/><path d="M9 7V5a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2v2M3 12h18"/>',
  image: '<rect x="3" y="4" width="18" height="16" rx="3"/><circle cx="8.5" cy="9" r="1.5"/><path d="M21 15l-5-5-8 8"/>',
  user: '<circle cx="12" cy="8" r="4"/><path d="M4 21c0-4 3.6-6.5 8-6.5s8 2.5 8 6.5"/>',
  'user-square': '<rect x="3" y="3" width="18" height="18" rx="4"/><circle cx="12" cy="10" r="3"/><path d="M6.5 19c.8-2.4 2.9-3.8 5.5-3.8s4.7 1.4 5.5 3.8"/>',
  users: '<circle cx="9" cy="8" r="3"/><path d="M3 20c0-3.5 2.7-5.5 6-5.5s6 2 6 5.5"/><circle cx="17" cy="9" r="2.5"/><path d="M21 19c0-2.5-1.7-4-4-4.3"/>',
  crop: '<path d="M6 2v14a2 2 0 0 0 2 2h14M18 22V8a2 2 0 0 0-2-2H2"/>',
  cutout: '<rect x="3" y="3" width="18" height="18" rx="4" stroke-dasharray="3 3"/><circle cx="12" cy="10" r="3"/><path d="M7 19c1-2.5 2.8-3.7 5-3.7s4 1.2 5 3.7"/>',
  shirt: '<path d="M8 3l4 2 4-2 4 3-2 4-2-1v12H8V9l-2 1-2-4z"/>',
  hd: '<rect x="3" y="5" width="18" height="14" rx="3"/><path d="M7 9v6M7 12h3M10 9v6M14 9h2.5a3 3 0 0 1 0 6H14z"/>',
  'chevron-right': '<path d="M9 6l6 6-6 6"/>',
  'chevron-left': '<path d="M15 6l-6 6 6 6"/>',
  'chevron-down': '<path d="M6 9l6 6 6-6"/>',
  upload: '<path d="M12 16V4M6 10l6-6 6 6M4 20h16"/>',
  download: '<path d="M12 4v12M6 10l6 6 6-6M4 20h16"/>',
  play: '<path d="M8 5.5v13l11-6.5z" fill="CURRENT"/>',
  gift: '<rect x="3" y="8" width="18" height="5" rx="1"/><path d="M5 13v8h14v-8M12 8v13M12 8c-2-3-5-3.5-5-1.5S9.5 8 12 8c2.5 0 5-1 5-3s-3-1.5-5 1.5"/>',
  star: '<path d="M12 3l2.7 5.6 6.1.9-4.4 4.3 1 6.1L12 17l-5.4 2.9 1-6.1L3.2 9.5l6.1-.9z"/>',
  heart: '<path d="M12 20s-7-4.5-7-10a4 4 0 0 1 7-2.5A4 4 0 0 1 19 10c0 5.5-7 10-7 10z"/>',
  'heart-fill': '<path d="M12 20s-7-4.5-7-10a4 4 0 0 1 7-2.5A4 4 0 0 1 19 10c0 5.5-7 10-7 10z" fill="CURRENT"/>',
  'file-text': '<path d="M6 3h8l4 4v14H6zM14 3v4h4M9 12h6M9 16h6"/>',
  headset: '<path d="M4 13a8 8 0 0 1 16 0"/><rect x="3" y="13" width="4" height="6" rx="1.5"/><rect x="17" y="13" width="4" height="6" rx="1.5"/>',
  settings: '<circle cx="12" cy="12" r="3"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>',
  shield: '<path d="M12 3l7 3v5c0 5-3.5 8-7 10-3.5-2-7-5-7-10V6zM9 12l2 2 4-4"/>',
  camera: '<path d="M4 8h3l2-3h6l2 3h3v11H4z"/><circle cx="12" cy="13" r="3"/>',
  album: '<rect x="3" y="5" width="18" height="14" rx="3"/><circle cx="8" cy="10" r="1.5"/><path d="M21 16l-5-5-7 7"/>',
  check: '<path d="M5 12l4 4L19 7"/>',
  close: '<path d="M6 6l12 12M18 6L6 18"/>',
  refresh: '<path d="M20 12a8 8 0 1 1-2.3-5.7M20 4v4h-4"/>',
  palette: '<path d="M12 3a9 9 0 0 0 0 18c1 0 1.5-.7 1.5-1.5s-.6-1.2-1-1.6c-.5-.6-.3-1.4.6-1.4H15a6 6 0 0 0 6-6c0-4-4-7.5-9-7.5z"/><circle cx="7.5" cy="12" r="1"/><circle cx="10" cy="8" r="1"/><circle cx="14.5" cy="8" r="1"/>',
  scissors: '<circle cx="6" cy="6" r="2.5"/><circle cx="6" cy="18" r="2.5"/><path d="M8 7.5L20 18M8 16.5L20 6"/>',
  sparkles: '<path d="M12 3l1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8zM19 16l.8 2.2L22 19l-2.2.8L19 22l-.8-2.2L16 19l2.2-.8z"/>',
  wand: '<path d="M15 4l5 5-11 11H4v-5zM13 6l5 5M6 3v2M3 6h2M19 15v2M18 20h2"/>',
  share: '<path d="M4 12v7a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-7M12 15V3M8 7l4-4 4 4"/>',
  layers: '<path d="M12 3l9 5-9 5-9-5zM3 13l9 5 9-5M3 17l9 5 9-5"/>',
  sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>',
  cake: '<path d="M4 20h16M5 20v-7h14v7M5 13a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2M8 11V8M12 11V7M16 11V8"/>',
  building: '<rect x="5" y="3" width="14" height="18" rx="1"/><path d="M9 7h2M13 7h2M9 11h2M13 11h2M9 15h2M13 15h2M10 21v-3h4v3"/>',
  monitor: '<rect x="3" y="4" width="18" height="12" rx="2"/><path d="M8 20h8M12 16v4"/>',
  presentation: '<path d="M4 4h16v10H4zM12 14v3M8 20l4-3 4 3"/>',
  leaf: '<path d="M5 19c0-8 5-13 14-13 0 9-5 14-13 14M5 19l7-7"/>',
  crown: '<path d="M4 18h16l1-10-5 4-4-6-4 6-5-4z"/>',
  'message-circle': '<path d="M20 12a8 8 0 0 1-11.6 7.2L4 20l.9-3.9A8 8 0 1 1 20 12z"/>',
  paint: '<path d="M14 3l7 7-9 9H5v-7zM4 20l4-4"/>',
  clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
  trash: '<path d="M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3"/>',
  edit: '<path d="M4 20h4l10-10-4-4L4 16zM13 7l4 4"/>',
  info: '<circle cx="12" cy="12" r="9"/><path d="M12 8h.01M12 12v4"/>',
  alert: '<path d="M12 3l10 18H2zM12 10v4M12 18h.01"/>',
  chat: '<path d="M4 5h16v11H9l-5 4z"/>',
  moments: '<circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="3"/><path d="M12 3v6M21 12h-6M12 21v-6M3 12h6"/>',
  qr: '<rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><path d="M14 14h3v3h-3zM20 14v3M17 20h3M14 20h1"/>',
  tie: '<path d="M9 3h6l-1 4 2 10-4 4-4-4 2-10z"/>',
  'user-check': '<circle cx="10" cy="8" r="4"/><path d="M2 21c0-4 3.6-6.5 8-6.5 1.2 0 2.3.2 3.3.5M16 19l2 2 4-4"/>',
  hourglass: '<path d="M6 3h12M6 21h12M8 3v3l4 6 4-6V3M8 21v-3l4-6 4 6v3"/>',
  ruler: '<path d="M3 17L17 3l4 4L7 21zM8 16l1.5-1.5M11 13l1.5-1.5M14 10l1.5-1.5"/>',
  link: '<path d="M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  'photo-plus': '<rect x="3" y="4" width="18" height="16" rx="3"/><path d="M12 8v6M9 11h6"/>',
}

const B64 = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/'
function b64(str: string): string {
  let out = ''
  for (let i = 0; i < str.length; i += 3) {
    const a = str.charCodeAt(i)
    const b = i + 1 < str.length ? str.charCodeAt(i + 1) : NaN
    const c = i + 2 < str.length ? str.charCodeAt(i + 2) : NaN
    const n = (a << 16) | ((isNaN(b) ? 0 : b) << 8) | (isNaN(c) ? 0 : c)
    out += B64[(n >> 18) & 63] + B64[(n >> 12) & 63]
    out += isNaN(b) ? '=' : B64[(n >> 6) & 63]
    out += isNaN(c) ? '=' : B64[n & 63]
  }
  return out
}

const cache = new Map<string, string>()

/** Returns a data URI for the named icon in the given colour. */
export function iconSrc(name: string, color = '#2F7BF6', strokeWidth = 1.75): string {
  const key = `${name}|${color}|${strokeWidth}`
  const hit = cache.get(key)
  if (hit) return hit
  const body = (ICON_PATHS[name] || ICON_PATHS.info).replace(/CURRENT/g, color)
  const svg =
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${color}" ` +
    `stroke-width="${strokeWidth}" stroke-linecap="round" stroke-linejoin="round">${body}</svg>`
  const uri = `data:image/svg+xml;base64,${b64(svg)}`
  cache.set(key, uri)
  return uri
}
