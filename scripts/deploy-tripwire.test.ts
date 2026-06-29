import { describe, expect, it } from "vitest";
import { shouldAbortDeploy } from "./deploy-tripwire.mjs";

// 部署护栏的边界行为（ADR-0018）。阈值默认 5%，判定用「严格大于」——恰好 5% 放行、略超即中止。
describe("shouldAbortDeploy 部署护栏", () => {
  const cases: Array<{ name: string; live: number; neu: number; ratio?: number; abort: boolean }> = [
    { name: "首次部署（线上无基线）放行", live: 0, neu: 40000, abort: false },
    { name: "文件数持平放行", live: 40000, neu: 40000, abort: false },
    { name: "正常增长放行", live: 40000, neu: 44000, abort: false },
    { name: "小幅回落 4.5%（< 5%）放行", live: 40000, neu: 38200, abort: false },
    { name: "恰好 5% 回落放行（非严格大于）", live: 40000, neu: 38000, abort: false },
    { name: "略超 5% 回落中止", live: 40000, neu: 37999, abort: true },
    { name: "半截构建骤降中止", live: 40000, neu: 100, abort: true },
    { name: "归零（空 dist）中止", live: 40000, neu: 0, abort: true },
    { name: "自定义阈值放宽到 50%：30% 回落放行", live: 40000, neu: 28000, ratio: 0.5, abort: false },
    { name: "非有限计数（NaN）保守中止", live: NaN, neu: 40000, abort: true },
    { name: "负数计数保守中止", live: -1, neu: 40000, abort: true },
  ];

  for (const c of cases) {
    it(c.name, () => {
      expect(shouldAbortDeploy(c.live, c.neu, c.ratio).abort).toBe(c.abort);
    });
  }
});
