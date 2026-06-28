package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/store"
)

// fenduanCmd 解析一分一段表 → JSON（按省份分目录），供站点做分数↔位次换算。
// DB 投影省份（js）从 SQLite staging 读；其余从官方 xlsx 读。
func fenduanCmd(args []string) {
	fs := flag.NewFlagSet("fenduan", flag.ExitOnError)
	src := fs.String("src", defaultSrc(), "官方数据根目录")
	out := fs.String("out", filepath.Join("src", "data"), "JSON 输出目录（其下按省份 slug 分目录）")
	dbPath := fs.String("db", filepath.Join("out", "zhiyuan.db"), "SQLite staging 库（DB 投影省份用，如 js）")
	provSlug := fs.String("prov", "hlj", "省份 slug：hlj / zj / js")
	_ = fs.Parse(args)
	p := mustProv(*provSlug)

	// 构建期 staging 管线省份（js/hn/cq…）：从 SQLite 投影；其余从官方 xlsx 读。见 ADR-0014。
	if _, ok := provParsers[p.slug]; ok {
		fenduanFromDB(*dbPath, *out, p)
		return
	}

	type job struct {
		path, track string
		year        int
	}
	var jobs []job
	switch p.slug {
	case "zj":
		// 2026 起官方源是省考试院 PDF（《浙江省2026年普通高校招生成绩分数段表(总分)》，
		// zjzs.net art_45_12452），导出为同名 xlsx 放进源目录即可与历年统一回流。
		for y := 2022; y <= 2026; y++ {
			jobs = append(jobs, job{
				zjPath(*src, zjDataDir, "一分一段", fmt.Sprintf("浙江%d年的一分一段表.xlsx", y)),
				"综合", y})
		}
	}

	dir := filepath.Join(srcDir(*out, p), "yifenyiduan")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fatal(err)
	}
	for _, j := range jobs {
		// 源文件缺失则跳过（增量回流：只重建本机有源表的年份），其余错误仍硬失败以暴露问题。
		if _, statErr := os.Stat(j.path); os.IsNotExist(statErr) {
			fmt.Printf("⚠ 跳过 %s · %s %d：源文件不存在 %s\n", p.name, j.track, j.year, j.path)
			continue
		}
		y, err := core.ParseYiFenYiDuanXLSX(j.path, p.name, j.track, j.year)
		if err != nil {
			fatal(err)
		}
		outName := fmt.Sprintf("%s-%d.json", trackSlug[j.track], j.year)
		writeJSON(filepath.Join(dir, outName), y)
		fmt.Printf("✓ %s · %s %d 一分一段：%d 个分数段 → %s\n",
			p.name, j.track, j.year, len(y.Entries), outName)
	}
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
