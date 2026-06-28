package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/sunfmin/zhiyuanwiki/internal/store"
)

// landingCmd 产出省份列表落地页的全国级数据：本省院校数（校址在该省的高校数，来自全国
// school 表，全 31 省皆有，含未上线省）。写 src/data/home-schools.json（中文省名→数）。
// 见 ADR-0016。高考人数从前端按 committed 一分一段派生（hlj/zj 不在 staging DB），不在此 emit。
func landingCmd(args []string) {
	fs := flag.NewFlagSet("landing", flag.ExitOnError)
	dbPath := fs.String("db", filepath.Join("out", "zhiyuan.db"), "SQLite staging 库（全国 school 表来源）")
	out := fs.String("out", filepath.Join("src", "data"), "JSON 输出目录")
	_ = fs.Parse(args)

	db, err := store.Open(*dbPath)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	counts, err := db.HomeSchoolCounts()
	if err != nil {
		fatal(err)
	}
	path := filepath.Join(*out, "home-schools.json")
	writeJSON(path, counts)
	fmt.Printf("本省院校数：%d 省（校址在本省的高校数，全国 school 表）→ %s\n", len(counts), path)
}
