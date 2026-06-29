// Package tj 是天津招生计划的省份专属解析。天津是 3+3「综合 + 院校专业组」（group 模型），录取分数表
// 与 group3p12 同形（科类=综合，复用 group3p12.ParseScores/ParseYiFenYiDuan），唯独招生计划表异形：
//   - 无「院校代码」列——但「专业组代码」= 院校代码 + 专业组（如 005603 = 0056 + 03），剥去专业组后缀即得；
//   - 无「科类」列——天津全省综合，注入「综合」；
//   - 计划列名「计划数」、选科列名「选科」、艺体与普通类混表（按批次只留 普通类...本科批）。
// 组码与录取分数表不同形（计划 01 vs 分数 （01））无害：本站组视图按 (院校,专业名) 挂往年位次、
// 组码仅取自计划侧建组（见 core.BuildGroups2026）。
package tj

import (
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

func planHeader(r []string) bool {
	return core.HasCell(r, "专业组代码") && core.HasCell(r, "专业名称") && core.HasCell(r, "计划数")
}

// ParsePlan 解析天津招生计划 xlsx → 计划行（仅普通类本科批，注入科类=综合）。表头驱动。
func ParsePlan(path string) ([]core.PlanRow, error) {
	s, err := core.OpenSheet(path, planHeader)
	if err != nil {
		return nil, err
	}
	return parsePlan(s), nil
}

func parsePlan(s *core.Sheet) []core.PlanRow {
	col := s.Col
	cBatch := col("批次")
	cGroupRaw, cGroup := col("专业组代码"), col("专业组")
	cName, cMajor := col("院校名称"), col("专业名称")
	cRemark, cSelKe := col("备注", "专业备注"), col("选科", "选科要求")
	cPlan := col("计划数", "计划人数", "招生人数")
	cTuition, cSchooling := s.ColContains("学费"), s.ColContains("学制")

	var out []core.PlanRow
	for _, r := range s.Data {
		batch := core.Cell(r, cBatch)
		// 只留「普通类...本科批」（A/B 阶段）；艺考/体育/高职高专/提前/特殊类型批一律丢。
		if !strings.Contains(batch, "普通类") || !strings.Contains(batch, "本科批") || strings.Contains(batch, "提前") {
			continue
		}
		group := strings.TrimSpace(core.Cell(r, cGroup))
		// 院校代码 = 专业组代码 剥去专业组后缀（005603 - 03 = 0056）。
		code := core.NormSchoolCode(strings.TrimSuffix(strings.TrimSpace(core.Cell(r, cGroupRaw)), group))
		name := core.StripParenTail(core.Cell(r, cMajor))
		if code == "" || name == "" {
			continue
		}
		plan, _ := core.ParseLeadingInt(core.Cell(r, cPlan))
		out = append(out, core.PlanRow{
			Year:       2025, // 天津计划表无年份列；本文件即 2025 计划
			Track:      "综合",
			SchoolCode: code,
			SchoolName: strings.TrimSpace(core.Cell(r, cName)),
			GroupCode:  group,
			GroupName:  group,
			MajorName:  name,
			Remark:     strings.TrimSpace(core.Cell(r, cRemark)),
			SelKe:      strings.TrimSpace(core.Cell(r, cSelKe)),
			Plan:       plan,
			Schooling:  strings.TrimSpace(core.Cell(r, cSchooling)),
			Tuition:    strings.TrimSpace(core.Cell(r, cTuition)),
		})
	}
	return out
}
