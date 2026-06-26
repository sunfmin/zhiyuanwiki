// 位次定位：冲/稳/保分档 + 选科判定。纯函数，可测；与 Go internal/hlj 选科逻辑镜像。

export type Bucket = "冲" | "稳" | "保" | "out";

/**
 * 冲稳保分档。V=访客位次，R=该专业等效录取位次（越小越难）。
 * 阈值：你的位次明显好于录取线→保；约等于→稳；略低于→冲；太低→够不着。
 */
export function classify(V: number, R: number): Bucket {
  if (R <= 0 || V <= 0) return "out";
  const ratio = V / R;
  if (ratio <= 0.9) return "保";
  if (ratio <= 1.02) return "稳";
  if (ratio <= 1.15) return "冲";
  return "out";
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
