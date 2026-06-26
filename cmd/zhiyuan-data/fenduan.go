package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunfmin/zhiyuanwiki/internal/hlj"
)

// fenduanCmd 解析一分一段表 xlsx → JSON，供站点做分数↔位次换算。
func fenduanCmd(args []string) {
	fs := flag.NewFlagSet("fenduan", flag.ExitOnError)
	src := fs.String("src", defaultSrc(), "官方数据根目录")
	out := fs.String("out", filepath.Join("src", "data", "yifenyiduan"), "JSON 输出目录")
	_ = fs.Parse(args)

	jobs := []struct {
		file, province, track, outName string
		year                           int
	}{
		{"黑龙江2026物理类一分一段表.xlsx", "黑龙江", "物理", "wuli-2026.json", 2026},
	}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fatal(err)
	}
	for _, j := range jobs {
		path := filepath.Join(*src, j.file)
		y, err := hlj.ParseYiFenYiDuanXLSX(path, j.province, j.track, j.year)
		if err != nil {
			fatal(err)
		}
		writeJSON(filepath.Join(*out, j.outName), y)
		fmt.Printf("✓ %s · %s %d 一分一段：%d 个分数段 → %s\n",
			j.province, j.track, j.year, len(y.Entries), j.outName)
	}
}

func defaultSrc() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Developments", "zhiyuan", "官方数据")
}

func writeJSON(path string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "错误:", err)
	os.Exit(1)
}
