// 落地页（省份列表页）的纯逻辑：把每省的 schools.json 汇总成「本站收录」摘要，
// 再把全国花名册排成展示行（已上线置顶 + 敬请期待按拼音 A→Z）。见 ADR-0016。
// 数据装载（import.meta.glob）留在页面/data.ts；本模块只做可单测的纯计算。
import type { SchoolIndexEntry } from "./data";
import type { RosterEntry } from "./provinces";

// CoverageSummary 是某省「本站收录」轴的摘要——衡量本站收了多少，不是省情。
// 招生院校数受收录年份影响（见 CONTEXT.md「招生院校」），并列时须配覆盖年份作口径。
export interface CoverageSummary {
  recruitSchools: number; // 招生院校（可报考）数 = schools.json 行数
  dataRows: number; // 数据条数 = 院校×专业叶子汇总
  minYear: number; // 覆盖年份下界（无数据为 0）
  maxYear: number; // 覆盖年份上界（无数据为 0）
  count985: number; // 名校数：985（修 tag 前为过标值，见 #18）
  count211: number; // 名校数：211
}

// summarize 把一省的院校索引汇总成收录摘要。
export function summarize(schools: SchoolIndexEntry[]): CoverageSummary {
  let dataRows = 0;
  let count985 = 0;
  let count211 = 0;
  let minYear = Infinity;
  let maxYear = 0;
  for (const s of schools) {
    dataRows += s.leafCount ?? 0;
    if (s.is985) count985++;
    if (s.is211) count211++;
    for (const track of Object.keys(s.ranges ?? {})) {
      const y = s.ranges[track]?.year;
      if (typeof y === "number" && y > 0) {
        if (y < minYear) minYear = y;
        if (y > maxYear) maxYear = y;
      }
    }
  }
  return {
    recruitSchools: schools.length,
    dataRows,
    minYear: minYear === Infinity ? 0 : minYear,
    maxYear,
    count985,
    count211,
  };
}

// ProvinceRow 是落地页一行：已上线（有 slug、可点、带收录摘要）或敬请期待（无 slug）。
export interface ProvinceRow {
  name: string;
  pinyin: string;
  slug?: string; // 有 = 已上线，整行链到 /[slug]/
  live: boolean;
  coverage?: CoverageSummary; // 仅已上线省有
}

// buildProvinceRows 把花名册排成展示顺序：已上线置顶（暂按招生院校数降序，高考人数排序
// 在 #16 接管）→ 敬请期待按拼音 A→Z。coverageOf 仅对已上线 slug 调用。
export function buildProvinceRows(
  roster: RosterEntry[],
  liveByName: Map<string, string>,
  coverageOf: (slug: string) => CoverageSummary,
): ProvinceRow[] {
  const rows: ProvinceRow[] = roster.map((r) => {
    const slug = liveByName.get(r.name);
    return slug
      ? { name: r.name, pinyin: r.pinyin, slug, live: true, coverage: coverageOf(slug) }
      : { name: r.name, pinyin: r.pinyin, live: false };
  });

  const live = rows
    .filter((r) => r.live)
    .sort((a, b) => (b.coverage?.recruitSchools ?? 0) - (a.coverage?.recruitSchools ?? 0));
  const pending = rows
    .filter((r) => !r.live)
    .sort((a, b) => a.pinyin.localeCompare(b.pinyin));

  return [...live, ...pending];
}

// coverageYears 把覆盖年份格式化为展示串：单年「2025」/ 跨年「2022–2025」/ 无「—」。
export function coverageYears(c: CoverageSummary | undefined): string {
  if (!c || c.maxYear === 0) return "—";
  return c.minYear === c.maxYear ? String(c.minYear) : `${c.minYear}–${c.maxYear}`;
}
