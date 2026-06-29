// 部署护栏（ADR-0018）：迁到 R2 后用 `rclone sync --delete` 上传 dist，若某次构建半截/损坏导致
// HTML 数骤降，--delete 会把线上大量页面一并删掉。本函数是纯决策——新构建 HTML 数比线上少超过
// maxDropRatio（默认 5%）即中止部署；线上无基线（首次部署，liveCount=0）则放行。
//
// 单一真相：CI（.github/workflows/deploy.yml）与单测都调用它，逻辑只此一份（见 scripts/deploy-tripwire.test.ts）。
import { pathToFileURL } from "node:url";

/**
 * @param {number} liveCount 线上现有 HTML 文件数（CI 用 rclone 查 R2 得）
 * @param {number} newCount 本次构建 dist 内 HTML 文件数
 * @param {number} [maxDropRatio=0.05] 允许的最大回落比例（0–1）
 * @returns {{ abort: boolean, reason: string }}
 */
export function shouldAbortDeploy(liveCount, newCount, maxDropRatio = 0.05) {
  const finite = [liveCount, newCount, maxDropRatio].every((n) => Number.isFinite(n));
  if (!finite || newCount < 0 || liveCount < 0) {
    return { abort: true, reason: `计数无效（live=${liveCount} new=${newCount} ratio=${maxDropRatio}）：保守中止` };
  }
  if (liveCount === 0) {
    return { abort: false, reason: `线上无基线（首次部署）：放行 ${newCount} 文件` };
  }
  const dropRatio = (liveCount - newCount) / liveCount; // 正 = 减少
  const pct = (dropRatio * 100).toFixed(1);
  const cap = (maxDropRatio * 100).toFixed(1);
  if (dropRatio > maxDropRatio) {
    return {
      abort: true,
      reason: `新构建 ${newCount} 比线上 ${liveCount} 少 ${pct}%（> ${cap}% 阈值）：疑似半截构建，中止`,
    };
  }
  return { abort: false, reason: `新构建 ${newCount} vs 线上 ${liveCount}（回落 ${pct}% ≤ ${cap}%）：放行` };
}

// CLI：node scripts/deploy-tripwire.mjs <liveCount> <newCount> [maxDropRatio]
// 中止 → 退出码 1（CI step 失败）；放行 → 0。决策理由打到 stderr。
if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const [liveArg, newArg, ratioArg] = process.argv.slice(2);
  const res = shouldAbortDeploy(
    Number(liveArg),
    Number(newArg),
    ratioArg === undefined ? undefined : Number(ratioArg),
  );
  console.error(res.reason);
  process.exit(res.abort ? 1 : 0);
}
