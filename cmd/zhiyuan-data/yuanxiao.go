package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/hlj"
)

// yuanxiaoCmd 解析专业录取分数线 → 院校 / 院校×专业叶子 / 2026 报考视图 JSON（按省份分目录）。
func yuanxiaoCmd(args []string) {
	fs := flag.NewFlagSet("yuanxiao", flag.ExitOnError)
	src := fs.String("src", defaultSrc(), "官方数据根目录")
	out := fs.String("out", filepath.Join("src", "data"), "JSON 输出目录（其下按省份 slug 分目录）")
	pub := fs.String("pub", filepath.Join("public", "data"), "客户端公开数据目录（其下按省份 slug 分目录）")
	dbPath := fs.String("db", filepath.Join("out", "zhiyuan.db"), "SQLite staging 库（DB 投影省份用，如 js）")
	provSlug := fs.String("prov", "hlj", "省份 slug：hlj / zj / js")
	_ = fs.Parse(args)
	p := mustProv(*provSlug)

	var b schoolBundle
	switch {
	case p.slug == "hlj":
		b = buildHLJBundle(*src)
	case p.slug == "zj":
		b = buildZJBundle(*src)
	default: // 构建期 staging 管线省份（js/hn/cq…）：从 SQLite 投影，见 ADR-0014
		if _, ok := provParsers[p.slug]; !ok {
			fatal(fmt.Errorf("yuanxiao 暂未支持省份 %q", p.slug))
		}
		b = buildDBBundle(*dbPath, p)
	}
	emitSchoolData(p, b, srcDir(*out, p), pubDir(*pub, p))
}

// buildHLJBundle 解析黑龙江万师兄数据：专业分数线(物理/历史·本科批)、招生计划→2026 组视图、
// 旧格式表的院校属性/门类映射（按校名挂接）。
func buildHLJBundle(src string) schoolBundle {
	scoreDir := filepath.Join(src,
		"24-万师兄-黑龙江2026年高考志愿填报大数据",
		"04-万师兄-黑龙江高考-专业录取分数线-2020-2025")

	years := []int{2023, 2024, 2025} // 新科类（物理/历史 + 位次），见 ADR-0007
	var all []core.MajorScoreRow
	for _, y := range years {
		path := findScoreFile(scoreDir, y)
		if path == "" {
			fmt.Fprintf(os.Stderr, "跳过 %d：未找到文件\n", y)
			continue
		}
		rows, err := hlj.ParseMajorScoresXLSX(path)
		if err != nil {
			fatal(err)
		}
		fmt.Printf("  %d：%d 行（物理/历史·本科批·含位次）\n", y, len(rows))
		all = append(all, rows...)
	}
	if len(all) == 0 {
		fatal(fmt.Errorf("未解析到任何专业分数线行"))
	}

	schools, leaves := core.AggregateLeaves(all)

	tagFiles := tagSourceFiles(src)
	meta := hlj.LoadSchoolMeta(tagFiles)
	menlei := core.LoadMenlei(tagFiles)
	fmt.Printf("  院校属性库 %d 所 · 专业→门类精确映射 %d 条\n", meta.Len(), menlei.Len())

	// 一分一段总人数（用于等效位次缩放）。目前仅 2026 物理为 .xlsx 可读。
	totals := map[core.YearTrack]int{}
	if y, err := core.ParseYiFenYiDuanXLSX(
		filepath.Join(src, "黑龙江2026物理类一分一段表.xlsx"), "黑龙江", "物理", 2026); err == nil {
		totals[core.YearTrack{Year: 2026, Track: "物理"}] = y.Total()
	}

	planPath := filepath.Join(src,
		"24-万师兄-黑龙江2026年高考志愿填报大数据",
		"01-万师兄-黑龙江高考-招生计划-2020-2026", "黑龙江_招生计划_2026.xlsx")
	var groupsByCode map[string][]hlj.Group2026
	if planRows, err := hlj.ParsePlanXLSX(planPath); err != nil {
		fmt.Fprintf(os.Stderr, "警告：未读到 2026 招生计划（%v），跳过组视图\n", err)
		groupsByCode = map[string][]hlj.Group2026{}
	} else {
		fmt.Printf("  2026 招生计划：%d 行（物理/历史·本科批）\n", len(planRows))
		groupsByCode = hlj.BuildGroups2026(planRows, leaves, totals, menlei.Code)
	}

	byCode := map[string][]core.MajorLeaf{}
	for _, lf := range leaves {
		byCode[lf.SchoolCode] = append(byCode[lf.SchoolCode], lf)
	}

	b := schoolBundle{
		schools: schools, leaves: leaves,
		details: map[string]schoolDetail{},
		meta:    map[string]schoolMetaOut{},
		levels:  map[string][3]bool{},
	}
	groupCount := 0
	for _, s := range schools {
		d := schoolDetail{School: s, Leaves: byCode[s.Code], Groups2026: groupsByCode[s.Code]}
		groupCount += len(d.Groups2026)
		b.details[s.Code] = d
		if m, ok := meta.Lookup(s.Name); ok {
			b.levels[s.Code] = [3]bool{m.Is985, m.Is211, m.IsShuangYiLiu}
			b.meta[s.Code] = schoolMetaOut{
				Province: m.Province, City: m.City, CityTier: core.CityTier(m.City),
				Owner: m.Ownership, Kind: m.Kind, Levels: m.Levels(),
			}
		}
	}
	fmt.Printf("  2026 院校专业组 %d 个\n", groupCount)
	return b
}

// findScoreFile 在目录里找某年的专业分数线文件（连字符或下划线命名）。
func findScoreFile(dir string, year int) string {
	for _, name := range []string{
		fmt.Sprintf("黑龙江-专业分数线-%d.xlsx", year),
		fmt.Sprintf("黑龙江_专业分数线_%d.xlsx", year),
	} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// tagSourceFiles 列出含「_985/_211/双一流」列的旧格式文件（相对数据根目录）。
// 跨年合并以最大化覆盖（标签是院校级属性，与年份无关；缺失项静默跳过）。
func tagSourceFiles(src string) []string {
	rel := []string{
		"24-万师兄-黑龙江2026年高考志愿填报大数据/01-万师兄-黑龙江高考-招生计划-2020-2026/黑龙江_招生计划_2020.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/01-万师兄-黑龙江高考-招生计划-2020-2026/黑龙江_招生计划_2021.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/01-万师兄-黑龙江高考-招生计划-2020-2026/黑龙江_招生计划_2022.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/01-万师兄-黑龙江高考-招生计划-2020-2026/黑龙江_招生计划_2023.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/04-万师兄-黑龙江高考-专业录取分数线-2020-2025/黑龙江_专业分数线_2020.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/04-万师兄-黑龙江高考-专业录取分数线-2020-2025/黑龙江_专业分数线_2021.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/03-万师兄-黑龙江高考-投档线-2020-2025/黑龙江_投档线_2020.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/03-万师兄-黑龙江高考-投档线-2020-2025/黑龙江_投档线_2021.xlsx",
		"24-万师兄-黑龙江2026年高考志愿填报大数据/03-万师兄-黑龙江高考-投档线-2020-2025/黑龙江_投档线_2022.xlsx",
	}
	out := make([]string, len(rel))
	for i, r := range rel {
		out[i] = filepath.Join(src, filepath.FromSlash(r))
	}
	return out
}
