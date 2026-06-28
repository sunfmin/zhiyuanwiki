package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// province 是一个省份的预处理配置：slug（数据目录/URL 段）、中文名、科类列表。
// 见 ADR-0002（省份×科类×年份 一等键）与 ADR-0009（多省份泛化）。
type province struct {
	slug   string   // 数据目录与 URL 段：hlj / zj
	name   string   // 黑龙江 / 浙江
	tracks []string // 科类：黑龙江=[物理,历史]；浙江=[综合]
}

var provinces = map[string]province{
	"hlj": {slug: "hlj", name: "黑龙江", tracks: []string{"物理", "历史"}},
	"zj":  {slug: "zj", name: "浙江", tracks: []string{"综合"}},
	"js":  {slug: "js", name: "江苏", tracks: []string{"物理", "历史"}},
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
		fmt.Fprintf(os.Stderr, "未知省份 %q（支持：hlj, zj）\n", slug)
		os.Exit(2)
	}
	return p
}

// srcDir 返回某省份在 src/data 下的输出目录。
func srcDir(out string, p province) string { return filepath.Join(out, p.slug) }

// pubDir 返回某省份在 public/data 下的输出目录。
func pubDir(pub string, p province) string { return filepath.Join(pub, p.slug) }
