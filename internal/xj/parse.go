// Package xj 是新疆（老文理 理科/文科）的解析器。新疆仍是老高考（理科/文科、专业平行志愿、无院校
// 专业组），源表与新高考 group 省同形（含最低位次的专业录取分数 + 招生计划 + 一分一段），唯一差别
// 是科类口径——理科/文科。故复用 internal/group3p12 的逐行解析，仅把 keep 科类集合换成 {理科,文科}
// 透传进 *With 变体；不并入 group3p12 的默认 keep，否则会污染重庆/贵州等省的老文理历史行。
// 所属专业组列全空 → 走 major 模型（buildMajorBundle，已 track-aware）。见 issue #27、ADR-0014。
package xj

import (
	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/group3p12"
)

// keep 是新疆（老文理）放行的科类。艺术理/体育理/艺术文 等不在内，会被过滤掉。
var keep = map[string]bool{"理科": true, "文科": true}

// ParseScores 解析新疆「专业录取分数」xlsx（仅理科/文科本科、含最低位次）。
func ParseScores(path string) ([]core.MajorScoreRow, error) {
	return group3p12.ParseScoresWith(path, keep)
}

// ParsePlan 解析新疆「招生计划」xlsx（仅理科/文科本科）。
func ParsePlan(path string) ([]core.PlanRow, error) {
	return group3p12.ParsePlanWith(path, keep)
}

// ParseYiFenYiDuan 解析新疆一分一段 xlsx（单文件含理科/文科）。源表无「批次」列，group3p12 的
// yfd 解析在无批次列时自动跳过批次过滤，故老文理整体分布全留。
func ParseYiFenYiDuan(path, province string, year int) ([]*core.YiFenYiDuan, error) {
	return group3p12.ParseYiFenYiDuanWith(path, province, year, keep)
}
