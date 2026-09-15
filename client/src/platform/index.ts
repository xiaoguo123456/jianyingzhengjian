/* Platform layer entry (docs/FRONTEND_ARCHITECTURE.md §5). Business code imports from here only.
   Every import is present for type-checking; conditional compilation keeps only one set at runtime. */
import type { AdsPlatform, AuthPlatform, PrivacyPlatform, SharePlatform, SubscribePlatform } from './types'
// #ifdef MP-WEIXIN
import { auth as mpAuth } from './auth.mp'
import { ads as mpAds } from './ads.mp'
import { share as mpShare } from './share.mp'
import { privacy as mpPrivacy } from './privacy.mp'
import { subscribe as mpSubscribe } from './subscribe.mp'
// #endif
// #ifdef H5
import { auth as h5Auth } from './auth.h5'
import { ads as simAds } from './ads.sim'
import { share as h5Share } from './share.h5'
import { privacy as h5Privacy } from './privacy.h5'
import { subscribe as noopSubscribe } from './subscribe.noop'
// #endif
// #ifdef APP-PLUS
import { auth as appAuth } from './auth.app'
import { ads as appAds } from './ads.sim'
import { share as appShare } from './share.app'
import { privacy as appPrivacy } from './privacy.h5'
import { subscribe as appSubscribe } from './subscribe.noop'
// #endif

let auth!: AuthPlatform
let ads!: AdsPlatform
let share!: SharePlatform
let privacy!: PrivacyPlatform
let subscribe!: SubscribePlatform

// #ifdef MP-WEIXIN
auth = mpAuth; ads = mpAds; share = mpShare; privacy = mpPrivacy; subscribe = mpSubscribe
// #endif
// #ifdef H5
auth = h5Auth; ads = simAds; share = h5Share; privacy = h5Privacy; subscribe = noopSubscribe
// #endif
// #ifdef APP-PLUS
auth = appAuth; ads = appAds; share = appShare; privacy = appPrivacy; subscribe = appSubscribe
// #endif

export { auth, ads, share, privacy, subscribe }
export * from './media'
export * from './album'
export * from './nav'
export type { AdsPlatform, AuthPlatform, PrivacyPlatform, SharePlatform, SubscribePlatform } from './types'
