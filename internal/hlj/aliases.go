// Package hlj 是黑龙江高考数据的省份专属领域逻辑：专业录取分数线解析（物理/历史·本科批）、
// 招生计划→2026 院校专业组视图、万师兄旧格式表的院校属性/门类映射、选科判定。
// 省份无关的原语（专业名键、行表聚合、一分一段换算、等效位次、xlsx 助手、城市层级、
// 中外合作/学费）已抽到 internal/core；本文件按原有 hlj.* 名再导出，保持内部与 cmd 调用不变。
package hlj

import "github.com/sunfmin/zhiyuanwiki/internal/core"

// 类型别名（与 core 同一类型，黑龙江历史调用方与测试沿用短名）。
type (
	School           = core.School
	YearScore        = core.YearScore
	MajorLeaf        = core.MajorLeaf
	MajorScoreRow    = core.MajorScoreRow
	YiFenYiDuan      = core.YiFenYiDuan
	FenduanEntry     = core.FenduanEntry
	YearTrack        = core.YearTrack
	MenleiClassifier = core.MenleiClassifier
)

// 函数/方法再导出。
var (
	NormalizeMajorName   = core.NormalizeMajorName
	MajorKey             = core.MajorKey
	AggregateLeaves      = core.AggregateLeaves
	ParseYiFenYiDuanXLSX = core.ParseYiFenYiDuanXLSX
	EquivRank            = core.EquivRank
	IsCoop               = core.IsCoop
	ParseTuition         = core.ParseTuition
	CityTier             = core.CityTier
	LoadMenlei           = core.LoadMenlei
	NewMenleiClassifier  = core.NewMenleiClassifier

	// 供本包剩余解析器（major_scores/plan/school_meta）继续以小写名调用的 xlsx 与名称助手。
	hasCell         = core.HasCell
	findCol         = core.FindCol
	cell            = core.Cell
	parseLeadingInt = core.ParseLeadingInt
	normName        = core.NormName
	baseName        = core.BaseName
)
