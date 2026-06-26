// 位次定位结果的多维过滤。纯函数 matchesFilters 是主缝（vitest 表驱动测试）；
// Locator island 只管状态/UI，分档与过滤都走纯函数。维度间 AND、单维度多选内部 OR。
// 见 ADR-0008 / issue #9。

import type { LocEntry } from "./dingwei";

// public/data/school-meta.json 一条（按院校代码建键，紧凑键、空值省略）。
export interface SchoolMeta {
  p?: string; // 省份
  c?: string; // 城市
  ct?: string; // 城市层级
  o?: string; // 办学性质（公办/民办）
  k?: string; // 学校类别（综合类/理工类/…）
  lv?: string[]; // 层次：["985","211","双一流"] 中为真者
}
export type SchoolMetaMap = Record<string, SchoolMeta>;

// 12 学科门类 + 其他，码↔名。与 Go internal/hlj/menlei.go 的 menleiToCode 镜像，改一处要同步。
export const CATEGORIES: { code: string; label: string }[] = [
  { code: "工", label: "工学" },
  { code: "理", label: "理学" },
  { code: "医", label: "医学" },
  { code: "管", label: "管理学" },
  { code: "经", label: "经济学" },
  { code: "文", label: "文学" },
  { code: "法", label: "法学" },
  { code: "教", label: "教育学" },
  { code: "农", label: "农学" },
  { code: "艺", label: "艺术学" },
  { code: "史", label: "历史学" },
  { code: "哲", label: "哲学" },
  { code: "他", label: "其他" },
];
export const CATEGORY_LABEL: Record<string, string> = Object.fromEntries(
  CATEGORIES.map((c) => [c.code, c.label]),
);

// 固定选项（来源稳定，无须从数据派生）。省份/学校类别从已加载的 school-meta 派生。
export const LEVELS = ["985", "211", "双一流"] as const;
export const OWNERSHIPS = ["公办", "民办"] as const;
export const CITY_TIERS = ["一线", "新一线", "二线", "三线"] as const;

// 高收费阈值（元/年）：学费 ≥ 此值 视为高收费。
export const HIGH_FEE = 20000;

export interface Filters {
  provinces: string[]; // 省份（OR）
  levels: string[]; // 院校层次 985/211/双一流（OR）
  ownership: string[]; // 办学性质 公办/民办（OR）
  kinds: string[]; // 学校类别（OR）
  cityTiers: string[]; // 城市层级（OR）
  categories: string[]; // 专业大类=门类码（OR）
  keyword: string; // 专业关键词（空格分隔多词=任一匹配 OR；子串、大小写不敏感）
  minPlan: number; // 计划人数下限（>=，0=不限）
  maxGroupSize: number; // 组内专业数上限（<=，0=不限）
  hideCoopHighFee: boolean; // 隐藏中外合作及高收费
}

export function emptyFilters(): Filters {
  return {
    provinces: [],
    levels: [],
    ownership: [],
    kinds: [],
    cityTiers: [],
    categories: [],
    keyword: "",
    minPlan: 0,
    maxGroupSize: 0,
    hideCoopHighFee: false,
  };
}

/** 是否有任一过滤生效（用于决定空档文案、是否显示 chip 行）。 */
export function anyActive(f: Filters): boolean {
  return (
    f.provinces.length > 0 ||
    f.levels.length > 0 ||
    f.ownership.length > 0 ||
    f.kinds.length > 0 ||
    f.cityTiers.length > 0 ||
    f.categories.length > 0 ||
    f.keyword.trim() !== "" ||
    f.minPlan > 0 ||
    f.maxGroupSize > 0 ||
    f.hideCoopHighFee
  );
}

/**
 * 一条定位结果是否通过全部过滤。维度间 AND、单维度多选内部 OR。
 * 院校级维度（省份/层次/性质/类别/城市层级）取自 meta[entry.sc]；meta 缺失或该字段为空＝
 * 「未知」，仅当用户显式筛该维度时才被排除（不显式筛则放行），见 ADR-0008 Further Notes。
 */
export function matchesFilters(e: LocEntry, meta: SchoolMetaMap, f: Filters): boolean {
  const m = meta[e.sc];

  if (f.provinces.length && !(m?.p && f.provinces.includes(m.p))) return false;
  if (f.ownership.length && !(m?.o && f.ownership.includes(m.o))) return false;
  if (f.kinds.length && !(m?.k && f.kinds.includes(m.k))) return false;
  if (f.cityTiers.length && !(m?.ct && f.cityTiers.includes(m.ct))) return false;
  if (f.levels.length && !(m?.lv && f.levels.some((l) => m.lv!.includes(l)))) return false;

  if (f.categories.length && !(e.mc && f.categories.includes(e.mc))) return false;

  // 关键词：空格分隔多词，命中任一即可（OR）；子串、大小写不敏感。
  const kw = f.keyword.trim().toLowerCase();
  if (kw) {
    const name = (e.mn || "").toLowerCase();
    const terms = kw.split(/\s+/).filter(Boolean);
    if (!terms.some((t) => name.includes(t))) return false;
  }

  if (f.minPlan > 0 && (e.pl || 0) < f.minPlan) return false;
  if (f.maxGroupSize > 0 && (e.gs || 0) > f.maxGroupSize) return false;
  if (f.hideCoopHighFee && (e.cw || (e.tu || 0) >= HIGH_FEE)) return false;

  return true;
}
