// Command zhiyuan-data 是高考志愿数据的 build 前预处理工具：
// 把官方 xlsx 解析、清洗、换算成结构化 JSON，供 Astro 在 build 时读取。
// 见 docs/adr/0005。
package main

import (
	"fmt"
	"os"
)

const usage = `zhiyuan-data — 高考志愿数据预处理工具（多省份）

用法:
  zhiyuan-data <command> [-prov hlj|zj] [flags]

命令:
  import     解析官方 xlsx + 全国表 → SQLite staging（按省幂等，见 ADR-0014）
  import2026 把抓取的 2026 一分一段 JSON（out/2026yfd/<slug>.json）additive 入库（只动 year=2026）
  fenduan    解析一分一段表 → JSON
  yuanxiao   解析专业录取分数线 → 院校 / 院校×专业 / 2026 报考视图 JSON
  zhuanye    跨校聚合专业 → 专业索引与详情
  dingwei    构建位次定位索引
  landing    产出省份列表落地页全国数据（本省院校数）→ src/data/home-schools.json

-prov 选择省份（默认 hlj 黑龙江；zj 浙江），产物按 src/data/<slug>、public/data/<slug> 分目录。
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
	case "import":
		importCmd(os.Args[2:])
	case "import2026":
		import2026Cmd(os.Args[2:])
	case "fenduan":
		fenduanCmd(os.Args[2:])
	case "yuanxiao":
		yuanxiaoCmd(os.Args[2:])
	case "dingwei":
		dingweiCmd(os.Args[2:])
	case "zhuanye":
		zhuanyeCmd(os.Args[2:])
	case "landing":
		landingCmd(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "未知命令 %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}
