package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/sunfmin/zhiyuanwiki/internal/store"
)

// landingCmd 产出省份列表落地页的全国级数据（见 ADR-0016）：
//   - 本省院校数：校址在该省的高校数（全国 school 表，全 31 省皆有，含未上线省）→ home-schools.json（中文省名→数）。
//   - 本科招生计划：各省最新年本科批招生计划总数（plan 表，仅已入库省）→ benke-plan.json（slug→{plan,year}）。
//
// 高考人数/本科线从前端按 committed 一分一段派生（6 省统一源），不在此 emit。
func landingCmd(args []string) {
	fs := flag.NewFlagSet("landing", flag.ExitOnError)
	dbPath := fs.String("db", filepath.Join("out", "zhiyuan.db"), "SQLite staging 库（全国 school 表 / plan 表来源）")
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
	homePath := filepath.Join(*out, "home-schools.json")
	writeJSON(homePath, counts)
	fmt.Printf("本省院校数：%d 省（校址在本省的高校数，全国 school 表）→ %s\n", len(counts), homePath)

	plans, err := db.PlanTotalsLatest()
	if err != nil {
		fatal(err)
	}
	planPath := filepath.Join(*out, "benke-plan.json")
	writeJSON(planPath, plans)
	fmt.Printf("本科招生计划：%d 省（最新年本科批计划总数，plan 表）→ %s\n", len(plans), planPath)
}
