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
/* CSS custom properties — docs/DESIGN_SYSTEM.md §2 */
page {
  --color-primary: #2F7BF6;
  --color-primary-strong: #1F66E0;
  --color-primary-soft: #E8F1FF;
  --color-primary-tint: #F2F7FF;
  --gradient-primary: linear-gradient(90deg, #3A8DFF 0%, #35C3E6 100%);
  --gradient-banner: linear-gradient(135deg, #DCEBFF 0%, #EAF3FF 60%, #FFFFFF 100%);
  --color-bg: #F2F4F8;
  --color-surface: #FFFFFF;
  --color-border: #E6ECF5;
  --color-text: #111827;
  --color-text-2: #4B5563;
  --color-text-3: #8A94A6;
  --color-success: #22C55E;
  --color-warning: #F59E0B;
  --color-danger: #EF4444;
  --color-mask: rgba(17, 24, 39, 0.45);
  --shadow-card: 0 4rpx 16rpx rgba(47, 123, 246, 0.06);
  --shadow-sheet: 0 -8rpx 32rpx rgba(17, 24, 39, 0.08);

  background-color: var(--color-bg);
  color: var(--color-text);
  font-family: -apple-system, BlinkMacSystemFont, 'PingFang SC', 'Helvetica Neue', 'Microsoft YaHei', sans-serif;
  font-size: 28rpx;
  line-height: 1.5;
}

/* shared utilities */
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
.section { margin-top: 48rpx; }
.bottom-space { height: calc(160rpx + env(safe-area-inset-bottom)); }
</style>
