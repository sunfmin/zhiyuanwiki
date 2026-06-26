package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/zj"
)

// 浙江源数据相对路径（目录名含全角【】（）等，须精确匹配）。
const (
	zjRoot      = "09、浙江-2026高考志愿填报资料"
	zjDataDir   = "3-浙江录取数据2022-2025【持续更新】"
	zjScoreXLSX = "22-25年全国高校在浙江的专业录取分数.xlsx"
	zjLianXLSX  = "22-25年浙江（一表联动）.xlsx"
	zjPlanXLSX  = "2026浙江招生计划.xlsx"
)

// zjRefYear 是浙江等效位次的目标年口径：取最近有一分一段的年份（2026 高考未考、无一分一段）。
const zjRefYear = 2025

func zjPath(src string, parts ...string) string {
	return filepath.Join(append([]string{src, zjRoot}, parts...)...)
}

// buildZJBundle 解析浙江数据：专业录取分数(综合·一段/二段/提前批)、一表联动院校/专业属性、
// 2026 招生计划→院校×专业报考视图（专业平行志愿，无组）。
func buildZJBundle(src string) schoolBundle {
	scorePath := zjPath(src, zjDataDir, zjScoreXLSX)
	rows, err := zj.ParseMajorScoresXLSX(scorePath)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("  专业录取分数：%d 行（综合·一段/二段/提前批·含位次）\n", len(rows))
	schools, leaves := core.AggregateLeaves(rows)

	lianPath := zjPath(src, zjDataDir, zjLianXLSX)
	attrs := zj.LoadAttrs([]string{scorePath}, lianPath)
	menlei := core.LoadMenlei([]string{lianPath})
	fmt.Printf("  院校属性 %d 所 · 专业→门类精确映射 %d 条\n", attrs.Len(), menlei.Len())

	// 各年一分一段总人数 → 等效位次缩放（浙江有 2022-2025 四年）。
	totals := map[core.YearTrack]int{}
	for y := 2022; y <= 2025; y++ {
		path := zjPath(src, zjDataDir, "一分一段", fmt.Sprintf("浙江%d年的一分一段表.xlsx", y))
		if t, err := core.ParseYiFenYiDuanXLSX(path, "浙江", zj.Track, y); err == nil {
			totals[core.YearTrack{Year: y, Track: zj.Track}] = t.Total()
		}
	}

	planPath := zjPath(src, "2-浙江26招生计划+政策汇总【持续更新】", "1-浙江2026招生计划", zjPlanXLSX)
	var planByCode map[string][]zj.PlanMajor
	if planRows, err := zj.ParsePlan2026XLSX(planPath); err != nil {
		fmt.Fprintf(os.Stderr, "警告：未读到 2026 招生计划（%v），跳过报考视图\n", err)
		planByCode = map[string][]zj.PlanMajor{}
	} else {
		fmt.Printf("  2026 招生计划：%d 行（综合）\n", len(planRows))
		planByCode = zj.BuildPlan2026(planRows, leaves, totals, zjRefYear, menlei)
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
	planCount := 0
	for _, s := range schools {
		d := schoolDetail{School: s, Leaves: byCode[s.Code], Plan2026: planByCode[s.Code]}
		planCount += len(d.Plan2026)
		b.details[s.Code] = d
		if a, ok := attrs.Lookup(s.Code); ok {
			b.levels[s.Code] = [3]bool{a.Is985, a.Is211, a.IsShuangYiLiu}
			b.meta[s.Code] = schoolMetaOut{
				Province: a.Province, City: a.City, CityTier: a.CityTier,
				Owner: a.Ownership, Kind: a.Kind, Levels: a.Levels(),
			}
		}
	}
	fmt.Printf("  2026 院校×专业 %d 个\n", planCount)
	return b
}
