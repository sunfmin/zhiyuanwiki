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
  fenduanYear: number; // 该表的年份（黑龙江 2026 物理；浙江 2026 综合）
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
    fenduanYear: 2026,
    intro: "浙江 · 普通类一段/二段",
    batchLabel: "普通类一段/二段 · 综合（专业平行志愿）",
  },
  js: {
    slug: "js",
    name: "江苏",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "江苏 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
  hn: {
    slug: "hn",
    name: "湖南",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "湖南 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
  sc: {
    slug: "sc",
    name: "四川",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "四川 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
  ah: {
    slug: "ah",
    name: "安徽",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "安徽 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
};

export const PROVINCE_SLUGS = Object.keys(PROVINCES);
export const DEFAULT_PROVINCE = "hlj";

// 全国 31 省级行政区花名册（港澳台不计入高考统招）。落地页名单与 Base 省份切换器共用的
// 单一真相源——省名 + 拼音（用于落地页「敬请期待」段按拼音 A→Z 排序）。是否已上线由
// PROVINCES（已配置数据的省）派生，见 liveSlugByName。见 ADR-0016。
export interface RosterEntry {
  name: string; // 中文省名
  pinyin: string; // 全拼，用于 A→Z 排序（山西 shanxi / 陕西 shaanxi 天然有别）
}

export const PROVINCE_ROSTER: RosterEntry[] = [
  { name: "北京", pinyin: "beijing" },
  { name: "天津", pinyin: "tianjin" },
  { name: "河北", pinyin: "hebei" },
  { name: "山西", pinyin: "shanxi" },
  { name: "内蒙古", pinyin: "neimenggu" },
  { name: "辽宁", pinyin: "liaoning" },
  { name: "吉林", pinyin: "jilin" },
  { name: "黑龙江", pinyin: "heilongjiang" },
  { name: "上海", pinyin: "shanghai" },
  { name: "江苏", pinyin: "jiangsu" },
  { name: "浙江", pinyin: "zhejiang" },
  { name: "安徽", pinyin: "anhui" },
  { name: "福建", pinyin: "fujian" },
  { name: "江西", pinyin: "jiangxi" },
  { name: "山东", pinyin: "shandong" },
  { name: "河南", pinyin: "henan" },
  { name: "湖北", pinyin: "hubei" },
  { name: "湖南", pinyin: "hunan" },
  { name: "广东", pinyin: "guangdong" },
  { name: "广西", pinyin: "guangxi" },
  { name: "海南", pinyin: "hainan" },
  { name: "重庆", pinyin: "chongqing" },
  { name: "四川", pinyin: "sichuan" },
  { name: "贵州", pinyin: "guizhou" },
  { name: "云南", pinyin: "yunnan" },
  { name: "西藏", pinyin: "xizang" },
  { name: "陕西", pinyin: "shaanxi" },
  { name: "甘肃", pinyin: "gansu" },
  { name: "青海", pinyin: "qinghai" },
  { name: "宁夏", pinyin: "ningxia" },
  { name: "新疆", pinyin: "xinjiang" },
];

// liveSlugByName 把已上线省名映射到其 slug（落地页据此决定行是否可点进 /[slug]/）。
export function liveSlugByName(): Map<string, string> {
  return new Map(Object.values(PROVINCES).map((p) => [p.name, p.slug]));
}

export function provinceConfig(slug: string): ProvinceConfig {
  return PROVINCES[slug] ?? PROVINCES[DEFAULT_PROVINCE];
}

// trackSlug 取某省某科类的文件名片段。
export function trackSlugOf(cfg: ProvinceConfig, track: string): string {
  return cfg.tracks.find((t) => t.name === track)?.slug ?? track;
}
