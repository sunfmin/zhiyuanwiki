package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunfmin/zhiyuanwiki/internal/store"
)

// fenduanCmd 从 SQLite staging 投影某省一分一段 JSON（按省份分目录），供站点做分数↔位次换算。
// 全部省份都已入库（ADR-0014），统一从 DB 投影。
func fenduanCmd(args []string) {
	fs := flag.NewFlagSet("fenduan", flag.ExitOnError)
	out := fs.String("out", filepath.Join("src", "data"), "JSON 输出目录（其下按省份 slug 分目录）")
	dbPath := fs.String("db", filepath.Join("out", "zhiyuan.db"), "SQLite staging 库")
	provSlug := fs.String("prov", "hlj", "省份 slug：hlj / zj / js …")
	_ = fs.Parse(args)
	p := mustProv(*provSlug)
	fenduanFromDB(*dbPath, *out, p)
}

// fenduanFromDB 从 SQLite staging 投影某省一分一段 JSON（每 年×科类 一个文件）。见 ADR-0014。
func fenduanFromDB(dbPath, out string, p province) {
	db, err := store.Open(dbPath)
	if err != nil {
		fatal(err)
	}
	defer db.Close()
	yds, err := db.LoadYiFenYiDuan(p.slug, p.name)
	if err != nil {
		fatal(err)
	}
	if len(yds) == 0 {
		fatal(fmt.Errorf("DB 无 %s 一分一段——先跑 `zhiyuan-data import -prov %s`", p.name, p.slug))
	}
	dir := filepath.Join(srcDir(out, p), "yifenyiduan")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fatal(err)
	}
	for _, y := range yds {
		slug, ok := trackSlug[y.Track]
		if !ok {
			continue
		}
		outName := fmt.Sprintf("%s-%d.json", slug, y.Year)
		writeJSON(filepath.Join(dir, outName), y)
		fmt.Printf("✓ %s · %s %d 一分一段：%d 个分数段 → %s\n",
			p.name, y.Track, y.Year, len(y.Entries), outName)
	}
}
