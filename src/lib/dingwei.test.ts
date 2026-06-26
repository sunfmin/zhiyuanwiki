import { describe, it, expect } from "vitest";
import { classify, selKeAllows } from "./dingwei";

describe("classify", () => {
  // R=10000 录取位次。
  it.each([
    ["明显好于线-保", 8000, 10000, "保"],
    ["边界0.90-保", 9000, 10000, "保"],
    ["约等于-稳", 10000, 10000, "稳"],
    ["略低于线-冲", 11000, 10000, "冲"],
    ["太冲-out", 12000, 10000, "out"],
    ["位次无效-out", 0, 10000, "out"],
  ])("%s", (_n, V, R, want) => {
    expect(classify(V as number, R as number)).toBe(want);
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
