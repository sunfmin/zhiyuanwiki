// Package xz 是西藏（老文理 理科/文科）的解析器。西藏与新疆同属老高考（理科/文科、专业平行志愿、
// 无院校专业组），但有一处全站独有的硬约束：**西藏考试院不发布一分一段表，录取数据也无最低位次**
// （22-25 专业录取分数表「最低位次」列全空；学信网全国分数段统计汇总里西藏栏位空白）。
//
// 全站本以位次为准（见 CONTEXT.md / ADR-0014 当初把西藏列入「待接入·缺数据」）。既然位次客观缺失，
// 西藏改走「只有分数」口径：解析复用 group3p12 的逐行循环，但用 ParseScoresScoreOnly（不丢无位次行、
// MinRank 落 0、以最低分数为门槛）；下游 majorx/dingwei 据 PrevRank==0 && PrevScore>0 切到分数域
// 投影/定位，前端 locatorBasis="score"。不提供一分一段解析（源根本没有）。
//
// 另注：西藏录取线官方分 A 类（区内世居少数民族）/ B 类（汉族及区外），但源表未按 A/B 标注，同一
// (年·校·科·批·专业) 常出现 2–4 个不同最低分（汉/少数民族两线混在一起）。聚合时按最低分取代表、
// 同时记分数跨度上界（见 core.AggregateLeaves 的无位次分支），叶子页与定位明示「含两类、未区分」。
package xz

import (
	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/group3p12"
)

// keep 是西藏（老文理）放行的科类。艺术文/体育文/艺术理 等不在内，会被过滤掉。
var keep = map[string]bool{"理科": true, "文科": true}

// ParseScores 解析西藏「专业录取分数」xlsx（仅理科/文科本科）。无最低位次——走 ParseScoresScoreOnly。
func ParseScores(path string) ([]core.MajorScoreRow, error) {
	return group3p12.ParseScoresScoreOnly(path, keep)
}

// ParsePlan 解析西藏「招生计划」xlsx（仅理科/文科本科）。计划侧不含位次，与其他省同形，直接复用。
func ParsePlan(path string) ([]core.PlanRow, error) {
	return group3p12.ParsePlanWith(path, keep)
}
