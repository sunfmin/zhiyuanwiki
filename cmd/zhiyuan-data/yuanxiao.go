package main

import (
	"flag"
	"path/filepath"
)

// yuanxiaoCmd 从 SQLite staging 投影院校 / 院校×专业叶子 / 2026 报考视图 JSON（按省份分目录）。
// 全部省份都已入库（ADR-0014），按填报模型选投影：group→buildDBBundle（院校专业组）、
// major→buildDBBundleMajor（专业平行志愿，浙江）。无 per-slug 硬编码。
func yuanxiaoCmd(args []string) {
	fs := flag.NewFlagSet("yuanxiao", flag.ExitOnError)
	out := fs.String("out", filepath.Join("src", "data"), "JSON 输出目录（其下按省份 slug 分目录）")
	pub := fs.String("pub", filepath.Join("public", "data"), "客户端公开数据目录（其下按省份 slug 分目录）")
	dbPath := fs.String("db", filepath.Join("out", "zhiyuan.db"), "SQLite staging 库")
	provSlug := fs.String("prov", "hlj", "省份 slug：hlj / zj / js …")
	_ = fs.Parse(args)
	p := mustProv(*provSlug)

	var b schoolBundle
	switch p.model {
	case "major-zj": // 浙江：一表联动 by-code 属性（含 city_tier）+ 单科类综合，见 #21
		b = buildDBBundleMajor(*dbPath, p)
	case "major": // 专业平行志愿（无院校专业组）：全国 school 表按校名挂属性，支持双科类（重庆/辽宁…）
		b = buildMajorBundle(*dbPath, p)
	default: // group
		b = buildDBBundle(*dbPath, p)
	}
	emitSchoolData(p, b, srcDir(*out, p), pubDir(*pub, p))
}
