// 构建期数据装载（import.meta.glob，按省份分目录）。仅放「轻」索引与一分一段
// （每省各一两个文件）；每校/每专业详情的大 glob 留在各自详情页内联，避免被所有页打包。
import type { YiFenYiDuan } from "./fenduan";
import { provinceConfig, trackSlugOf } from "./provinces";
import { gaokaoLatestCommon, type Gaokao } from "./landing";
import homeSchoolsData from "../data/home-schools.json";
import benkePlanData from "../data/benke-plan.json";
import gaokaoTotalData from "../data/gaokao-total.json";

export type TrackRange = {
  year: number;
  minScore: number;
  maxScore: number;
  minRank: number;
  maxRank: number;
};

export interface SchoolIndexEntry {
  code: string;
  name: string;
  leafCount: number;
  ranges: Record<string, TrackRange>; // 科类名 → 区间
  is985?: boolean;
  is211?: boolean;
  isShuangYiLiu?: boolean;
}

export type YearScore = {
  year: number;
  track: string;
  minScore: number;
  minRank: number;
  maxScore: number;
};
export type Leaf = { majorKey: string; majorName: string; selKe: string; years: YearScore[] };

// 黑龙江：院校专业组内专业
export type GroupMajor = {
  majorName: string;
  majorKey: string;
  selKe: string;
  plan: number;
  tuition: string;
  menlei?: string;
  coop?: boolean;
  prevYear?: number;
  prevRank?: number;
  equivRank?: number;
};
export type Group2026 = {
  groupCode: string;
  groupName: string;
  track: string;
  selKe: string;
  majors: GroupMajor[];
};

// 浙江：院校×专业（专业平行志愿）
export type PlanMajor = {
  majorName: string;
  majorKey: string;
  selKe: string;
  plan: number;
  tuition?: string;
  schooling?: string;
  menlei?: string;
  coop?: boolean;
  prevYear?: number;
  prevRank?: number;
  equivRank?: number;
  prevScore?: number; // 只有分数省（西藏）：最近年最低分（位次缺失时的定位/排序基准）
};

export interface SchoolDetail {
  code: string;
  name: string;
  leaves: Leaf[];
  groups2026?: Group2026[];
  plan2026?: PlanMajor[];
}

export interface MajorIndexEntry {
  key: string;
  name: string;
  schoolCount: number;
}
export type MajorSchool = {
  sc: string;
  sn: string;
  mn?: string; // 叶子全名（含方向后缀），用于专业页内消歧（如大类各方向）
  mk: string;
  minRank: number;
  minScore?: number; // 只有分数省（西藏）：最近年最低分（位次缺失时横向比较用）
  year: number;
  track: string;
};
export interface MajorDetail {
  key: string;
  name: string;
  schools: MajorSchool[];
}

const schoolsIndex = import.meta.glob("/src/data/*/schools.json", { eager: true });
const majorsIndex = import.meta.glob("/src/data/*/majors.json", { eager: true });
const fenduanTables = import.meta.glob("/src/data/*/yifenyiduan/*.json", { eager: true });

export function schoolsOf(prov: string): SchoolIndexEntry[] {
  return ((schoolsIndex[`/src/data/${prov}/schools.json`] as any)?.default ?? []) as SchoolIndexEntry[];
}

export function majorsOf(prov: string): MajorIndexEntry[] {
  return ((majorsIndex[`/src/data/${prov}/majors.json`] as any)?.default ?? []) as MajorIndexEntry[];
}

// fenduanOf 取该省定位/换算所用的一分一段表（黑龙江 2026 物理；浙江 2026 综合）。
// 只有分数省（西藏）无一分一段——返回空表 stub（entries 空），定位走分数域不依赖它（见 provinces.locatorBasis）。
export function fenduanOf(prov: string): YiFenYiDuan {
  const cfg = provinceConfig(prov);
  const slug = trackSlugOf(cfg, cfg.fenduanTrack);
  const key = `/src/data/${prov}/yifenyiduan/${slug}-${cfg.fenduanYear}.json`;
  const tbl = fenduanTables[key] as { default: YiFenYiDuan } | undefined;
  if (!tbl) return { province: cfg.name, track: cfg.fenduanTrack, year: cfg.fenduanYear, entries: [] };
  return tbl.default;
}

// 本省院校数（省情）：校址在该省的高校数，来自 landing emit 的全国 school 表投影（中文省名→数）。
const homeSchools = homeSchoolsData as Record<string, number>;
export function homeSchoolsOf(name: string): number | undefined {
  return homeSchools[name];
}

// 本科招生计划（省情）：该省最新年本科批招生计划总数，来自 landing emit 的 plan 表投影（slug→{plan,year}）。
// 仅已入库省有值（pending 省显「—」）。配合高考人数可粗读本省本科竞争度。
const benkePlan = benkePlanData as Record<string, { plan: number; year: number }>;
export function benkePlanOf(slug: string): { plan: number; year: number } | undefined {
  return benkePlan[slug];
}

// benkeLineOf 取某省「物理本科线」（本科批控制线）：物理科一分一段最新年的 controlLine。
// 源一分一段「控制线」列是本科批控制线（本科线，如江苏 2025 物理 463），不是更高的特控线。
// 无物理科（浙江综合）或源无控制线（黑龙江走 core 解析路径未采）→ undefined → 显「—」。
export function benkeLineOf(prov: string): { line: number; year: number } | undefined {
  let best: { line: number; year: number } | undefined;
  for (const key of Object.keys(fenduanTables)) {
    const m = key.match(/\/src\/data\/([^/]+)\/yifenyiduan\/wuli-(\d+)\.json$/);
    if (!m || m[1] !== prov) continue;
    const tbl = (fenduanTables[key] as any).default as YiFenYiDuan;
    const year = Number(m[2]);
    if (tbl.controlLine && (!best || year > best.year)) best = { line: tbl.controlLine, year };
  }
  return best;
}

// 真实统考人数覆盖（省情）：江苏/山西/上海/北京 等省一分一段只发布到本科批控制线，
// 「一分一段最大累计」= 本科上线人数（非全体统考考生），用作高考人数/分母会让本科招生计划占比虚高，
// 且与「满库省」（河北/山东等，一分一段到分数下限）不可比。故对这些省用官方普通类统考排名总人数覆盖，
// 使高考人数与本科计划占比同口径（全体统考考生）。不在此表的省照常用一分一段最大累计。来源见各条 source。
const gaokaoTotal = gaokaoTotalData as Record<
  string,
  { count: number; year: number; source: string; note?: string }
>;

// gaokaoOf 算某省高考人数（统考排名人数）：优先取 gaokaoTotal 覆盖（截断省用官方总数）；
// 其余省从已收录的各科类一分一段表，取最新共同年的最大累计求和。
// 数据全在前端（hlj/zj 不在 staging DB，但 committed 一分一段是 6 省统一源）。见 ADR-0016。
export function gaokaoOf(prov: string): Gaokao | undefined {
  const real = gaokaoTotal[prov];
  if (real) return { count: real.count, year: real.year };
  const cfg = provinceConfig(prov);
  const avail = new Map<string, Map<number, number>>();
  for (const t of cfg.tracks) {
    const slug = trackSlugOf(cfg, t.name);
    const byYear = new Map<number, number>();
    for (const key of Object.keys(fenduanTables)) {
      const m = key.match(/\/src\/data\/([^/]+)\/yifenyiduan\/([^/]+)-(\d+)\.json$/);
      if (!m || m[1] !== prov || m[2] !== slug) continue;
      const tbl = (fenduanTables[key] as any).default as YiFenYiDuan;
      let mx = 0;
      for (const e of tbl.entries) if (e.cumulative > mx) mx = e.cumulative;
      byYear.set(Number(m[3]), mx);
    }
    avail.set(t.name, byYear);
  }
  return gaokaoLatestCommon(
    cfg.tracks.map((t) => t.name),
    avail,
  );
}
