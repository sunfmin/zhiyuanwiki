package main

import (
	"flag"
	"fmt"
	"path/filepath"
)

// yuanxiaoCmd 解析专业录取分数线 → 院校 / 院校×专业叶子 / 2026 报考视图 JSON（按省份分目录）。
func yuanxiaoCmd(args []string) {
	fs := flag.NewFlagSet("yuanxiao", flag.ExitOnError)
	src := fs.String("src", defaultSrc(), "官方数据根目录")
	out := fs.String("out", filepath.Join("src", "data"), "JSON 输出目录（其下按省份 slug 分目录）")
	pub := fs.String("pub", filepath.Join("public", "data"), "客户端公开数据目录（其下按省份 slug 分目录）")
	dbPath := fs.String("db", filepath.Join("out", "zhiyuan.db"), "SQLite staging 库（DB 投影省份用）")
	provSlug := fs.String("prov", "hlj", "省份 slug：hlj / zj / js")
	_ = fs.Parse(args)
	p := mustProv(*provSlug)

	var b schoolBundle
	switch {
	case p.slug == "zj":
		b = buildZJBundle(*src) // 浙江仍为 legacy（#21 退役，改走 buildDBBundleMajor）
	default: // 构建期 staging 管线省份（js/hn/sc/ah/hlj…）：从 SQLite 投影，见 ADR-0014
		if _, ok := provParsers[p.slug]; !ok {
			fatal(fmt.Errorf("yuanxiao 暂未支持省份 %q", p.slug))
		}
		b = buildDBBundle(*dbPath, p)
	}
	emitSchoolData(p, b, srcDir(*out, p), pubDir(*pub, p))
}
