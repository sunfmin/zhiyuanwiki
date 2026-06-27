import { describe, it, expect } from "vitest";
import { classify, bucketize, selKeAllows, type LocEntry } from "./dingwei";

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
