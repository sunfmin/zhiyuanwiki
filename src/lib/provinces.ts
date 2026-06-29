// 省份配置：路由、科类、填报模型、选科模型的单一真相源。Astro 页面与 Preact island 共用。
// 与 Go cmd/zhiyuan-data/provinces.go 的 slug / 科类 / trackSlug 镜像，改一处要同步。
// 见 ADR-0009（多省份泛化）。

export type FillModel = "group" | "major"; // 黑龙江=院校专业组；浙江=院校×专业（专业平行志愿）
export type SubjectMode = "primary+reselect" | "pick3of7" | "pick3of6" | "wenli";
// 黑龙江=首选物理/历史+再选；浙江=7选3（含技术）；北京/上海/海南/山东=6选3（无技术）；
// 新疆=wenli（老高考 理科/文科，无选科，仅科类切换）

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
  gx: {
    slug: "gx",
    name: "广西",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "广西 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
  hb: {
    slug: "hb",
    name: "湖北",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "湖北 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
  yn: {
    slug: "yn",
    name: "云南",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "云南 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
  henan: {
    slug: "henan",
    name: "河南",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "河南 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
  sx: {
    slug: "sx",
    name: "陕西",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "陕西 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
  nm: {
    slug: "nm",
    name: "内蒙古",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "内蒙古 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
  gd: {
    slug: "gd",
    name: "广东",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "广东 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
  fj: {
    slug: "fj",
    name: "福建",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "福建 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
  nx: {
    slug: "nx",
    name: "宁夏",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "宁夏 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
  bj: {
    slug: "bj",
    name: "北京",
    tracks: [{ name: "综合", slug: "zonghe" }],
    fillModel: "group",
    subjectMode: "pick3of6",
    fenduanTrack: "综合",
    fenduanYear: 2025,
    intro: "北京 · 2025",
    batchLabel: "本科批 · 综合（院校专业组）",
  },
  sh: {
    slug: "sh",
    name: "上海",
    tracks: [{ name: "综合", slug: "zonghe" }],
    fillModel: "group",
    subjectMode: "pick3of6",
    fenduanTrack: "综合",
    fenduanYear: 2025,
    intro: "上海 · 2025",
    batchLabel: "本科批 · 综合（院校专业组）",
  },
  hain: {
    slug: "hain",
    name: "海南",
    tracks: [{ name: "综合", slug: "zonghe" }],
    fillModel: "group",
    subjectMode: "pick3of6",
    fenduanTrack: "综合",
    fenduanYear: 2025,
    intro: "海南 · 2025",
    batchLabel: "本科批 · 综合（院校专业组）",
  },
  jx: {
    slug: "jx",
    name: "江西",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "江西 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
  jl: {
    slug: "jl",
    name: "吉林",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "吉林 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
  gs: {
    slug: "gs",
    name: "甘肃",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "group",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "甘肃 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
  },
  cq: {
    slug: "cq",
    name: "重庆",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "major",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "重庆 · 2025",
    batchLabel: "本科批 · 物理 / 历史（专业平行志愿）",
  },
  gz: {
    slug: "gz",
    name: "贵州",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "major",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "贵州 · 2025",
    batchLabel: "本科批 · 物理 / 历史（专业平行志愿）",
  },
  ln: {
    slug: "ln",
    name: "辽宁",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "major",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "辽宁 · 2025",
    batchLabel: "本科批 · 物理 / 历史（专业平行志愿）",
  },
  hebei: {
    slug: "hebei",
    name: "河北",
    tracks: [
      { name: "物理", slug: "wuli" },
      { name: "历史", slug: "lishi" },
    ],
    fillModel: "major",
    subjectMode: "primary+reselect",
    fenduanTrack: "物理",
    fenduanYear: 2025,
    intro: "河北 · 2025",
    batchLabel: "本科批 · 物理 / 历史（专业平行志愿）",
  },
  sd: {
    slug: "sd",
    name: "山东",
    tracks: [{ name: "综合", slug: "zonghe" }],
    fillModel: "major",
    subjectMode: "pick3of6",
    fenduanTrack: "综合",
    fenduanYear: 2025,
    intro: "山东 · 2025",
    batchLabel: "普通类一段/二段 · 综合（专业平行志愿）",
  },
  tj: {
    slug: "tj",
    name: "天津",
    tracks: [{ name: "综合", slug: "zonghe" }],
    fillModel: "group",
    subjectMode: "pick3of6",
    fenduanTrack: "综合",
    fenduanYear: 2025,
    intro: "天津 · 2025",
    batchLabel: "本科批 · 综合（院校专业组）",
  },
  xj: {
    slug: "xj",
    name: "新疆",
    tracks: [
      { name: "理科", slug: "like" },
      { name: "文科", slug: "wenke" },
    ],
    fillModel: "major",
    subjectMode: "wenli", // 老高考：无选科，仅理科/文科切换
    fenduanTrack: "理科", // 理科有 2025 一分一段可做分数↔位次；文科直接输位次
    fenduanYear: 2025,
    intro: "新疆 · 2025（老高考）",
    batchLabel: "本科批 · 理科 / 文科（专业平行志愿）",
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
