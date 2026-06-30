import { describe, it, expect } from "vitest";
import { PROVINCES, fillRange, type FillWindow } from "./provinces";

describe("fillRange — 填报窗口紧凑展示", () => {
  it("跨日：6月29日—7月2日", () => {
    expect(fillRange({ start: "2026-06-29", end: "2026-07-02" })).toBe("6月29日—7月2日");
  });
  it("去前导零：6月5日—6月9日", () => {
    expect(fillRange({ start: "2026-06-05", end: "2026-06-09" })).toBe("6月5日—6月9日");
  });
  it("同一天只显一个日期（不显「—」）", () => {
    expect(fillRange({ start: "2026-07-01", end: "2026-07-01" })).toBe("7月1日");
  });
});

// 数据完整性：每个配置了 fill 的省，日期必须是合法 ISO、2026 年、start ≤ end，
// endTime（若有）须为 HH:MM。手录数据的安全网——录错一处即红。
describe("PROVINCES.fill 数据完整性", () => {
  const ISO = /^2026-\d{2}-\d{2}$/;
  const withFill = Object.entries(PROVINCES).filter(([, c]) => c.fill) as [string, { fill: FillWindow }][];

  it("至少录入了多数省份（防止整体漏填）", () => {
    expect(withFill.length).toBeGreaterThanOrEqual(20);
  });

  for (const [slug, c] of withFill) {
    it(`${slug}: 起止日期合法且 start ≤ end`, () => {
      const f = c.fill;
      expect(f.start, `${slug}.start 非 2026 ISO`).toMatch(ISO);
      expect(f.end, `${slug}.end 非 2026 ISO`).toMatch(ISO);
      expect(new Date(f.start).getTime(), `${slug}: start 应 ≤ end`).toBeLessThanOrEqual(new Date(f.end).getTime());
      if (f.endTime != null) expect(f.endTime, `${slug}.endTime 非 HH:MM`).toMatch(/^\d{1,2}:\d{2}$/);
    });
  }
});
