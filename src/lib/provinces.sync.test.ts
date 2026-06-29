import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { PROVINCES } from "./provinces";

// 省份身份的「单一真相」目前有两个家：Go 管线 cmd/zhiyuan-data/provinces.go 与前端
// src/lib/provinces.ts。加新省/改科类/改 model/改 trackSlug 必须两边都改——本测试是**安全网**，
// 不是真相收敛：任一边漂移而另一边没跟，`npm test` 即红。真正把两边收敛成一处是后续工作（#33 之外另议）。
//
// 比对的只是「身份」字段：slug 集合、name、科类（顺序）、每科类的 trackSlug、填报 model。
// 前端专属的展示字段（subjectMode / fenduanTrack / fenduanYear / intro / batchLabel）不在此列——
// 它们本就只属于前端，不该出现在 Go 侧。

interface GoProvince {
  slug: string;
  name: string;
  tracks: string[];
  model: string;
}

function readGoSource(): string {
  const p = fileURLToPath(new URL("../../cmd/zhiyuan-data/provinces.go", import.meta.url));
  return readFileSync(p, "utf8");
}

// parseGoProvinces 从 provinces.go 抽出 `provinces` map 的每条身份。
// 形如：`"hlj": {slug: "hlj", name: "黑龙江", tracks: []string{"物理", "历史"}, model: "group"},`
// （行尾可带 // 注释；条目间的整行注释会被自然跳过。）
function parseGoProvinces(src: string): Record<string, GoProvince> {
  const entry =
    /"(\w+)":\s*\{slug:\s*"([^"]+)",\s*name:\s*"([^"]+)",\s*tracks:\s*\[\]string\{([^}]*)\},\s*model:\s*"([^"]+)"\}/g;
  const out: Record<string, GoProvince> = {};
  for (const m of src.matchAll(entry)) {
    const [, key, slug, name, tracksInner, model] = m;
    const tracks = [...tracksInner.matchAll(/"([^"]+)"/g)].map((t) => t[1]);
    out[key] = { slug, name, tracks, model };
  }
  return out;
}

// parseGoTrackSlug 抽出 `var trackSlug = map[string]string{ "物理": "wuli", ... }`。
function parseGoTrackSlug(src: string): Record<string, string> {
  const block = src.match(/var trackSlug = map\[string\]string\{([\s\S]*?)\}/);
  if (!block) throw new Error("provinces.go: 找不到 trackSlug map");
  const out: Record<string, string> = {};
  for (const m of block[1].matchAll(/"([^"]+)":\s*"(\w+)"/g)) out[m[1]] = m[2];
  return out;
}

// Go 的 model 有 major-zj（浙江一表联动专用投影），前端 FillModel 只分 group/major——
// major-zj 在前端就是 major。归一后比对。
function goModelToFill(model: string): string {
  return model === "group" ? "group" : "major";
}

const goSrc = readGoSource();
const goProv = parseGoProvinces(goSrc);
const goTrackSlug = parseGoTrackSlug(goSrc);

describe("province identity stays in sync (Go ↔ TS)", () => {
  it("解析出的 Go 省份非空（正则没失配）", () => {
    expect(Object.keys(goProv).length).toBeGreaterThan(20);
    expect(Object.keys(goTrackSlug).length).toBeGreaterThan(2);
  });

  it("slug 集合一致", () => {
    const go = Object.keys(goProv).sort();
    const ts = Object.keys(PROVINCES).sort();
    expect(go).toEqual(ts);
  });

  it("每省 name / 科类 / model 一致", () => {
    for (const slug of Object.keys(PROVINCES)) {
      const ts = PROVINCES[slug];
      const go = goProv[slug];
      expect(go, `Go 缺省份 ${slug}`).toBeDefined();
      if (!go) continue;
      expect(go.slug, `${slug}: Go map key 与 slug 字段不符`).toBe(slug);
      expect(ts.name, `${slug}: name 漂移`).toBe(go.name);
      expect(
        ts.tracks.map((t) => t.name),
        `${slug}: 科类（顺序）漂移`,
      ).toEqual(go.tracks);
      expect(ts.fillModel, `${slug}: 填报 model 漂移`).toBe(goModelToFill(go.model));
    }
  });

  it("每科类的 trackSlug 文件名片段一致", () => {
    for (const slug of Object.keys(PROVINCES)) {
      for (const t of PROVINCES[slug].tracks) {
        expect(
          goTrackSlug[t.name],
          `Go trackSlug 缺科类 ${t.name}（${slug} 用到）`,
        ).toBeDefined();
        expect(t.slug, `${slug}/${t.name}: trackSlug 漂移`).toBe(goTrackSlug[t.name]);
      }
    }
  });
});
