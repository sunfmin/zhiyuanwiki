import { describe, it, expect } from "vitest";
import {
  classify,
  reachColor,
  bucketize,
  assembleColumns,
  selKeAllows,
  type LocEntry,
  type BucketGroups,
} from "./dingwei";

describe("classify", () => {
  // R=10000 录取位次。
  it.each([
    ["过保：好太多-浪费位次", 7000, 10000, "过保"],
    ["边界0.75-保", 7500, 10000, "保"],
    ["明显好于线-保", 8000, 10000, "保"],
    ["边界0.90-保", 9000, 10000, "保"],
    ["约等于-稳", 10000, 10000, "稳"],
    ["略低于线-冲", 11000, 10000, "冲"],
    ["边界1.15-冲", 11500, 10000, "冲"],
    ["太难-够不着", 11600, 10000, "够不着"],
    ["太难-够不着(大差)", 23000, 10000, "够不着"],
    ["位次无效-null", 0, 10000, null],
    ["录取位次无效-null", 10000, 0, null],
  ])("%s", (_n, V, R, want) => {
    expect(classify(V as number, R as number)).toBe(want);
  });
});

describe("classify 顶尖段兜底（floor）", () => {
  // 不传 floor（=0）时与原比值口径逐位等价：顶尖段 V 极小、R 远大 → 仍判过保。
  it("floor 默认 0：行为与原比值口径一致", () => {
    expect(classify(500, 1000)).toBe("过保"); // 500/1000=0.5 <0.75
    expect(classify(500, 1000, 0)).toBe(classify(500, 1000));
  });

  // floor 接管（V<floor）：把「优于均值不多」的专业从过保救回稳/保，填满冲/稳/保。
  it("V<floor：原本塌进过保的近档被救回保", () => {
    // V=500,R=1000,floor=2000 → effR=500+(1000-500)*(500/2000)=625，ratio=0.8 → 保（原为过保）。
    expect(classify(500, 1000)).toBe("过保");
    expect(classify(500, 1000, 2000)).toBe("保");
  });

  it("floor 不会把远易专业也救起：差距足够大仍判过保", () => {
    // V=500,R=2000,floor=2000 → effR=875，ratio≈0.571 <0.75 → 仍过保。
    expect(classify(500, 2000, 2000)).toBe("过保");
  });

  it("V≥floor：兜底不介入，与不传 floor 完全相同", () => {
    expect(classify(3000, 6000, 2000)).toBe(classify(3000, 6000)); // 都为过保
    expect(classify(3000, 3000, 2000)).toBe(classify(3000, 3000)); // 都为稳
  });

  it("bucketize 带 floor：顶尖段不再整列塌空", () => {
    // V=500（顶尖段）面对一批更易的专业：无 floor 全进过保；floor=2000 时近档落入冲/稳/保。
    const cand = [e(900), e(1000), e(1200), e(1500), e(3000)];
    const noFloor = bucketize(500, cand);
    expect(noFloor.冲.length + noFloor.稳.length + noFloor.保.length).toBe(0); // 全过保
    const withFloor = bucketize(500, cand, 2000);
    expect(withFloor.冲.length + withFloor.稳.length + withFloor.保.length).toBeGreaterThan(0);
  });
});

describe("reachColor 与 classify 共享阈值", () => {
  // R=10000。配色频谱：≤1.02 easy（稳得住）/ ≤1.08 mid（较易冲）/ >1.08 hard（偏难·够不着）。
  it.each([
    ["稳区→easy", 9000, 10000, "easy"],
    ["持平→easy", 10000, 10000, "easy"],
    ["边界1.02→easy", 10200, 10000, "easy"],
    ["较易冲→mid", 10500, 10000, "mid"],
    ["边界1.08→mid", 10800, 10000, "mid"],
    ["偏难冲→hard", 11000, 10000, "hard"],
    ["够不着→hard", 13000, 10000, "hard"],
    ["R无效→easy（沿用旧行为）", 10000, 0, "easy"],
  ])("%s", (_n, V, R, want) => {
    expect(reachColor(V as number, R as number)).toBe(want);
  });

  it("wenMax 边界与 classify 一致：刚过 1.02 既是「冲」也是「mid」（不再是 easy）", () => {
    // ratio 1.03：classify→冲；reachColor→mid。两者拐点同源，不会再各说各话。
    expect(classify(10300, 10000)).toBe("冲");
    expect(reachColor(10300, 10000)).toBe("mid");
  });
});

// 构造定位条目：只有 r（等效位次）影响分档，其余给占位值。
const e = (r: number): LocEntry => ({ sc: "1", sn: "x", mn: "m", mk: "k", sk: "", pl: 1, r, py: 2025 });

describe("bucketize", () => {
  it("稀疏集不凑数：有几条显几条，不向上填满", () => {
    // 只有 3 条候选，各落一档（R<V=更难=冲；R>V=更易=保）——绝不像旧 WINDOW 法凑到 100。
    // V=10000：r9000→1.11冲、r10000→稳、r12000→0.83保。
    const g = bucketize(10000, [e(9000), e(10000), e(12000)]);
    expect(g.冲.length).toBe(1);
    expect(g.稳.length).toBe(1);
    expect(g.保.length).toBe(1);
    expect(g.够不着.length).toBe(0);
    expect(g.过保.length).toBe(0);
  });

  it("远档不误标成冲/保：高你 13000 位进够不着，而非冲", () => {
    // 30 万考生里"高你 13000 位"= r 远好于 V → 够不着，不是冲。
    const g = bucketize(20000, [e(7000) /* 远难 */, e(40000) /* 远易 */]);
    expect(g.冲.length).toBe(0);
    expect(g.保.length).toBe(0);
    expect(g.够不着.map((x) => x.r)).toEqual([7000]);
    expect(g.过保.map((x) => x.r)).toEqual([40000]);
  });

  it("每档最贴近本人位次在前（按 |R−V| 升序）", () => {
    // V=10000，两条冲（R<V）：r9000(差1000) 比 r8800(差1200) 更贴近，应排前。故意逆序传入。
    const g = bucketize(10000, [e(8800), e(9000)]);
    expect(g.冲.map((x) => x.r)).toEqual([9000, 8800]);
  });

  it("位次无效 → 全空", () => {
    const g = bucketize(0, [e(10000)]);
    expect(Object.values(g).every((l) => l.length === 0)).toBe(true);
  });
});

describe("selKeAllows", () => {
  const wuhuasheng = new Set(["物理", "化学", "生物"]);
  it.each([
    ["不限", true],
    ["化学", true],
    ["化学和生物", true],
    ["化学或生物", true],
    ["政治", false],
    ["政治或地理", false],
    ["", true],
  ])("物化生 vs %s", (req, want) => {
    expect(selKeAllows(req as string, wuhuasheng)).toBe(want);
  });

  it("物化生不能报要求地理的组", () => {
    expect(selKeAllows("地理", wuhuasheng)).toBe(false);
  });
});

describe("assembleColumns", () => {
  // n 条占位条目；base 错开 r 以免与其它桶撞键（assembleColumns 只看数量，不看 r）。
  const mk = (n: number, base = 1000): LocEntry[] => Array.from({ length: n }, (_, i) => e(base + i));
  const groups = (g: Partial<BucketGroups>): BucketGroups => ({
    够不着: [], 冲: [], 稳: [], 保: [], 过保: [], ...g,
  });

  it("稳列永远无远档", () => {
    const cols = assembleColumns(groups({ 稳: mk(3), 够不着: mk(50, 5000), 过保: mk(50, 9000) }), { cap: 30, target: 100 });
    expect(cols.稳.far).toBeNull();
  });

  it("冲列稀疏：主档只装真档（不凑数），末尾用够不着补齐到 target", () => {
    const cols = assembleColumns(groups({ 冲: mk(3), 够不着: mk(200, 5000) }), { cap: 30, target: 100 });
    expect(cols.冲.all.length).toBe(3); // 主档不被远档撑大
    expect(cols.冲.capped.length).toBe(3); // 少于 cap，不截断
    expect(cols.冲.hasMore).toBe(false);
    expect(cols.冲.far?.bucket).toBe("够不着");
    expect(cols.冲.far?.entries.length).toBe(97); // 100 - 3
  });

  it("保列稀疏：末尾用过保补齐到 target", () => {
    const cols = assembleColumns(groups({ 保: mk(10), 过保: mk(200, 5000) }), { cap: 30, target: 100 });
    expect(cols.保.far?.bucket).toBe("过保");
    expect(cols.保.far?.entries.length).toBe(90); // 100 - 10
  });

  it("真实档 ≥ target 则不补远档", () => {
    const cols = assembleColumns(groups({ 冲: mk(120), 够不着: mk(50, 5000) }), { cap: 30, target: 100 });
    expect(cols.冲.far).toBeNull();
  });

  it("密集：收起态截断到 cap，hasMore 为真，all 仍保留全部", () => {
    const cols = assembleColumns(groups({ 冲: mk(50) }), { cap: 30, target: 100 });
    expect(cols.冲.all.length).toBe(50);
    expect(cols.冲.capped.length).toBe(30);
    expect(cols.冲.hasMore).toBe(true);
  });

  it("远档桶为空则不挂（即便未达 target）", () => {
    const cols = assembleColumns(groups({ 冲: mk(3) }), { cap: 30, target: 100 });
    expect(cols.冲.far).toBeNull();
  });
});
