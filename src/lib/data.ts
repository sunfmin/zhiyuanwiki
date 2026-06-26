// 构建期数据装载（import.meta.glob，按省份分目录）。仅放「轻」索引与一分一段
// （每省各一两个文件）；每校/每专业详情的大 glob 留在各自详情页内联，避免被所有页打包。
import type { YiFenYiDuan } from "./fenduan";
import { provinceConfig, trackSlugOf } from "./provinces";

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

// fenduanOf 取该省定位/换算所用的一分一段表（黑龙江 2026 物理；浙江 2025 综合）。
export function fenduanOf(prov: string): YiFenYiDuan {
  const cfg = provinceConfig(prov);
  const slug = trackSlugOf(cfg, cfg.fenduanTrack);
  const key = `/src/data/${prov}/yifenyiduan/${slug}-${cfg.fenduanYear}.json`;
  return (fenduanTables[key] as any).default as YiFenYiDuan;
}
