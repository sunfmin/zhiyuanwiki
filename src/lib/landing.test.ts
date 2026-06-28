import { describe, it, expect } from "vitest";
import {
  summarize,
  buildProvinceRows,
  coverageYears,
  gaokaoLatestCommon,
  famousCount,
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
      school({ name: "北京大学", leafCount: 10, is985: true, is211: true, ranges: { 物理: { year: 2022 } as any } }),
      school({ name: "苏州大学", leafCount: 5, is211: true, ranges: { 历史: { year: 2025 } as any } }),
      school({ name: "某职院", leafCount: 3, ranges: {} }),
    ]);
    expect(c.recruitSchools).toBe(3);
    expect(c.dataRows).toBe(18);
    expect(c.count985).toBe(1);
    expect(c.count211).toBe(2);
    expect(c.minYear).toBe(2022);
    expect(c.maxYear).toBe(2025);
  });

  it("名校数把分校区/医学院收口到母体（哈工大本部+威海+深圳 计 1 所 985）", () => {
    const c = summarize([
      school({ name: "哈尔滨工业大学", is985: true, is211: true }),
      school({ name: "哈尔滨工业大学(威海校区)", is985: true, is211: true }),
      school({ name: "哈尔滨工业大学(深圳)", is985: true, is211: true }),
      school({ name: "北京大学", is985: true, is211: true }),
      school({ name: "北京大学医学部", is985: true, is211: true }),
    ]);
    expect(c.count985).toBe(2); // 哈工大 + 北大，校区/医学部并入
    expect(c.count211).toBe(2);
  });

  it("空省 → 年份为 0，计数为 0", () => {
    const c = summarize([]);
    expect(c).toEqual({ recruitSchools: 0, dataRows: 0, minYear: 0, maxYear: 0, count985: 0, count211: 0 });
  });
});

describe("famousCount — 分校区前缀收口", () => {
  it("母体存在时，校区/医学院/分校并入母体", () => {
    expect(
      famousCount([
        "哈尔滨工业大学",
        "哈尔滨工业大学(威海校区)",
        "哈尔滨工业大学(深圳)",
        "东北大学",
        "东北大学秦皇岛分校",
        "复旦大学",
        "复旦大学上海医学院",
      ]),
    ).toBe(3); // 哈工大 / 东北大学 / 复旦
  });
  it("无前缀关系的独立大学各计一所；重复名去重", () => {
    expect(famousCount(["北京大学", "清华大学", "北京大学"])).toBe(2);
  });
  it("空", () => expect(famousCount([])).toBe(0));
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

describe("gaokaoLatestCommon — 最新共同年的各科类最大累计之和", () => {
  it("3+1+2：取两科类都有的最新年，物理+历史相加", () => {
    const avail = new Map([
      ["物理", new Map([[2024, 180000], [2025, 190000]])],
      ["历史", new Map([[2024, 45000], [2025, 47000]])],
    ]);
    expect(gaokaoLatestCommon(["物理", "历史"], avail)).toEqual({ count: 237000, year: 2025 });
  });

  it("单科类（综合）：直接取最新年最大累计", () => {
    const avail = new Map([["综合", new Map([[2025, 390000], [2026, 395000]])]]);
    expect(gaokaoLatestCommon(["综合"], avail)).toEqual({ count: 395000, year: 2026 });
  });

  it("某科类完全缺数据（黑龙江缺历史）→ undefined", () => {
    const avail = new Map([
      ["物理", new Map([[2026, 190000]])],
      ["历史", new Map<number, number>()],
    ]);
    expect(gaokaoLatestCommon(["物理", "历史"], avail)).toBeUndefined();
  });

  it("两科类年份无交集 → undefined（不跨年硬凑）", () => {
    const avail = new Map([
      ["物理", new Map([[2025, 1]])],
      ["历史", new Map([[2024, 1]])],
    ]);
    expect(gaokaoLatestCommon(["物理", "历史"], avail)).toBeUndefined();
  });
});

describe("buildProvinceRows — 已上线按高考人数降序 + 敬请期待拼音 A→Z", () => {
  const roster: RosterEntry[] = [
    { name: "安徽", pinyin: "anhui" },
    { name: "浙江", pinyin: "zhejiang" },
    { name: "黑龙江", pinyin: "heilongjiang" }, // 已上线但无高考人数（缺历史）
    { name: "河南", pinyin: "henan" }, // 未上线
    { name: "北京", pinyin: "beijing" }, // 未上线
  ];
  const live = new Map([
    ["浙江", "zj"],
    ["安徽", "ah"],
    ["黑龙江", "hlj"],
  ]);
  const cov: Record<string, CoverageSummary> = {
    zj: { recruitSchools: 1693, dataRows: 41692, minYear: 2022, maxYear: 2025, count985: 50, count211: 126 },
    ah: { recruitSchools: 472, dataRows: 8970, minYear: 2025, maxYear: 2025, count985: 10, count211: 44 },
    hlj: { recruitSchools: 1077, dataRows: 19895, minYear: 2023, maxYear: 2025, count985: 50, count211: 134 },
  };
  const home: Record<string, number> = { 浙江: 113, 安徽: 128, 黑龙江: 82, 河南: 185, 北京: 102 };
  const gk: Record<string, { count: number; year: number } | undefined> = {
    zj: { count: 390000, year: 2026 },
    ah: { count: 236000, year: 2025 },
    hlj: undefined, // 缺历史
  };
  const bk: Record<string, { line: number; year: number } | undefined> = {
    zj: undefined, // 综合无物理本科线
    ah: { line: 461, year: 2025 },
    hlj: undefined,
  };
  const rows = buildProvinceRows(roster, live, (s) => cov[s], (n) => home[n], (s) => gk[s], (s) => bk[s]);

  it("已上线按高考人数降序，无高考人数的黑龙江殿后（仍在已上线段内）", () => {
    expect(rows.slice(0, 3).map((r) => r.name)).toEqual(["浙江", "安徽", "黑龙江"]);
    expect(rows[2].slug).toBe("hlj");
    expect(rows[2].gaokao).toBeUndefined();
  });

  it("本省院校数全省皆有（含未上线河南 185）", () => {
    expect(rows.find((r) => r.name === "河南")?.homeSchools).toBe(185);
    expect(rows.find((r) => r.name === "浙江")?.homeSchools).toBe(113);
  });

  it("物理本科线：安徽 461 有值，浙江（综合）无", () => {
    expect(rows.find((r) => r.name === "安徽")?.benkeLine).toEqual({ line: 461, year: 2025 });
    expect(rows.find((r) => r.name === "浙江")?.benkeLine).toBeUndefined();
  });

  it("敬请期待在后，按拼音 A→Z（北京 < 河南），无 coverage/gaokao", () => {
    expect(rows.slice(3).map((r) => r.name)).toEqual(["北京", "河南"]);
    expect(rows[3].live).toBe(false);
    expect(rows[3].coverage).toBeUndefined();
    expect(rows[3].gaokao).toBeUndefined();
  });

  it("不为未上线省调用 coverageOf / gaokaoOf", () => {
    const cCalls: string[] = [];
    const gCalls: string[] = [];
    buildProvinceRows(
      roster, live,
      (s) => { cCalls.push(s); return cov[s]; },
      (n) => home[n],
      (s) => { gCalls.push(s); return gk[s]; },
      (s) => bk[s],
    );
    expect(cCalls.sort()).toEqual(["ah", "hlj", "zj"]);
    expect(gCalls.sort()).toEqual(["ah", "hlj", "zj"]);
  });
});
