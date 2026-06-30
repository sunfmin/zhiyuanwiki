// 定位页搜索状态 ↔ URL query 的编解码。纯函数是主缝（vitest 表驱动测试），
// Locator island 只负责在挂载时读 URL、状态变更时 replaceState 回写。
// 目的：定位页「所有的搜索」都体现在 URL 里——复制链接给别人即可直接看到同一份结果。
//
// 设计：
//  - 只编码会影响结果的维度（科类 / 分数或位次 / 选科 / 全部筛选）；纯 UI 态（抽屉开合、
//    手机端当前档、展开剩余）不进 URL。
//  - 分数/位次合并进单一参数：mode=score → `score=NNN`，mode=rank → `rank=NNN`，参数名自解释。
//  - 筛选数组参数名与 Filters 字段 1:1（provinces/levels/…），改一处即对齐，少一层映射出错的机会。
//  - URL 有任一本模块参数即「URL 权威」：完全按 URL 重建（缺省维度回落到默认），不再混入 localStorage，
//    否则分享链接会被收链接者本地的旧输入污染。见 hasLocatorParams。

import { emptyFilters, type Filters } from "./filters";

export interface LocatorSearch {
  track: string;
  mode: "score" | "rank";
  val: string; // 原始输入串（数字）
  subjects: string[]; // 已选再选/选考科目（不含首选科类——那是 track）
  filters: Filters;
}

// 与 Filters 同名、可多选的数组维度。逗号分隔（OR）。
const ARRAY_FIELDS = [
  "provinces",
  "levels",
  "ownership",
  "kinds",
  "cityTiers",
  "categories",
] as const;

// 本模块认领的全部 query 参数名——用于判定一个 URL 是否「带定位搜索」。
export const LOCATOR_PARAM_KEYS = [
  "track",
  "score",
  "rank",
  "subjects",
  ...ARRAY_FIELDS,
  "majorKeyword",
  "schoolKeyword",
  "minPlan",
  "maxGroupSize",
  "hideCoopHighFee",
] as const;

/** URL 是否携带任一定位搜索参数（携带则按 URL 权威重建，忽略 localStorage）。 */
export function hasLocatorParams(params: URLSearchParams): boolean {
  return LOCATOR_PARAM_KEYS.some((k) => params.has(k));
}

/**
 * 搜索状态 → query string（不含前导 `?`）。只输出非默认/有意义的维度，空搜索返回 ""。
 * 选科只在「有输入值」时输出：无输入时不出结果，选科也就无须分享（且 wenli 省无选科）。
 */
export function encodeLocatorURL(
  s: LocatorSearch,
  opts: { defaultTrack: string; multiTrack: boolean; wenli: boolean },
): string {
  const p = new URLSearchParams();

  if (opts.multiTrack && s.track && s.track !== opts.defaultTrack) p.set("track", s.track);

  const v = s.val.trim();
  if (v) {
    p.set(s.mode === "score" ? "score" : "rank", v);
    // 选科随输入值一起分享（显式写出，不依赖收链接者的默认值是否相同）。
    if (!opts.wenli) p.set("subjects", s.subjects.join(","));
  }

  const f = s.filters;
  for (const key of ARRAY_FIELDS) {
    if (f[key].length) p.set(key, f[key].join(","));
  }
  if (f.majorKeyword.trim()) p.set("majorKeyword", f.majorKeyword.trim());
  if (f.schoolKeyword.trim()) p.set("schoolKeyword", f.schoolKeyword.trim());
  if (f.minPlan > 0) p.set("minPlan", String(f.minPlan));
  if (f.maxGroupSize > 0) p.set("maxGroupSize", String(f.maxGroupSize));
  if (f.hideCoopHighFee) p.set("hideCoopHighFee", "1");

  return p.toString();
}

/**
 * query → 搜索状态覆盖项。filters 总是完整返回（emptyFilters + 覆盖）；track/mode/val/subjects
 * 仅在 URL 显式给出时返回，缺省由调用方回落到自身默认（默认与省份配置相关，纯函数不掺和）。
 */
export function decodeLocatorURL(
  params: URLSearchParams,
  opts: { tracks: string[] },
): {
  track?: string;
  mode?: "score" | "rank";
  val?: string;
  subjects?: string[];
  filters: Filters;
} {
  const out: {
    track?: string;
    mode?: "score" | "rank";
    val?: string;
    subjects?: string[];
    filters: Filters;
  } = { filters: emptyFilters() };

  const track = params.get("track");
  if (track && opts.tracks.includes(track)) out.track = track;

  // score 优先于 rank（理论上不会同时出现）；值非正数视为无效，丢弃。
  if (params.has("score")) {
    out.mode = "score";
    out.val = sanitizeVal(params.get("score"));
  } else if (params.has("rank")) {
    out.mode = "rank";
    out.val = sanitizeVal(params.get("rank"));
  }

  if (params.has("subjects")) out.subjects = splitList(params.get("subjects"));

  const f = out.filters;
  for (const key of ARRAY_FIELDS) {
    if (params.has(key)) f[key] = splitList(params.get(key));
  }
  f.majorKeyword = params.get("majorKeyword") ?? "";
  f.schoolKeyword = params.get("schoolKeyword") ?? "";
  f.minPlan = clampNonNeg(params.get("minPlan"));
  f.maxGroupSize = clampNonNeg(params.get("maxGroupSize"));
  f.hideCoopHighFee = params.get("hideCoopHighFee") === "1";

  return out;
}

function splitList(raw: string | null): string[] {
  if (!raw) return [];
  return raw.split(",").map((s) => s.trim()).filter(Boolean);
}

// 输入串规整为正整数串；非数字 / ≤0 → ""（等同未输入）。
function sanitizeVal(raw: string | null): string {
  const n = parseInt((raw ?? "").trim(), 10);
  return Number.isNaN(n) || n <= 0 ? "" : String(n);
}

function clampNonNeg(raw: string | null): number {
  const n = parseInt((raw ?? "").trim(), 10);
  return Number.isNaN(n) || n < 0 ? 0 : n;
}
