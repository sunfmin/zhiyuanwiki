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
	"zj":  {slug: "zj", name: "浙江", tracks: []string{"综合"}, model: "major"},
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
}

// trackSlug 把科类名映射成 ascii 文件名片段（定位索引/一分一段文件名）。
// 前端 src/lib/provinces.ts 有镜像，改这里要同步。
var trackSlug = map[string]string{
	"物理": "wuli",
	"历史": "lishi",
	"综合": "zonghe",
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
