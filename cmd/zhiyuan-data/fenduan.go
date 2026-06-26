package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

// fenduanCmd 解析一分一段表 xlsx → JSON（按省份分目录），供站点做分数↔位次换算。
func fenduanCmd(args []string) {
	fs := flag.NewFlagSet("fenduan", flag.ExitOnError)
	src := fs.String("src", defaultSrc(), "官方数据根目录")
	out := fs.String("out", filepath.Join("src", "data"), "JSON 输出目录（其下按省份 slug 分目录）")
	provSlug := fs.String("prov", "hlj", "省份 slug：hlj / zj")
	_ = fs.Parse(args)
	p := mustProv(*provSlug)

	type job struct {
		path, track string
		year        int
	}
	var jobs []job
	switch p.slug {
	case "hlj":
		// 目前仅 2026 物理为 .xlsx 可读（历史年份为 .xls，待接入）。
		jobs = append(jobs, job{filepath.Join(*src, "黑龙江2026物理类一分一段表.xlsx"), "物理", 2026})
	case "zj":
		for y := 2022; y <= 2025; y++ {
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
