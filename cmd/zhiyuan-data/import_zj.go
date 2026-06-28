package main

import (
	"fmt"
	"os"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/store"
	"github.com/sunfmin/zhiyuanwiki/internal/zj"
)

// importZJ 把浙江入库 staging。浙江源在万师兄树（与旧 buildZJBundle 同源，保证投影产物 diff 一致），
// 目录名含全角标点、布局与 各省份/<省> 不同，故走专用 importZJ 而非通用 importProvince（见 issue #20）。
//   - 专业录取分数（综合·一段/二段/提前批·含位次）：22-25 合表。
//   - 招生计划（2026·综合）：经 zj.ToCorePlan 落统一 plan 表（招生类型存 Batch 列）。
//   - 一分一段（综合，2022-2025）：四年总人数供 BuildPlan2026 等效位次缩放。
//
// 全国院校属性表已由 importCmd 的 importNational 先行刷新；浙江特有 city_tier/按码属性的保全见 #21。
func importZJ(db *store.DB, p province) {
	src := defaultSrc() // 浙江源在万师兄树

	scorePath := zjPath(src, zjDataDir, zjScoreXLSX)
	scores, err := zj.ParseMajorScoresXLSX(scorePath)
	if err != nil {
		fatal(err)
	}
	if err := db.ReplaceScores(p.slug, scores); err != nil {
		fatal(err)
	}
	fmt.Printf("  专业录取分数：%d 行（综合·一段/二段/提前批·含位次）→ major_score\n", len(scores))

	planPath := zjPath(src, "2-浙江26招生计划+政策汇总【持续更新】", "1-浙江2026招生计划", zjPlanXLSX)
	if planRows, err := zj.ParsePlan2026XLSX(planPath); err != nil {
		fmt.Fprintf(os.Stderr, "⚠ 浙江 2026 招生计划解析失败（%v），跳过（报考视图将为空）\n", err)
	} else {
		if err := db.ReplacePlan(p.slug, zj.ToCorePlan(planRows)); err != nil {
			fatal(err)
		}
		fmt.Printf("  招生计划：%d 行（2026·综合）→ plan\n", len(planRows))
	}

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
