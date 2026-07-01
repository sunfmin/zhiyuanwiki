package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/store"
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

// importZJ 把浙江入库 staging。浙江源在万师兄树（与旧 buildZJBundle 同源，保证投影产物 diff 一致），
// 目录名含全角标点、布局与 各省份/<省> 不同，故走专用 importZJ 而非通用 importProvince（见 issue #20/#21）。
//   - 专业录取分数（综合·一段/二段/提前批·含位次）：22-25 合表。
//   - 招生计划（2026·综合）：经 zj.ToCorePlan 落统一 plan 表（招生类型存 Batch 列）。
//   - 一分一段（综合，2022-2025）：四年总人数供 BuildPlan2026 等效位次缩放。
//   - 院校属性（按院校代码）：「一表联动」+ 分数表内联列 → school_attr 表。浙江 city_tier 是源表显式
//     标签、且按码挂接，全国 school 表（按校名、无 city_tier）无法表达，故另立按码投影（见 #21）。
func importZJ(db *store.DB, p province) {
	src := defaultSrc() // 浙江源根（09、浙江…，原万师兄树，已归整到 高考志愿/）

	scorePath := zjPath(src, zjDataDir, zjScoreXLSX)
	logSrc("录取分数", scorePath)
	scores, err := zj.ParseMajorScoresXLSX(scorePath)
	if err != nil {
		fatal(err)
	}
	if err := db.ReplaceScores(p.slug, scores); err != nil {
		fatal(err)
	}
	fmt.Printf("  专业录取分数：%d 行（综合·一段/二段/提前批·含位次）→ major_score\n", len(scores))

	planPath := zjPath(src, "2-浙江26招生计划+政策汇总【持续更新】", "1-浙江2026招生计划", zjPlanXLSX)
	logSrc("招生计划", planPath)
	if planRows, err := zj.ParsePlan2026XLSX(planPath); err != nil {
		fmt.Fprintf(os.Stderr, "⚠ 浙江 2026 招生计划解析失败（%v），跳过（报考视图将为空）\n", err)
	} else {
		if err := db.ReplacePlan(p.slug, zj.ToCorePlan(planRows)); err != nil {
			fatal(err)
		}
		fmt.Printf("  招生计划：%d 行（2026·综合）→ plan\n", len(planRows))
	}

	// 院校属性（按代码）：一表联动（城市/层级/类型/层次）+ 分数表内联列（省/性质/985/211 兜底）。
	lianPath := zjPath(src, zjDataDir, zjLianXLSX)
	logSrc("院校属性(一表联动)", lianPath)
	attrs := zj.LoadAttrs([]string{scorePath}, lianPath)
	var attrRows []store.SchoolAttr
	for code, a := range attrs.All() {
		attrRows = append(attrRows, store.SchoolAttr{
			Code: code, Province: a.Province, City: a.City, CityTier: a.CityTier,
			Ownership: a.Ownership, Kind: a.Kind, Is985: a.Is985, Is211: a.Is211, Syl: a.IsShuangYiLiu,
		})
	}
	if err := db.ReplaceSchoolAttrs(p.slug, attrRows); err != nil {
		fatal(err)
	}
	fmt.Printf("  院校属性（按代码）：%d 所 → school_attr\n", len(attrRows))

	var allYfd []*core.YiFenYiDuan
	for y := 2022; y <= 2025; y++ {
		path := zjPath(src, zjDataDir, "一分一段", fmt.Sprintf("浙江%d年的一分一段表.xlsx", y))
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			continue // 增量回流：只入本机有源表的年份
		}
		yds, err := zj.ParseYiFenYiDuan(path, p.name, y)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ 一分一段 浙江%d 解析失败：%v\n", y, err)
			continue
		}
		allYfd = append(allYfd, yds...)
	}
	if err := db.ReplaceYiFenYiDuan(p.slug, allYfd); err != nil {
		fatal(err)
	}
	fmt.Printf("  一分一段：%d 个(年×科类) → yifenyiduan\n", len(allYfd))
	fmt.Printf("✓ %s入库完成 → %s\n", p.name, "out/zhiyuan.db")
}
