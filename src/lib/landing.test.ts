import { describe, it, expect } from "vitest";
import {
  summarize,
  buildProvinceRows,
  coverageYears,
  type CoverageSummary,
} from "./landing";
import type { SchoolIndexEntry } from "./data";
import type { RosterEntry } from "./provinces";

function school(over: Partial<SchoolIndexEntry> = {}): SchoolIndexEntry {
  return {
    code: "0001",
    name: "某大学",
    leafCount: 10,
    ranges: { 物理: { year: 2025, minScore: 600, maxScore: 650, minRank: 1, maxRank: 9 } },
    ...over,
  };
}

describe("summarize — 本站收录摘要", () => {
  it("汇总院校数 / 数据条 / 名校 / 覆盖年份跨多校多科类", () => {
    const c = summarize([
      school({ leafCount: 10, is985: true, is211: true, ranges: { 物理: { year: 2022 } as any } }),
      school({ leafCount: 5, is211: true, ranges: { 历史: { year: 2025 } as any } }),
      school({ leafCount: 3, ranges: {} }),
    ]);
    expect(c.recruitSchools).toBe(3);
    expect(c.dataRows).toBe(18);
    expect(c.count985).toBe(1);
    expect(c.count211).toBe(2);
    expect(c.minYear).toBe(2022);
    expect(c.maxYear).toBe(2025);
  });

  it("空省 → 年份为 0，计数为 0", () => {
    const c = summarize([]);
    expect(c).toEqual({ recruitSchools: 0, dataRows: 0, minYear: 0, maxYear: 0, count985: 0, count211: 0 });
  });
});

describe("coverageYears — 覆盖年份展示", () => {
  const c = (minYear: number, maxYear: number): CoverageSummary => ({
    recruitSchools: 0, dataRows: 0, minYear, maxYear, count985: 0, count211: 0,
  });
  it.each([
    ["跨年", c(2022, 2025), "2022–2025"],
    ["单年", c(2025, 2025), "2025"],
    ["无数据", c(0, 0), "—"],
    ["undefined", undefined, "—"],
  ])("%s", (_label, input, want) => {
    expect(coverageYears(input as any)).toBe(want);
  });
});

describe("buildProvinceRows — 已上线置顶 + 敬请期待拼音 A→Z", () => {
  const roster: RosterEntry[] = [
    { name: "安徽", pinyin: "anhui" },
    { name: "浙江", pinyin: "zhejiang" },
    { name: "河南", pinyin: "henan" }, // 未上线
    { name: "北京", pinyin: "beijing" }, // 未上线
  ];
  const live = new Map([
    ["浙江", "zj"],
    ["安徽", "ah"],
  ]);
  const cov: Record<string, CoverageSummary> = {
    zj: { recruitSchools: 1693, dataRows: 41692, minYear: 2022, maxYear: 2025, count985: 50, count211: 126 },
    ah: { recruitSchools: 472, dataRows: 8970, minYear: 2025, maxYear: 2025, count985: 10, count211: 44 },
  };
  const rows = buildProvinceRows(roster, live, (s) => cov[s]);

  it("已上线在前，按招生院校数降序（浙江 1693 > 安徽 472）", () => {
    expect(rows.slice(0, 2).map((r) => r.name)).toEqual(["浙江", "安徽"]);
    expect(rows[0].slug).toBe("zj");
    expect(rows[0].coverage?.recruitSchools).toBe(1693);
  });

  it("敬请期待在后，按拼音 A→Z（北京 beijing < 河南 henan）", () => {
    expect(rows.slice(2).map((r) => r.name)).toEqual(["北京", "河南"]);
    expect(rows[2].live).toBe(false);
    expect(rows[2].slug).toBeUndefined();
    expect(rows[2].coverage).toBeUndefined();
  });

  it("不为未上线省调用 coverageOf", () => {
    const calls: string[] = [];
    buildProvinceRows(roster, live, (s) => {
      calls.push(s);
      return cov[s];
    });
    expect(calls.sort()).toEqual(["ah", "zj"]);
  });
});
