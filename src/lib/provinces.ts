// 省份配置：路由、科类、填报模型、选科模型的单一真相源。Astro 页面与 Preact island 共用。
// 与 Go cmd/zhiyuan-data/provinces.go 的 slug / 科类 / trackSlug 镜像，改一处要同步。
// 见 ADR-0009（多省份泛化）。

export type FillModel = "group" | "major"; // 黑龙江=院校专业组；浙江=院校×专业（专业平行志愿）
export type SubjectMode = "primary+reselect" | "pick3of7"; // 黑龙江=首选物理/历史+再选；浙江=7选3

export interface TrackConf {
  name: string; // 科类名：物理 / 历史 / 综合
  slug: string; // 文件名片段：wuli / lishi / zonghe
}

export interface ProvinceConfig {
  slug: string; // hlj / zj —— 数据目录与 URL 段
  name: string; // 黑龙江 / 浙江
  tracks: TrackConf[]; // 黑龙江两科类；浙江单科类「综合」
  fillModel: FillModel;
  subjectMode: SubjectMode;
  fenduanTrack: string; // 定位/换算所用一分一段表的科类
  fenduanYear: number; // 该表的年份（黑龙江 2026 物理；浙江 2025 综合）
  intro: string; // 主页/列表的省份说明片段
  batchLabel: string; // 数据口径标签
}

export const PROVINCES: Record<string, ProvinceConfig> = {
  hlj: {
    slug: "hlj",
    name: "黑龙江",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2026,
    intro: "黑龙江 · 2026",
    batchLabel: "本科批 · 物理 / 历史类",
  },
  zj: {
    slug: "zj",
    name: "浙江",
    tracks: [{ name: "综合", slug: "zonghe" }],
    fillModel: "major",
    subjectMode: "pick3of7",
    fenduanTrack: "综合",
    fenduanYear: 2025,
    intro: "浙江 · 普通类一段/二段",
    batchLabel: "普通类一段/二段 · 综合（专业平行志愿）",
  },
};

export const PROVINCE_SLUGS = Object.keys(PROVINCES);
export const DEFAULT_PROVINCE = "hlj";

export function provinceConfig(slug: string): ProvinceConfig {
  return PROVINCES[slug] ?? PROVINCES[DEFAULT_PROVINCE];
}

// trackSlug 取某省某科类的文件名片段。
export function trackSlugOf(cfg: ProvinceConfig, track: string): string {
  return cfg.tracks.find((t) => t.name === track)?.slug ?? track;
}
