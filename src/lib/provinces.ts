// 省份配置：路由、科类、填报模型、选科模型的单一真相源。Astro 页面与 Preact island 共用。
// 与 Go cmd/zhiyuan-data/provinces.go 的 slug / 科类 / trackSlug / model 镜像，改一处要同步——
// 漂移会被 provinces.sync.test.ts 在 `npm test` 时拦下（安全网，非真相收敛）。
// 见 ADR-0009（多省份泛化）。

export type FillModel = "group" | "major"; // 黑龙江=院校专业组；浙江=院校×专业（专业平行志愿）
export type SubjectMode = "primary+reselect" | "pick3of7" | "pick3of6" | "wenli";
// 黑龙江=首选物理/历史+再选；浙江=7选3（含技术）；北京/上海/海南/山东=6选3（无技术）；
// 新疆=wenli（老高考 理科/文科，无选科，仅科类切换）

export interface TrackConf {
  name: string; // 科类名：物理 / 历史 / 综合
  slug: string; // 文件名片段：wuli / lishi / zonghe
}

// FillWindow：该省 2026 年「本科批 普通类 平行志愿」集中填报的时间窗（省情事实，非本站口径）。
// 纯展示字段——与 intro/batchLabel 一样只属前端，不进 Go 镜像、不参与 provinces.sync 比对。
// 分段填报（浙江一段/二段、山东多次）以「一段/第1次」为 start/end，其余段次写进 note。
export interface FillWindow {
  start: string; // 起始日期 ISO，如 "2026-06-29"
  end: string; // 截止日期 ISO，如 "2026-07-02"
  endTime?: string; // 截止时刻 "17:00"——别错过，强调用；缺省不显
  note?: string; // 段次 / 提前批合并 / 分轮 等简短中文备注
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
  fill?: FillWindow; // 2026 本科批普通类志愿填报时间窗（省情，纯展示；缺省 → 页面显「—」）
}

// 当前高考年份（「当年」的单一真相源）：列表页据此把已覆盖当年一分一段的省标绿。
// 数据落后于公告——新一年的一分一段陆续到位时，各省 fenduanYear 才逐个推进到此值。
export const CURRENT_GAOKAO_YEAR = 2026;

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
    fill: { start: "2026-07-01", end: "2026-07-05", endTime: "18:00", note: "本科批（第二阶段集中填报）" },
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
    fill: { start: "2026-06-29", end: "2026-06-30", endTime: "17:30", note: "普通类一段；二段 7/26–27" },
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
    fill: { start: "2026-06-28", end: "2026-07-02", endTime: "17:00", note: "第一阶段填本科院校专业组" },
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
    fenduanYear: 2026,
    intro: "湖南 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
    fill: { start: "2026-06-28", end: "2026-07-01", endTime: "17:00", note: "本科批集中填报；45 院校专业组" },
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
    fill: { start: "2026-06-25", end: "2026-07-01", endTime: "17:00", note: "本科批 7/1 截止；开始以系统开放为准" },
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
    fenduanYear: 2026,
    intro: "安徽 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
    fill: { start: "2026-07-04", end: "2026-07-07", endTime: "17:00", note: "普通本科批；提前批 6/29–7/1" },
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
    fenduanYear: 2026,
    intro: "广西 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
    fill: { start: "2026-06-29", end: "2026-07-03", endTime: "10:00", note: "本科普通批 6/29 15:00 起；40 组" },
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
    fill: { start: "2026-06-29", end: "2026-07-02", endTime: "17:00", note: "首次集中填报；本科普通批 7/2 止" },
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
    fill: { start: "2026-06-28", end: "2026-07-02", endTime: "18:00", note: "正式志愿统一填报窗" },
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
    fill: { start: "2026-06-30", end: "2026-07-03", endTime: "18:00", note: "本科批（第二段）；48 院校专业组" },
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
    fenduanYear: 2026,
    intro: "陕西 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
    fill: { start: "2026-06-25", end: "2026-06-30", endTime: "12:00", note: "本科批院校专业组；提前批另填" },
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
    fenduanYear: 2026,
    intro: "内蒙古 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
    fill: { start: "2026-06-30", end: "2026-07-04", endTime: "17:00", note: "改革后集中填报；含提前批/本科批" },
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
    fenduanYear: 2026,
    intro: "广东 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
    fill: { start: "2026-06-29", end: "2026-07-04", endTime: "16:00", note: "普通类 6/29 19:00 起；45 院校专业组" },
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
    fill: { start: "2026-07-03", end: "2026-07-06", endTime: "18:00", note: "普通类本科批常规志愿" },
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
    fenduanYear: 2026,
    intro: "宁夏 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
    fill: { start: "2026-06-25", end: "2026-07-01", endTime: "18:00", note: "本科批与提前批同窗填报" },
  },
  bj: {
    slug: "bj",
    name: "北京",
    tracks: [{ name: "综合", slug: "zonghe" }],
    fillModel: "group",
    subjectMode: "pick3of6",
    fenduanTrack: "综合",
    fenduanYear: 2026,
    intro: "北京 · 2025",
    batchLabel: "本科批 · 综合（院校专业组）",
    fill: { start: "2026-06-27", end: "2026-07-01", endTime: "17:00", note: "本科普通批；与提前批同窗" },
  },
  sh: {
    slug: "sh",
    name: "上海",
    tracks: [{ name: "综合", slug: "zonghe" }],
    fillModel: "group",
    subjectMode: "pick3of6",
    fenduanTrack: "综合",
    fenduanYear: 2026,
    intro: "上海 · 2025",
    batchLabel: "本科批 · 综合（院校专业组）",
    fill: { start: "2026-07-01", end: "2026-07-02", endTime: "17:00", note: "本科阶段集中填报；每日 8:00 起" },
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
    fill: { start: "2026-07-02", end: "2026-07-05", endTime: "17:30", note: "本科普通批 30 组；含特殊类型" },
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
    fenduanYear: 2026,
    intro: "江西 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
    fill: { start: "2026-06-30", end: "2026-07-04", endTime: "17:00", note: "本科批（第二次集中填报）" },
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
    fenduanYear: 2026,
    intro: "吉林 · 2025",
    batchLabel: "本科批 · 物理 / 历史（院校专业组）",
    fill: { start: "2026-06-28", end: "2026-07-02", endTime: "20:00", note: "本科批 50 组；每日 8:00–20:00" },
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
    fill: { start: "2026-06-26", end: "2026-07-01", endTime: "14:00", note: "首次集中填报；含提前批/本科批" },
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
    fenduanYear: 2026,
    intro: "重庆 · 2025",
    batchLabel: "本科批 · 物理 / 历史（专业平行志愿）",
    fill: { start: "2026-06-27", end: "2026-06-30", endTime: "18:00", note: "各批次统一时段填报" },
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
    fill: { start: "2026-06-28", end: "2026-07-02", endTime: "18:00", note: "各批次统一时段；96 专业平行志愿" },
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
    fenduanYear: 2026,
    intro: "辽宁 · 2025",
    batchLabel: "本科批 · 物理 / 历史（专业平行志愿）",
    fill: { start: "2026-06-19", end: "2026-06-30", endTime: "16:00", note: "全批次统一窗口；112「专业+学校」志愿" },
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
    fenduanYear: 2026,
    intro: "河北 · 2025",
    batchLabel: "本科批 · 物理 / 历史（专业平行志愿）",
    fill: { start: "2026-06-28", end: "2026-07-02", endTime: "17:00", note: "本科批一次集中填报" },
  },
  sd: {
    slug: "sd",
    name: "山东",
    tracks: [{ name: "综合", slug: "zonghe" }],
    fillModel: "major",
    subjectMode: "pick3of6",
    fenduanTrack: "综合",
    fenduanYear: 2026,
    intro: "山东 · 2025",
    batchLabel: "普通类一段/二段 · 综合（专业平行志愿）",
    fill: { start: "2026-07-05", end: "2026-07-07", endTime: "18:00", note: "常规批第 1 次（本科）；第 2 次 7/24–26" },
  },
  tj: {
    slug: "tj",
    name: "天津",
    tracks: [{ name: "综合", slug: "zonghe" }],
    fillModel: "group",
    subjectMode: "pick3of6",
    fenduanTrack: "综合",
    fenduanYear: 2026,
    intro: "天津 · 2025",
    batchLabel: "本科批 · 综合（院校专业组）",
    fill: { start: "2026-06-25", end: "2026-06-29", endTime: "17:00", note: "普通本科批 A/B 阶段同填" },
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
    fill: { start: "2026-06-25", end: "2026-07-03", endTime: "12:00", note: "老高考；本科批截止 7/3" },
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

// 高考满分（各省制度差异，集中一处便于一眼核对）：上海 660（语数外各 150 + 3 门选考各 70）、
// 海南为标准分制满分 900，其余新高考省份均为 750。表头据此显示「满分 X 分」，让高分位次有参照
// （如上海 600/660 ≈ 91 分位 → 全省第 1,250，不写明满分易被当成 750 制误读）。
const FULL_SCORE: Record<string, number> = { sh: 660, hain: 900 };
export function fullScoreOf(slug: string): number {
  return FULL_SCORE[slug] ?? 750;
}

// 录取分口径说明：上海官方把本科批最低分封顶在 580（高分段不披露，580 以上一律记 580/4096），
// 本站改用未封顶的「平均分/平均位次」作录取参考分（见 group3p12.ParseScoresAvg）——表头点明，
// 避免被误读为最低投档线。其余省份用真实最低分，无需说明。
const SCORE_BASIS_NOTE: Record<string, string> = {
  sh: "高分段最低分官方未公开，分数线按平均分口径",
};
export function scoreBasisNoteOf(slug: string): string | undefined {
  return SCORE_BASIS_NOTE[slug];
}

// trackSlug 取某省某科类的文件名片段。
export function trackSlugOf(cfg: ProvinceConfig, track: string): string {
  return cfg.tracks.find((t) => t.name === track)?.slug ?? track;
}

// mdLabel 把 ISO 日期 "2026-07-02" 取成中文「7月2日」（去掉前导零）。
function mdLabel(iso: string): string {
  const [, m, d] = iso.split("-");
  return `${parseInt(m, 10)}月${parseInt(d, 10)}日`;
}

// fillRange 把填报窗口排成紧凑展示串：「6月29日—7月2日」；同一天则只显一个「6月29日」。
// 用于省列表（一格）。截止时刻与备注另由页面取 fill.endTime / fill.note 拼。
export function fillRange(f: FillWindow): string {
  const a = mdLabel(f.start);
  const b = mdLabel(f.end);
  return a === b ? a : `${a}—${b}`;
}
