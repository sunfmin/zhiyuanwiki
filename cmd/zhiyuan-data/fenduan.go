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
