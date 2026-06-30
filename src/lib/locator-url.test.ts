import { describe, it, expect } from "vitest";
import {
  encodeLocatorURL,
  decodeLocatorURL,
  hasLocatorParams,
  type LocatorSearch,
} from "./locator-url";
import { emptyFilters, type Filters } from "./filters";

const f = (over: Partial<Filters> = {}): Filters => ({ ...emptyFilters(), ...over });

const search = (over: Partial<LocatorSearch> = {}): LocatorSearch => ({
  track: "物理",
  mode: "score",
  val: "",
  subjects: [],
  filters: emptyFilters(),
  ...over,
});

const HLJ = { defaultTrack: "物理", multiTrack: true, wenli: false };
const ZJ = { defaultTrack: "综合", multiTrack: false, wenli: false };

const decode = (qs: string, tracks = ["物理", "历史"]) =>
  decodeLocatorURL(new URLSearchParams(qs), { tracks });

describe("encodeLocatorURL — 只输出有意义的维度", () => {
  it("纯默认空搜索 → 空串", () => {
    expect(encodeLocatorURL(search(), HLJ)).toBe("");
  });

  it("默认科类不输出 track；非默认才输出", () => {
    expect(encodeLocatorURL(search({ track: "物理" }), HLJ)).toBe("");
    expect(encodeLocatorURL(search({ track: "历史" }), HLJ)).toBe("track=%E5%8E%86%E5%8F%B2");
  });

  it("单科类省份永不输出 track", () => {
    expect(encodeLocatorURL(search({ track: "综合" }), ZJ)).toBe("");
  });

  it("分数模式 → score=；位次模式 → rank=", () => {
    expect(decode(encodeLocatorURL(search({ mode: "score", val: "650" }), HLJ))).toMatchObject({
      mode: "score",
      val: "650",
    });
    expect(decode(encodeLocatorURL(search({ mode: "rank", val: "12000" }), HLJ))).toMatchObject({
      mode: "rank",
      val: "12000",
    });
  });

  it("有输入值时输出选科；无输入值时不输出选科", () => {
    const params = new URLSearchParams(
      encodeLocatorURL(search({ val: "650", subjects: ["化学", "生物"] }), HLJ),
    );
    expect(params.get("subjects")).toBe("化学,生物");

    const noVal = new URLSearchParams(
      encodeLocatorURL(search({ val: "", subjects: ["化学", "生物"] }), HLJ),
    );
    expect(noVal.has("subjects")).toBe(false);
  });

  it("wenli 省份从不输出选科", () => {
    const params = new URLSearchParams(
      encodeLocatorURL(search({ track: "文科", val: "650", subjects: ["化学"] }), {
        defaultTrack: "理科",
        multiTrack: true,
        wenli: true,
      }),
    );
    expect(params.has("subjects")).toBe(false);
  });

  it("筛选维度按需输出，名称与 Filters 字段一致", () => {
    const params = new URLSearchParams(
      encodeLocatorURL(
        search({
          filters: f({
            provinces: ["北京", "上海"],
            levels: ["985"],
            categories: ["工", "理"],
            majorKeyword: "计算机 软件",
            minPlan: 10,
            hideCoopHighFee: true,
          }),
        }),
        HLJ,
      ),
    );
    expect(params.get("provinces")).toBe("北京,上海");
    expect(params.get("levels")).toBe("985");
    expect(params.get("categories")).toBe("工,理");
    expect(params.get("majorKeyword")).toBe("计算机 软件");
    expect(params.get("minPlan")).toBe("10");
    expect(params.get("hideCoopHighFee")).toBe("1");
    // 未设置的维度不出现
    expect(params.has("ownership")).toBe(false);
    expect(params.has("maxGroupSize")).toBe(false);
  });
});

describe("decodeLocatorURL — 解析与回落", () => {
  it("空 query → 仅默认 filters，其余维度缺省（交回默认）", () => {
    const d = decode("");
    expect(d.track).toBeUndefined();
    expect(d.mode).toBeUndefined();
    expect(d.val).toBeUndefined();
    expect(d.subjects).toBeUndefined();
    expect(d.filters).toEqual(emptyFilters());
  });

  it("非法 track（不在该省科类内）被丢弃", () => {
    expect(decode("track=综合").track).toBeUndefined();
    expect(decode("track=历史").track).toBe("历史");
  });

  it("score/rank 值规整：非数字或 ≤0 → 空串", () => {
    expect(decode("score=abc").val).toBe("");
    expect(decode("rank=0").val).toBe("");
    expect(decode("rank=-5").val).toBe("");
    expect(decode("score=650").val).toBe("650");
  });

  it("score 与 rank 同时出现 → score 优先", () => {
    expect(decode("score=650&rank=12000").mode).toBe("score");
  });

  it("subjects 显式空（subjects=）→ 空数组（区别于缺省 undefined）", () => {
    expect(decode("subjects=").subjects).toEqual([]);
    expect(decode("subjects=化学,生物").subjects).toEqual(["化学", "生物"]);
    expect(decode("rank=1").subjects).toBeUndefined();
  });

  it("数组维度按逗号拆分、去空白", () => {
    expect(decode("provinces=北京,%20上海").filters.provinces).toEqual(["北京", "上海"]);
  });

  it("数字维度负数归零、非数字归零", () => {
    expect(decode("minPlan=-3").filters.minPlan).toBe(0);
    expect(decode("maxGroupSize=abc").filters.maxGroupSize).toBe(0);
    expect(decode("minPlan=10").filters.minPlan).toBe(10);
  });

  it("hideCoopHighFee 仅 '1' 为真", () => {
    expect(decode("hideCoopHighFee=1").filters.hideCoopHighFee).toBe(true);
    expect(decode("hideCoopHighFee=true").filters.hideCoopHighFee).toBe(false);
  });
});

describe("hasLocatorParams — 区分带搜索 / 干净 URL", () => {
  it.each([
    ["空", "", false],
    ["无关参数", "foo=1&utm_source=x", false],
    ["score", "score=650", true],
    ["rank", "rank=1", true],
    ["track", "track=历史", true],
    ["筛选维度", "levels=985", true],
    ["关键词", "schoolKeyword=北大", true],
  ])("%s → %s", (_n, qs, want) => {
    expect(hasLocatorParams(new URLSearchParams(qs as string))).toBe(want);
  });
});

describe("round-trip — encode∘decode 还原可分享状态", () => {
  it.each<[string, LocatorSearch, typeof HLJ | typeof ZJ]>([
    ["黑龙江 分数+选科+多筛选", search({
      track: "历史",
      mode: "score",
      val: "650",
      subjects: ["化学", "生物"],
      filters: f({ provinces: ["北京"], levels: ["985", "211"], categories: ["工"], majorKeyword: "计算机", minPlan: 5, maxGroupSize: 3, hideCoopHighFee: true }),
    }), HLJ],
    ["浙江 位次+7选3", search({
      track: "综合",
      mode: "rank",
      val: "8000",
      subjects: ["物理", "化学", "生物"],
      filters: f({ schoolKeyword: "浙江大学", ownership: ["公办"] }),
    }), ZJ],
  ])("%s", (_n, s, opts) => {
    const qs = encodeLocatorURL(s, opts);
    const d = decodeLocatorURL(new URLSearchParams(qs), {
      tracks: opts.multiTrack ? ["物理", "历史"] : ["综合"],
    });
    // track：默认科类编码时省略，解码回 undefined（调用方回落到默认）。
    expect(d.track ?? opts.defaultTrack).toBe(s.track);
    expect(d.mode).toBe(s.mode);
    expect(d.val).toBe(s.val);
    expect(d.subjects).toEqual(s.subjects);
    expect(d.filters).toEqual(s.filters);
  });
});
