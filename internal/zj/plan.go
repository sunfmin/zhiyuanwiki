package zj

// PlanMajor 是某院校的一个可填报专业，挂接了往年最低位次与等效位次。浙江为单科类「综合」（Track 空、
// 省略）；通用 major 管线（双科类省如重庆/辽宁）逐行带 Track，定位索引据此分科类分片。
//
// 类型定名于浙江接入之初、现由通用 major 投影（buildMajorBundle / buildPlanMajorsTracked）共用——
// 浙江自 ADR-0022 起也走这条通用路径，不再有专属计划解析（原 PlanRow2026 / BuildPlan2026 已退役）。
type PlanMajor struct {
	MajorName string `json:"majorName"`
	MajorKey  string `json:"majorKey"`
	Track     string `json:"track,omitempty"` // 科类（综合省留空；双科类省=物理/历史）
	SelKe     string `json:"selKe"`
	Plan      int    `json:"plan"`
	Tuition   string `json:"tuition,omitempty"`
	Schooling string `json:"schooling,omitempty"`
	Menlei    string `json:"menlei,omitempty"` // 学科门类 1 字码
	Coop      bool   `json:"coop,omitempty"`   // 中外合作办学
	PrevYear  int    `json:"prevYear,omitempty"`
	PrevRank  int    `json:"prevRank,omitempty"`  // 最近年份最低位次
	EquivRank int    `json:"equivRank,omitempty"` // 等效到 refYear
	PrevScore int    `json:"prevScore,omitempty"` // 最近年份最低分（只有分数省=西藏的定位/排序基准，位次缺失时用它）
}
