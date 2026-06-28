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

// Gaokao 是某省的高考人数（统考排名人数，见 CONTEXT.md）及其所属年份。
export interface Gaokao {
  count: number; // = 各配置科类「最新共同年」一分一段最大累计之和
  year: number;
}

// gaokaoLatestCommon 取各配置科类都有数据的「最新共同年」，把各科类该年最大累计求和。
// 任一科类无数据（如黑龙江缺历史一分一段）→ 无共同年 → undefined（页面显示「—」，不凑不全的数）。
// avail：科类名 → (年份 → 该年最大累计)。
export function gaokaoLatestCommon(
  tracks: string[],
  avail: Map<string, Map<number, number>>,
): Gaokao | undefined {
  const perTrack = tracks.map((t) => avail.get(t) ?? new Map<number, number>());
  if (perTrack.some((m) => m.size === 0)) return undefined;
  const common = [...perTrack[0].keys()].filter((y) => perTrack.every((m) => m.has(y)));
  if (common.length === 0) return undefined;
  const year = Math.max(...common);
  let count = 0;
  for (const m of perTrack) count += m.get(year)!;
  return { count, year };
}

// BenkeLine 是某省物理本科线（本科批控制线）及其年份。
export interface BenkeLine {
  line: number;
  year: number;
}

// ProvinceRow 是落地页一行：已上线（有 slug、可点、带收录摘要）或敬请期待（无 slug）。
export interface ProvinceRow {
  name: string;
  pinyin: string;
  slug?: string; // 有 = 已上线，整行链到 /[slug]/
  live: boolean;
  homeSchools?: number; // 本省院校数（省情）——全 31 省皆有（含未上线）
  coverage?: CoverageSummary; // 本站收录——仅已上线省
  gaokao?: Gaokao; // 高考人数（省情）——仅已上线且科类数据完整
  benkeLine?: BenkeLine; // 物理本科线（省情）——仅有物理科一分一段控制线的省
}

// buildProvinceRows 把花名册排成展示顺序：已上线置顶（按高考人数降序，缺高考人数的省如黑龙江
// 殿后）→ 敬请期待按拼音 A→Z。coverageOf/gaokaoOf/benkeOf 仅对已上线 slug 调用；homeOf 按名取全省。
export function buildProvinceRows(
  roster: RosterEntry[],
  liveByName: Map<string, string>,
  coverageOf: (slug: string) => CoverageSummary,
  homeOf: (name: string) => number | undefined,
  gaokaoOf: (slug: string) => Gaokao | undefined,
  benkeOf: (slug: string) => BenkeLine | undefined,
): ProvinceRow[] {
  const rows: ProvinceRow[] = roster.map((r) => {
    const slug = liveByName.get(r.name);
    const homeSchools = homeOf(r.name);
    return slug
      ? {
          name: r.name,
          pinyin: r.pinyin,
          slug,
          live: true,
          homeSchools,
          coverage: coverageOf(slug),
          gaokao: gaokaoOf(slug),
          benkeLine: benkeOf(slug),
        }
      : { name: r.name, pinyin: r.pinyin, live: false, homeSchools };
  });

  const live = rows
    .filter((r) => r.live)
    .sort((a, b) => (b.gaokao?.count ?? -1) - (a.gaokao?.count ?? -1));
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
