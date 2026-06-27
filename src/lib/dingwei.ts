// 位次定位：冲/稳/保分档 + 选科判定。纯函数，可测；与 Go internal/hlj 选科逻辑镜像。

// 把握频谱（难 → 易）。够不着/过保是两端"远档"，不是冲/保。详见 ADR-0010 与 CONTEXT.md。
export type Bucket = "够不着" | "冲" | "稳" | "保" | "过保";
// 冲稳保三主档（定位页分列、院校页角标主色）。
export type MainTier = "冲" | "稳" | "保";
export const MAIN_TIERS: MainTier[] = ["冲", "稳", "保"];

/**
 * 冲稳保分档。V=访客位次，R=该专业等效录取位次（越小越难）。
 * 按比值 V/R 贴把握标签——你明显好于线→保；约等于→稳；略低于→冲；
 * 太难→够不着；太易（白白浪费位次）→过保。无效输入→null（不是某一档）。
 * 阈值沿用院校页角标历史值 1.02/1.15/0.90；0.75 是新加的保档下限。
 */
export function classify(V: number, R: number): Bucket | null {
  if (R <= 0 || V <= 0) return null;
  const ratio = V / R;
  if (ratio > 1.15) return "够不着";
  if (ratio > 1.02) return "冲";
  if (ratio > 0.9) return "稳";
  if (ratio >= 0.75) return "保";
  return "过保";
}

export type BucketGroups = Record<Bucket, LocEntry[]>;

const emptyGroups = (): BucketGroups => ({ 够不着: [], 冲: [], 稳: [], 保: [], 过保: [] });

/**
 * 按把握把候选分档：classify 分组 + 每档"最贴近本人位次"在前（按 |R−V| 升序）。
 * **不凑数、不截断**——稀疏就有几条显几条，密集时的显示上限由调用方（UI）处理。
 * 调用方须先做选科 / 筛选过滤；r≤0 的条目（classify 返回 null）自动落空。
 */
export function bucketize(V: number, entries: LocEntry[]): BucketGroups {
  const out = emptyGroups();
  if (V <= 0) return out;
  for (const e of entries) {
    const b = classify(V, e.r);
    if (b) out[b].push(e);
  }
  for (const k of Object.keys(out) as Bucket[]) {
    out[k].sort((a, b) => Math.abs(a.r - V) - Math.abs(b.r - V));
  }
  return out;
}

// 列装配（ADR-0010 承重规则的唯一实现，原内联在 Locator.render）。
// TIER_CAP：密集时每主档收起态先显多少条——纯展示上限，防刷屏，绝不向上填满主档。
// TARGET：冲/保列用远档「补齐」到的目标条数（「保证 100」的浏览量诉求，只发生在远档区）。
export const TIER_CAP = 30;
export const TARGET = 100;

// 主档 → 远档映射：冲列末尾挂「够不着」、保列末尾挂「过保」、稳列无远档。
const FAR_OF: Record<MainTier, "够不着" | "过保" | null> = { 冲: "够不着", 稳: null, 保: "过保" };

export interface TierColumn {
  tier: MainTier;
  all: LocEntry[]; // 全部真实档（计数、「展开剩余」用）——主档只装真档，不凑数
  capped: LocEntry[]; // 收起态主列（截断到 cap）
  hasMore: boolean; // all 超过 cap，可「展开剩余」
  // 远档预览：把本列补齐到 target（真实档已 ≥target、或远档桶为空 → null，即不挂）。
  far: { bucket: "够不着" | "过保"; entries: LocEntry[] } | null;
}

/**
 * 把分档结果装配成冲/稳/保三列。**主档只装真档**（不凑数）：密集时收起态截断到 cap、可展开；
 * 稀疏就有几条显几条。仅在**远档区**用「够不着」/「过保」把冲列、保列各补齐到 target
 * （真实档已 ≥target 则不补），**稳列无远档**。补进来的远档归调用方降级展示，绝不冒充冲/保。
 */
export function assembleColumns(
  groups: BucketGroups,
  opts: { cap?: number; target?: number } = {},
): Record<MainTier, TierColumn> {
  const cap = opts.cap ?? TIER_CAP;
  const target = opts.target ?? TARGET;
  const out = {} as Record<MainTier, TierColumn>;
  for (const tier of MAIN_TIERS) {
    const all = groups[tier];
    const farBucket = FAR_OF[tier];
    let far: TierColumn["far"] = null;
    if (farBucket) {
      const need = Math.max(0, target - all.length);
      const entries = need > 0 ? groups[farBucket].slice(0, need) : [];
      if (entries.length > 0) far = { bucket: farBucket, entries };
    }
    out[tier] = { tier, all, capped: all.slice(0, cap), hasMore: all.length > cap, far };
  }
  return out;
}

const SUBJECTS = ["物理", "历史", "化学", "生物", "政治", "地理"];

/** 黑龙江选科判定（"不限"/"化学"/"化学和生物"/"化学或生物"），与 Go SelKeAllows 镜像。 */
export function selKeAllows(req: string, chosen: Set<string>): boolean {
  req = (req || "").trim();
  if (!req || req.includes("不限")) return true;
  const subs = SUBJECTS.filter((s) => req.includes(s));
  if (subs.length === 0) return true;
  if (req.includes("或")) return subs.some((s) => chosen.has(s));
  return subs.every((s) => chosen.has(s));
}

// 浙江选科要求格式：全名+「、」+「(N科必选)」=且；缩写块「物化生/史政地/物地技」=且；
// 「物/化/生(3选1)」=或；「X必选」=单科；「思想政治」即政治。chosen 以「政治」为规范名。
const ZJ_ABBR: Record<string, string> = {
  物: "物理", 化: "化学", 生: "生物", 政: "政治", 史: "历史", 地: "地理", 技: "技术",
};

function zjExtractSubjects(req: string): string[] {
  const set = new Set<string>();
  for (const s of ["物理", "化学", "生物", "历史", "地理", "技术"]) if (req.includes(s)) set.add(s);
  if (req.includes("政治")) set.add("政治"); // 含「思想政治」
  if (set.size) return [...set];
  // 无全名 → 缩写块（取首个括号前），逐字映射。
  const head = req.split(/[(（]/)[0];
  for (const ch of head) if (ZJ_ABBR[ch]) set.add(ZJ_ABBR[ch]);
  return [...set];
}

/** 浙江 7选3 选科判定。chosen 为考生选考科目集合（≤3，规范名含「政治」「技术」）。 */
export function selKeAllowsZJ(req: string, chosen: Set<string>): boolean {
  req = (req || "").trim();
  if (!req || req.includes("不限")) return true;
  const isOr = /选1|选一|\/|或|任选/.test(req);
  const subs = zjExtractSubjects(req);
  if (subs.length === 0) return true; // 无法识别默认放行（保守）
  return isOr ? subs.some((s) => chosen.has(s)) : subs.every((s) => chosen.has(s));
}

// 紧凑定位索引项（与 Go locatorEntry 的 JSON 键一致）。组字段（gc/gn/gs）仅黑龙江有。
export interface LocEntry {
  sc: string; // 院校代码
  sn: string; // 院校名称
  gc?: string; // 组代码（黑龙江）
  gn?: string; // 组名（黑龙江）
  mn: string; // 专业名
  mk: string; // 专业键
  sk: string; // 选科要求
  pl: number; // 计划
  r: number; // 等效位次
  py: number; // 挂接年份
  gs?: number; // 组内专业数（黑龙江）
  mc?: string; // 学科门类 1 字码（见 src/lib/filters.ts CATEGORIES）
  tu?: number; // 学费（元/年，待定/无→省略）
  cw?: boolean; // 中外合作办学
}
