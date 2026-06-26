// Command zhiyuan-data 是高考志愿数据的 build 前预处理工具：
// 把官方 xlsx 解析、清洗、换算成结构化 JSON，供 Astro 在 build 时读取。
// 见 docs/adr/0005。
package main

import (
	"fmt"
	"os"
)

const usage = `zhiyuan-data — 高考志愿数据预处理工具（黑龙江）

用法:
  zhiyuan-data <command> [flags]

命令（随切片逐步实现）:
  fenduan    解析一分一段表 → JSON
  yuanxiao   解析专业录取分数线 → 院校 / 院校×专业 JSON
  dingwei    构建位次定位索引

原始数据默认从 ~/Developments/zhiyuan/官方数据 读取（用 -src 覆盖）。
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "-h", "--help", "help":
		fmt.Print(usage)
	case "fenduan":
		fenduanCmd(os.Args[2:])
	case "yuanxiao":
		yuanxiaoCmd(os.Args[2:])
	case "dingwei":
		dingweiCmd(os.Args[2:])
	case "zhuanye":
		zhuanyeCmd(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "未知命令 %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}
