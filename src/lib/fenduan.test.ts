import { describe, it, expect } from "vitest";
import { scoreToRank, rankToScore, type YiFenYiDuan } from "./fenduan";

// 取自官方 2026 物理类一分一段表顶部的真实形状（含"700以上"顶段，存为 score=700）。
const table: YiFenYiDuan = {
  province: "黑龙江",
  track: "物理",
  year: 2026,
  entries: [
    { score: 696, count: 3, cumulative: 40 },
    { score: 697, count: 7, cumulative: 37 },
    { score: 698, count: 6, cumulative: 30 },
    { score: 699, count: 7, cumulative: 24 },
    { score: 700, count: 17, cumulative: 17 },
  ],
};

describe("scoreToRank", () => {
  it.each([
    ["精确-顶段", 700, 17],
    ["精确-699", 699, 24],
    ["精确-最低段696", 696, 40],
    ["高于顶段-取顶段累计", 720, 17],
    ["缺失分-就近向上取698", 698, 30],
    ["低于最低段-取最低段", 690, 40],
  ])("%s", (_name, score, want) => {
    expect(scoreToRank(table, score as number)).toBe(want);
  });

  it("空表返回 null", () => {
    expect(scoreToRank({ ...table, entries: [] }, 600)).toBeNull();
  });
});

describe("rankToScore", () => {
  it.each([
    ["顶段内rank10-返回最高分", 10, 700],
    ["rank17边界-700", 17, 700],
    ["rank18落入699段", 18, 699],
    ["rank24边界-699", 24, 699],
    ["rank25落入698段", 25, 698],
    ["超过表底-返回最低分", 9999, 696],
  ])("%s", (_name, rank, want) => {
    expect(rankToScore(table, rank as number)).toBe(want);
  });
});
