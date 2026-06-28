package zj

import "github.com/sunfmin/zhiyuanwiki/internal/core"

// 浙江接入 SQLite staging（见 ADR-0014 / issue #20）的阻抗适配：浙江是 major 模型，计划行
// 用省专属的 PlanRow2026（有「招生类型」、无院校专业组），与统一 staging 的 core.PlanRow 不同形。
// 这里给出两向无损转换，入库走 ToCorePlan、投影走 FromCorePlan，BuildPlan2026 仍吃 PlanRow2026。

// ToCorePlan 把浙江 2026 计划行转成统一 core.PlanRow（入 plan 表）。科类固定「综合」、无组
// （GroupCode 空）；招生类型落 Batch 列——投影时还原它喂 core.IsCoop，否则中外合作判定会回归。
func ToCorePlan(rows []PlanRow2026) []core.PlanRow {
	out := make([]core.PlanRow, len(rows))
	for i, r := range rows {
		out[i] = core.PlanRow{
			Year:       r.Year,
			Track:      Track,
			Batch:      r.AdmitType,
			SchoolCode: r.SchoolCode,
			SchoolName: r.SchoolName,
			MajorName:  r.MajorName,
			Remark:     r.Remark,
			SelKe:      r.SelKe,
			Plan:       r.Plan,
			Schooling:  r.Schooling,
			Tuition:    r.Tuition,
		}
	}
	return out
}

// FromCorePlan 是 ToCorePlan 的逆：从 plan 表投影回浙江 2026 计划行，供 BuildPlan2026。
func FromCorePlan(rows []core.PlanRow) []PlanRow2026 {
	out := make([]PlanRow2026, len(rows))
	for i, r := range rows {
		out[i] = PlanRow2026{
			Year:       r.Year,
			SchoolCode: r.SchoolCode,
			SchoolName: r.SchoolName,
			AdmitType:  r.Batch,
			MajorName:  r.MajorName,
			Remark:     r.Remark,
			SelKe:      r.SelKe,
			Plan:       r.Plan,
			Schooling:  r.Schooling,
			Tuition:    r.Tuition,
		}
	}
	return out
}

// ParseYiFenYiDuan 解析浙江一分一段 xlsx（综合科类）→ 一张表，签名与统一 staging 的 YFD 解析器一致。
func ParseYiFenYiDuan(path, province string, year int) ([]*core.YiFenYiDuan, error) {
	y, err := core.ParseYiFenYiDuanXLSX(path, province, Track, year)
	if err != nil {
		return nil, err
	}
	return []*core.YiFenYiDuan{y}, nil
}
