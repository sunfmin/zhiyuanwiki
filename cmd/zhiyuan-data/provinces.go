package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// province 是一个省份的预处理配置：slug（数据目录/URL 段）、中文名、科类列表。
// 见 ADR-0002（省份×科类×年份 一等键）与 ADR-0009（多省份泛化）。
type province struct {
	slug   string   // 数据目录与 URL 段：hlj / zj
	name   string   // 黑龙江 / 浙江
	tracks []string // 科类：黑龙江=[物理,历史]；浙江=[综合]
	model  string   // 填报模型：group=院校专业组（多数省）；major=专业平行志愿（浙江）。决定 yuanxiao 投影路径。
}

var provinces = map[string]province{
	"hlj": {slug: "hlj", name: "黑龙江", tracks: []string{"物理", "历史"}, model: "group"},
	"zj":  {slug: "zj", name: "浙江", tracks: []string{"综合"}, model: "major-zj"}, // 一表联动 by-code 属性，专用投影
	"js":  {slug: "js", name: "江苏", tracks: []string{"物理", "历史"}, model: "group"},
	"hn":  {slug: "hn", name: "湖南", tracks: []string{"物理", "历史"}, model: "group"},
	"sc":  {slug: "sc", name: "四川", tracks: []string{"物理", "历史"}, model: "group"},
	"ah":  {slug: "ah", name: "安徽", tracks: []string{"物理", "历史"}, model: "group"},
	// 干净的统一格式 3+1+2 group 省（2025 单年，配方同 sc/ah）；云南/河南 2025 为其改革首年。见 ADR-0014。
	// 江西暂缓：2025 计划表是原始省考院导出（列名 计划数/专业组/选课要求、艺普混表），且其专业组代码
	// 「第501组」与录取分数表的「（501）」不同形，直接挂接组 fill 会失败，需专属解析 + 组码归一，另案。
	"gx":    {slug: "gx", name: "广西", tracks: []string{"物理", "历史"}, model: "group"},
	"hb":    {slug: "hb", name: "湖北", tracks: []string{"物理", "历史"}, model: "group"},
	"yn":    {slug: "yn", name: "云南", tracks: []string{"物理", "历史"}, model: "group"},
	"henan": {slug: "henan", name: "河南", tracks: []string{"物理", "历史"}, model: "group"},
	// 陕西：与四川同形，直接复用 group3p12。
	"sx": {slug: "sx", name: "陕西", tracks: []string{"物理", "历史"}, model: "group"},
	// 综合(3+3)+院校专业组：北京/上海/海南 单科类「综合」、仍走 group 模型（有真实院校专业组）。
	"bj":   {slug: "bj", name: "北京", tracks: []string{"综合"}, model: "group"},
	"sh":   {slug: "sh", name: "上海", tracks: []string{"综合"}, model: "group"},
	"hain": {slug: "hain", name: "海南", tracks: []string{"综合"}, model: "group"},
	// 物理/历史 group（分数/计划文件名异常，靠 ScoreMust/PlanMust 精确指向 2025 文件）。
	"nm": {slug: "nm", name: "内蒙古", tracks: []string{"物理", "历史"}, model: "group"},
	"gd": {slug: "gd", name: "广东", tracks: []string{"物理", "历史"}, model: "group"},
	"fj": {slug: "fj", name: "福建", tracks: []string{"物理", "历史"}, model: "group"},
	"nx": {slug: "nx", name: "宁夏", tracks: []string{"物理", "历史"}, model: "group"},
	// 计划表列名异形（江西 计划数/专业组/选课要求；吉林仅专业组名称「第 001 组」；甘肃 文理/组代码），
	// 由 group3p12 的列别名 + 组名兜底覆盖，仍复用同一解析器（组码仅作展示，fill 按校+专业挂接）。
	"jx": {slug: "jx", name: "江西", tracks: []string{"物理", "历史"}, model: "group"},
	"jl": {slug: "jl", name: "吉林", tracks: []string{"物理", "历史"}, model: "group"},
	"gs": {slug: "gs", name: "甘肃", tracks: []string{"物理", "历史"}, model: "group"},
	// 专业平行志愿（无院校专业组）→ major 模型（通用 buildMajorBundle，全国 school 表挂属性，支持双科类）。
	// score/plan 仍复用 group3p12 解析，仅 build 走 major。重庆/贵州/辽宁/河北=物理/历史；山东=综合。
	"cq":    {slug: "cq", name: "重庆", tracks: []string{"物理", "历史"}, model: "major"},
	"gz":    {slug: "gz", name: "贵州", tracks: []string{"物理", "历史"}, model: "major"},
	"ln":    {slug: "ln", name: "辽宁", tracks: []string{"物理", "历史"}, model: "major"},
	"hebei": {slug: "hebei", name: "河北", tracks: []string{"物理", "历史"}, model: "major"},
	"sd":    {slug: "sd", name: "山东", tracks: []string{"综合"}, model: "major"},
	// 天津：综合+院校专业组（group），计划表院校代码由专业组代码剥后缀得，自定义 tj.ParsePlan。
	"tj": {slug: "tj", name: "天津", tracks: []string{"综合"}, model: "group"},
	// 新疆：老高考（老文理 理科/文科，专业平行志愿，无院校专业组）→ major 模型。专属 internal/xj 解析
	// （keep={理科,文科}，不并入 group3p12 默认 keep）。前端走 subjectMode="wenli"（无选科）。见 issue #27。
	"xj": {slug: "xj", name: "新疆", tracks: []string{"理科", "文科"}, model: "major"},
}

// trackSlug 把科类名映射成 ascii 文件名片段（定位索引/一分一段文件名）。
// 前端 src/lib/provinces.ts 有镜像，改这里要同步。
var trackSlug = map[string]string{
	"物理": "wuli",
	"历史": "lishi",
	"综合": "zonghe",
	"理科": "like", // 老文理（新疆）
	"文科": "wenke",
}

// mustProv 解析 -prov 值为省份配置；未知 slug 直接退出。
func mustProv(slug string) province {
	p, ok := provinces[slug]
	if !ok {
		slugs := make([]string, 0, len(provinces))
		for s := range provinces {
			slugs = append(slugs, s)
		}
		sort.Strings(slugs)
		fmt.Fprintf(os.Stderr, "未知省份 %q（支持：%s）\n", slug, strings.Join(slugs, ", "))
		os.Exit(2)
	}
	return p
}

// srcDir 返回某省份在 src/data 下的输出目录。
func srcDir(out string, p province) string { return filepath.Join(out, p.slug) }

// pubDir 返回某省份在 public/data 下的输出目录。
func pubDir(pub string, p province) string { return filepath.Join(pub, p.slug) }
