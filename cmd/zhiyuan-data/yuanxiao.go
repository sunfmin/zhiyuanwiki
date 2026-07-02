package main

import (
	"flag"
	"path/filepath"

	"github.com/sunfmin/zhiyuanwiki/internal/store"
)

// yuanxiaoCmd 从 SQLite staging 投影院校 / 院校×专业叶子 / 2026 报考视图 JSON（按省份分目录）。
// 全部省份都已入库（ADR-0014），共享 buildBundle 骨架、按填报模型（ADR-0017）在此缝处选投影核心：
// group→projectGroups（院校专业组）、major→projectPlanMajors（专业平行志愿，含浙江/山东的综合与
// 重庆/辽宁的双科类）。DB 在此打开一次、注入 buildBundle。无 per-slug 硬编码。
func yuanxiaoCmd(args []string) {
	fs := flag.NewFlagSet("yuanxiao", flag.ExitOnError)
	out := fs.String("out", filepath.Join("src", "data"), "JSON 输出目录（其下按省份 slug 分目录）")
	pub := fs.String("pub", filepath.Join("public", "data"), "客户端公开数据目录（其下按省份 slug 分目录）")
	dbPath := fs.String("db", filepath.Join("out", "zhiyuan.db"), "SQLite staging 库")
	provSlug := fs.String("prov", "hlj", "省份 slug：hlj / zj / js …")
	_ = fs.Parse(args)
	p := mustProv(*provSlug)

	db, err := store.Open(*dbPath)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	project := projectGroups
	if p.model == "major" { // 专业平行志愿（无院校专业组）：综合(浙江/山东)或双科类(重庆/辽宁)
		project = projectPlanMajors
	}
	b := buildBundle(db, p, project)
	emitSchoolData(p, b, srcDir(*out, p), pubDir(*pub, p))
}
