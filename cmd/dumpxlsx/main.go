// dumpxlsx — 临时 xlsx 探查工具（不提交；构建到 out/ 供 recon 用）。
// 用法: dumpxlsx <file>...            打印 sheet 名 + 表头附近若干行
//   GREP=物理 dumpxlsx <file>         额外打印至多 6 行含「物理」的行（采样普通类行）
//   DISTINCT=科类 dumpxlsx <file>     打印「科类」列的去重取值及计数（看科类口径）
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

func main() {
	grep := os.Getenv("GREP")
	distinct := os.Getenv("DISTINCT")
	for _, path := range os.Args[1:] {
		f, err := excelize.OpenFile(path)
		if err != nil {
			fmt.Printf("ERR open %s: %v\n", path, err)
			continue
		}
		sheets := f.GetSheetList()
		fmt.Printf("\n=== %s\n  sheets=%v\n", path, sheets)
		rows, _ := f.GetRows(sheets[0])
		// 表头探测：前 15 行里首个非空单元 >=4 的行当表头
		hdr := 0
		for i := 0; i < len(rows) && i < 15; i++ {
			ne := 0
			for _, c := range rows[i] {
				if strings.TrimSpace(c) != "" {
					ne++
				}
			}
			if ne >= 4 {
				hdr = i
				break
			}
		}
		for i := 0; i <= hdr+2 && i < len(rows); i++ {
			fmt.Printf("  [%d] %s\n", i, strings.Join(rows[i], " | "))
		}
		if distinct != "" && hdr < len(rows) {
			col := -1
			for j, c := range rows[hdr] {
				if strings.TrimSpace(c) == distinct {
					col = j
					break
				}
			}
			if col >= 0 {
				seen := map[string]int{}
				for _, r := range rows[hdr+1:] {
					if col < len(r) {
						seen[strings.TrimSpace(r[col])]++
					}
				}
				fmt.Printf("  DISTINCT %q:", distinct)
				n := 0
				for k, c := range seen {
					fmt.Printf(" %q(%d)", k, c)
					if n++; n >= 25 {
						fmt.Print(" …")
						break
					}
				}
				fmt.Println()
			} else {
				fmt.Printf("  DISTINCT %q: 列未找到\n", distinct)
			}
		}
		if grep != "" {
			n := 0
			for i, r := range rows {
				line := strings.Join(r, " | ")
				if strings.Contains(line, grep) {
					fmt.Printf("  ~[%d] %s\n", i, line)
					if n++; n >= 6 {
						break
					}
				}
			}
		}
		f.Close()
	}
}
