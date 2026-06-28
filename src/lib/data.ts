// 构建期数据装载（import.meta.glob，按省份分目录）。仅放「轻」索引与一分一段
// （每省各一两个文件）；每校/每专业详情的大 glob 留在各自详情页内联，避免被所有页打包。
import type { YiFenYiDuan } from "./fenduan";
import { provinceConfig, trackSlugOf } from "./provinces";
import { gaokaoLatestCommon, type Gaokao } from "./landing";
import homeSchoolsData from "../data/home-schools.json";

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
export function fenduanOf(prov: string): YiFenYiDuan {
  const cfg = provinceConfig(prov);
  const slug = trackSlugOf(cfg, cfg.fenduanTrack);
  const key = `/src/data/${prov}/yifenyiduan/${slug}-${cfg.fenduanYear}.json`;
  return (fenduanTables[key] as any).default as YiFenYiDuan;
}

// 本省院校数（省情）：校址在该省的高校数，来自 landing emit 的全国 school 表投影（中文省名→数）。
const homeSchools = homeSchoolsData as Record<string, number>;
export function homeSchoolsOf(name: string): number | undefined {
  return homeSchools[name];
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

// gaokaoOf 算某省高考人数（统考排名人数）：从已收录的各科类一分一段表，取最新共同年的最大累计求和。
// 数据全在前端（hlj/zj 不在 staging DB，但 committed 一分一段是 6 省统一源）。见 ADR-0016。
export function gaokaoOf(prov: string): Gaokao | undefined {
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
