package hlj

import (
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

// PlanRow / Group2026 / GroupMajor / BuildGroups2026 是 3+1+2 院校专业组模型，已下沉到 core
// （江苏/黑龙江同形，见 ADR-0014）。这里保留别名，黑龙江解析器与既有调用方零改动。
type (
	PlanRow    = core.PlanRow
	Group2026  = core.Group2026
	GroupMajor = core.GroupMajor
)

// BuildGroups2026 见 core.BuildGroups2026。
var BuildGroups2026 = core.BuildGroups2026

// planHeader 判定招生计划表的表头行（含院校代码/专业名称/计划人数）。
func planHeader(r []string) bool {
	return core.HasCell(r, "院校代码") && core.HasCell(r, "专业名称") && core.HasCell(r, "计划人数")
}

// ParsePlanXLSX 解析招生计划 xlsx，只返回新科类（物理/历史）本科批行。表头驱动。
func ParsePlanXLSX(path string) ([]PlanRow, error) {
	s, err := core.OpenSheet(path, planHeader)
	if err != nil {
		return nil, err
	}
	return parsePlanSheet(s), nil
}

func parsePlanSheet(s *core.Sheet) []PlanRow {
	col := s.Col
	cYear, cTrack, cBatch := col("年份"), col("科类"), col("批次")
	cSchoolCode, cSchoolName := col("院校代码"), col("院校名称")
	cGroup, cGroupName := col("专业组代码"), col("专业组名称")
	cMajor, cSelKe := col("专业名称"), col("选科要求")
	cFull, cRemark := col("专业全称"), col("专业备注")
	cPlan, cSchooling, cTuition := col("计划人数"), col("学制"), col("学费")

	var out []PlanRow
	for _, r := range s.Data {
		track := strings.TrimSpace(core.Cell(r, cTrack))
		if !newGaokaoTracks[track] {
			continue
		}
		if !strings.Contains(core.Cell(r, cBatch), "本科") {
			continue
		}
		name := strings.TrimSpace(core.Cell(r, cMajor))
		code := strings.TrimSpace(core.Cell(r, cSchoolCode))
		if name == "" || code == "" {
			continue
		}
		year, _ := core.ParseLeadingInt(core.Cell(r, cYear))
		plan, _ := core.ParseLeadingInt(core.Cell(r, cPlan))
		out = append(out, PlanRow{
			Year:       year,
			Track:      track,
			SchoolCode: code,
			SchoolName: strings.TrimSpace(core.Cell(r, cSchoolName)),
			GroupCode:  strings.TrimSpace(core.Cell(r, cGroup)),
			GroupName:  strings.TrimSpace(core.Cell(r, cGroupName)),
			MajorName:  name,
			FullName:   strings.TrimSpace(core.Cell(r, cFull)),
			Remark:     strings.TrimSpace(core.Cell(r, cRemark)),
			SelKe:      strings.TrimSpace(core.Cell(r, cSelKe)),
			Plan:       plan,
			Schooling:  strings.TrimSpace(core.Cell(r, cSchooling)),
			Tuition:    strings.TrimSpace(core.Cell(r, cTuition)),
		})
	}
	return out
}
