import { describe, it, expect } from "vitest";
import { matchesFilters, emptyFilters, anyActive, type SchoolMetaMap } from "./filters";
import type { LocEntry } from "./dingwei";

// 测试夹具：两所院校（北京 985 公办综合 / 黑龙江民办理工 二线），若干定位结果。
const meta: SchoolMetaMap = {
  "1001": { p: "北京", c: "北京市", ct: "一线", o: "公办", k: "综合类", lv: ["985", "211", "双一流"] },
  "2002": { p: "黑龙江", c: "哈尔滨市", ct: "新一线", o: "民办", k: "理工类" },
  // 3003 故意不在 meta 中（覆盖率 ~91%，未挂接院校 → 院校级维度「未知」）。
};

function entry(over: Partial<LocEntry> = {}): LocEntry {
  return {
    sc: "1001", sn: "北京大学", gc: "001", gn: "第001组",
    mn: "计算机科学与技术", mk: "abc", sk: "化学", pl: 10, r: 100, py: 2025, gs: 3,
    mc: "工", tu: 5000, cw: false,
    ...over,
  };
}

const f = (over: Partial<ReturnType<typeof emptyFilters>> = {}) => ({ ...emptyFilters(), ...over });

describe("matchesFilters — 单维度命中/不命中", () => {
  it.each([
    ["省份命中", entry({ sc: "1001" }), f({ provinces: ["北京"] }), true],
    ["省份不命中", entry({ sc: "1001" }), f({ provinces: ["上海"] }), false],
    ["层次命中（985）", entry({ sc: "1001" }), f({ levels: ["985"] }), true],
    ["层次不命中（无211校选211）", entry({ sc: "2002" }), f({ levels: ["211"] }), false],
    ["办学性质-民办命中", entry({ sc: "2002" }), f({ ownership: ["民办"] }), true],
    ["学校类别命中", entry({ sc: "2002" }), f({ kinds: ["理工类"] }), true],
    ["城市层级命中", entry({ sc: "1001" }), f({ cityTiers: ["一线"] }), true],
    ["门类命中", entry({ mc: "工" }), f({ categories: ["工"] }), true],
    ["门类不命中", entry({ mc: "工" }), f({ categories: ["医", "理"] }), false],
  ])("%s", (_n, e, flt, want) => {
    expect(matchesFilters(e as LocEntry, meta, flt)).toBe(want);
  });
});

describe("matchesFilters — 维度间 AND", () => {
  const e = entry({ sc: "1001", mc: "工" }); // 北京·985·工学
  it("两维都命中 → 通过", () => {
    expect(matchesFilters(e, meta, f({ provinces: ["北京"], categories: ["工"] }))).toBe(true);
  });
  it("一维命中一维不命中 → 不通过", () => {
    expect(matchesFilters(e, meta, f({ provinces: ["北京"], categories: ["医"] }))).toBe(false);
  });
});

describe("matchesFilters — 单维度多选内部 OR", () => {
  it("省份多选含本省 → 通过", () => {
    expect(matchesFilters(entry({ sc: "2002" }), meta, f({ provinces: ["北京", "黑龙江"] }))).toBe(true);
  });
  it("层次多选任一命中 → 通过", () => {
    // 2002 无任何层次；选 985/211 都不命中 → 排除
    expect(matchesFilters(entry({ sc: "2002" }), meta, f({ levels: ["985", "211"] }))).toBe(false);
    // 1001 含 211 → 命中
    expect(matchesFilters(entry({ sc: "1001" }), meta, f({ levels: ["211", "双一流"] }))).toBe(true);
  });
});

describe("matchesFilters — 未知（未挂接 meta）", () => {
  const unknown = entry({ sc: "3003" }); // 不在 meta
  it("不筛该维度 → 放行", () => {
    expect(matchesFilters(unknown, meta, f({ categories: ["工"] }))).toBe(true);
  });
  it("显式筛省份 → 未知被排除", () => {
    expect(matchesFilters(unknown, meta, f({ provinces: ["北京"] }))).toBe(false);
  });
  it("显式筛层次 → 未知被排除", () => {
    expect(matchesFilters(unknown, meta, f({ levels: ["985"] }))).toBe(false);
  });
});

describe("matchesFilters — 专业关键词（空格=OR、大小写、子串）", () => {
  it.each([
    ["子串命中", entry({ mn: "计算机科学与技术" }), "计算机", true],
    ["不命中", entry({ mn: "临床医学" }), "计算机", false],
    ["大小写不敏感", entry({ mn: "AI与大数据" }), "ai", true],
    ["空关键词放行", entry({ mn: "随便" }), "  ", true],
    ["空格分隔-命中其一即可（OR）", entry({ mn: "软件工程" }), "计算机 软件", true],
    ["空格分隔-多词都不命中", entry({ mn: "临床医学" }), "计算机 软件", false],
    ["多个空格/前后空格-正常分词", entry({ mn: "网络空间安全" }), "  计算机   网络 ", true],
  ])("%s", (_n, e, kw, want) => {
    expect(matchesFilters(e as LocEntry, meta, f({ majorKeyword: kw as string }))).toBe(want);
  });
});

describe("matchesFilters — 院校关键词（同语义，匹配院校名 e.sn）", () => {
  it.each([
    ["子串命中", entry({ sn: "北京大学" }), "北京", true],
    ["不命中", entry({ sn: "北京大学" }), "清华", false],
    ["大小写不敏感", entry({ sn: "MIT 学院" }), "mit", true],
    ["空关键词放行", entry({ sn: "随便大学" }), "  ", true],
    ["空格分隔-命中其一即可（OR）", entry({ sn: "哈尔滨工业大学" }), "北京 哈尔滨", true],
    ["空格分隔-多词都不命中", entry({ sn: "复旦大学" }), "北京 哈尔滨", false],
  ])("%s", (_n, e, kw, want) => {
    expect(matchesFilters(e as LocEntry, meta, f({ schoolKeyword: kw as string }))).toBe(want);
  });
});

describe("matchesFilters — 院校关键词 AND 专业关键词（两框正交，框间 AND）", () => {
  const e = entry({ sn: "北京大学", mn: "计算机科学与技术" }); // 北大·计算机
  it.each([
    ["两框都命中 → 通过", "北京", "计算机", true],
    ["院校中专业不中 → 不通过", "北京", "临床", false],
    ["专业中院校不中 → 不通过", "清华", "计算机", false],
    ["两框都不中 → 不通过", "清华", "临床", false],
  ])("%s", (_n, sk, mk, want) => {
    expect(matchesFilters(e, meta, f({ schoolKeyword: sk as string, majorKeyword: mk as string }))).toBe(want);
  });
});

describe("matchesFilters — 计划下限 / 组内上限 边界", () => {
  it("计划下限：等于阈值通过、低于不通过", () => {
    expect(matchesFilters(entry({ pl: 10 }), meta, f({ minPlan: 10 }))).toBe(true);
    expect(matchesFilters(entry({ pl: 9 }), meta, f({ minPlan: 10 }))).toBe(false);
  });
  it("组内上限：等于阈值通过、超过不通过", () => {
    expect(matchesFilters(entry({ gs: 5 }), meta, f({ maxGroupSize: 5 }))).toBe(true);
    expect(matchesFilters(entry({ gs: 6 }), meta, f({ maxGroupSize: 5 }))).toBe(false);
  });
});

describe("matchesFilters — 隐藏中外合作及高收费（中外 OR 学费≥2万；待定不隐藏）", () => {
  const flt = f({ hideCoopHighFee: true });
  it.each([
    ["普通低学费 → 留", entry({ cw: false, tu: 5000 }), true],
    ["中外合作 → 隐", entry({ cw: true, tu: 5000 }), false],
    ["高收费≥2万 → 隐", entry({ cw: false, tu: 20000 }), false],
    ["学费待定（tu省略=0）→ 留", entry({ cw: false, tu: undefined }), true],
    ["1.9万未达阈值 → 留", entry({ cw: false, tu: 19000 }), true],
  ])("%s", (_n, e, want) => {
    expect(matchesFilters(e as LocEntry, meta, flt)).toBe(want);
  });
});

describe("anyActive / emptyFilters", () => {
  it("空过滤 → 无生效", () => {
    expect(anyActive(emptyFilters())).toBe(false);
  });
  it.each([
    ["省份", f({ provinces: ["北京"] })],
    ["专业关键词", f({ majorKeyword: "x" })],
    ["院校关键词", f({ schoolKeyword: "x" })],
    ["计划下限", f({ minPlan: 5 })],
    ["隐藏开关", f({ hideCoopHighFee: true })],
  ])("%s 生效 → true", (_n, flt) => {
    expect(anyActive(flt)).toBe(true);
  });
});
