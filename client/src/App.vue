<script setup lang="ts">
import { onLaunch, onShow } from '@dcloudio/uni-app'
import { useUserStore } from '@/store/user'
import { handleLaunchShare } from '@/composables/useShare'

onLaunch(() => {
  const user = useUserStore()
  user.bootstrap()
  try { handleLaunchShare(uni.getLaunchOptionsSync()) } catch { /* not available on this platform */ }
})
onShow(() => {
  try { handleLaunchShare((uni as any).getEnterOptionsSync ? (uni as any).getEnterOptionsSync() : null) } catch { /* ignore */ }
})
</script>

<style lang="scss">
/* 全局主题，与 uni.scss 中的变量保持一致。 */
page {
  --color-primary: #2864DC;
  --color-primary-strong: #1F66E0;
  --color-primary-soft: #EDF2FC;
  --color-primary-tint: #F4F6FB;
  --gradient-primary: #2864DC;
  --gradient-banner: #EDF0F5;
  --color-bg: #F7F8FA;
  --color-surface: #FFFFFF;
  --color-border: #E7E9ED;
  --color-text: #111827;
  --color-text-2: #4B5563;
  --color-text-3: #697386;
  --color-success: #22C55E;
  --color-warning: #F59E0B;
  --color-danger: #EF4444;
  --color-mask: rgba(17, 24, 39, 0.45);
  --shadow-card: none;
  /* H5 原生导航层级为 998，业务弹层须高于导航。 */
  --z-overlay: 1100;
  --shadow-sheet: 0 -8rpx 32rpx rgba(17, 24, 39, 0.08);

  background-color: var(--color-bg);
  color: var(--color-text);
  font-family: -apple-system, BlinkMacSystemFont, 'PingFang SC', 'Helvetica Neue', 'Microsoft YaHei', sans-serif;
  font-size: 28rpx;
  line-height: 1.5;
}

/* 共享布局工具类 */
.page { min-height: 100vh; box-sizing: border-box; }
.px { padding-left: 32rpx; padding-right: 32rpx; }
.card {
  background: var(--color-surface);
  border-radius: 24rpx;
  box-shadow: var(--shadow-card);
}
.row { display: flex; align-items: center; }
.between { display: flex; align-items: center; justify-content: space-between; }
.t-h2 { font-size: 34rpx; font-weight: 600; line-height: 1.3; color: var(--color-text); }
.t-strong { font-size: 30rpx; font-weight: 600; line-height: 1.4; color: var(--color-text); }
.t-body { font-size: 28rpx; color: var(--color-text-2); }
.t-cap { font-size: 24rpx; color: var(--color-text-3); line-height: 1.4; }
.num { font-variant-numeric: tabular-nums; }
.ellipsis { overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.section { margin-top: 40rpx; }
.template-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 28rpx 20rpx; }
.tab-bottom { height: calc(48rpx + env(safe-area-inset-bottom)); }
@media (prefers-reduced-motion: reduce) {
  view, image, button { animation-duration: 0.01ms !important; transition-duration: 0.01ms !important; animation-iteration-count: 1 !important; }
}
.bottom-space { height: calc(160rpx + env(safe-area-inset-bottom)); }
</style>
