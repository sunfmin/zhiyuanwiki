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

/** 选科判定，与 Go SelKeAllows 镜像。chosen 为已选科目集合。 */
export function selKeAllows(req: string, chosen: Set<string>): boolean {
  req = (req || "").trim();
  if (!req || req.includes("不限")) return true;
  const subs = SUBJECTS.filter((s) => req.includes(s));
  if (subs.length === 0) return true;
  if (req.includes("或")) return subs.some((s) => chosen.has(s));
  return subs.every((s) => chosen.has(s));
}

// 紧凑定位索引项（与 Go locatorEntry 的 JSON 键一致）。
export interface LocEntry {
  sc: string; // 院校代码
  sn: string; // 院校名称
  gc: string; // 组代码
  gn: string; // 组名
  mn: string; // 专业名
  mk: string; // 专业键
  sk: string; // 选科要求
  pl: number; // 计划
  r: number; // 等效位次
  py: number; // 挂接年份
  gs: number; // 组内专业数
  mc?: string; // 学科门类 1 字码（见 src/lib/filters.ts CATEGORIES）
  tu?: number; // 学费（元/年，待定/无→省略）
  cw?: boolean; // 中外合作办学
}
