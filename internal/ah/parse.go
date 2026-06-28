// Package ah 是安徽高考数据的省份专属解析（3+1+2：物理/历史 + 院校专业组）。
// 数据源是 各省份/ 树的统一格式表（与江苏同形）；省份无关的聚合/组装/门类用 internal/core，
// 入库/投影见 ADR-0014。逐行解析留在本包是 ADR-0013 有意为之的省份缝。
package ah

import (
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

// Tracks 是安徽收录的两个科类（已归一，去「类」后缀）。
var Tracks = []string{"物理", "历史"}

var keep = map[string]bool{"物理": true, "历史": true}

// canonTrack 把源表科类归一为站点口径：物理类→物理、历史类→历史；其余（艺术类（物理）等）原样
// 返回后被 keep 过滤掉。
func canonTrack(s string) string {
	s = strings.TrimSpace(s)
	switch s {
	case "物理类":
		return "物理"
	case "历史类":
		return "历史"
	}
	return s
}

// batchKeep：留本科（含本科批/本科提前批），丢专科/艺术/体育批。艺术/体育已由科类过滤掉，
// 这里再挡掉专科。
func batchKeep(batch string) bool { return strings.Contains(batch, "本科") }

func scoreHeader(r []string) bool {
	return core.HasCell(r, "院校代码") && core.HasCell(r, "最低位次")
}

// ParseScores 解析安徽「专业录取分数」xlsx → 行表（仅物理/历史本科、含最低位次）。表头驱动。
func ParseScores(path string) ([]core.MajorScoreRow, error) {
	s, err := core.OpenSheet(path, scoreHeader)
	if err != nil {
		return nil, err
	}
	return parseScores(s), nil
}

func parseScores(s *core.Sheet) []core.MajorScoreRow {
	col := s.Col
	cYear, cTrack, cBatch := col("年份"), col("科类"), col("批次")
	cCode, cName := col("院校代码"), col("院校名称")
	cGroup := col("所属专业组")
	cMajor, cSelKe := col("专业", "专业名称"), col("选科要求")
	cMin, cRank := col("最低分数", "最低分"), col("最低位次")

	var out []core.MajorScoreRow
	for _, r := range s.Data {
		track := canonTrack(core.Cell(r, cTrack))
		if !keep[track] || !batchKeep(core.Cell(r, cBatch)) {
			continue
		}
		minRank, hasRank := core.ParseLeadingInt(core.Cell(r, cRank))
		if !hasRank {
			continue
		}
		name := strings.TrimSpace(core.Cell(r, cMajor))
		code := core.NormSchoolCode(core.Cell(r, cCode))
		if name == "" || code == "" {
			continue
		}
		year, _ := core.ParseLeadingInt(core.Cell(r, cYear))
		minScore, _ := core.ParseLeadingInt(core.Cell(r, cMin))
		out = append(out, core.MajorScoreRow{
			Year:       year,
			Track:      track,
			SchoolCode: code,
			SchoolName: strings.TrimSpace(core.Cell(r, cName)),
			GroupCode:  strings.TrimSpace(core.Cell(r, cGroup)),
			MajorName:  name,
			SelKe:      strings.TrimSpace(core.Cell(r, cSelKe)),
			MinScore:   minScore,
			MinRank:    minRank,
		})
	}
	return out
}

func planHeader(r []string) bool {
	return core.HasCell(r, "院校代码") && core.HasCell(r, "专业名称") &&
		(core.HasCell(r, "招生人数") || core.HasCell(r, "计划人数"))
}

// ParsePlan 解析安徽「招生计划」xlsx → 计划行（仅物理/历史本科）。表头驱动。GroupCode 取所属专业组。
func ParsePlan(path string) ([]core.PlanRow, error) {
	s, err := core.OpenSheet(path, planHeader)
	if err != nil {
		return nil, err
	}
	return parsePlan(s), nil
}

func parsePlan(s *core.Sheet) []core.PlanRow {
	col := s.Col
	cYear, cTrack, cBatch := col("年份"), col("科类"), col("批次")
	cCode, cName := col("院校代码"), col("院校名称")
	cGroupCode, cGroupName := col("专业组代码", "所属专业组"), col("专业组名称")
	cMajor, cSelKe := col("专业名称", "专业"), col("选科要求")
	cRemark := col("专业备注")
	cPlan := col("计划人数", "招生人数")
	cSchooling, cTuition := col("学制(年)", "学制"), col("学费(元)", "学费")

	var out []core.PlanRow
	for _, r := range s.Data {
		track := canonTrack(core.Cell(r, cTrack))
		if !keep[track] || !batchKeep(core.Cell(r, cBatch)) {
			continue
		}
		// 安徽招生计划专业名带括号尾注（语种/办学地点等），录取分数表用裸名——截断以挂接。
		name := core.StripParenTail(core.Cell(r, cMajor))
		code := core.NormSchoolCode(core.Cell(r, cCode))
		if name == "" || code == "" {
			continue
		}
		year, _ := core.ParseLeadingInt(core.Cell(r, cYear))
		plan, _ := core.ParseLeadingInt(core.Cell(r, cPlan))
		gcode := strings.TrimSpace(core.Cell(r, cGroupCode))
		gname := strings.TrimSpace(core.Cell(r, cGroupName))
		if gname == "" {
			gname = gcode // 无独立组名时用组代码兜底（安徽源表有「NNN专业组」）
		}
		out = append(out, core.PlanRow{
			Year:       year,
			Track:      track,
			SchoolCode: code,
			SchoolName: strings.TrimSpace(core.Cell(r, cName)),
			GroupCode:  gcode,
			GroupName:  gname,
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

func yfdHeader(r []string) bool {
	return core.HasCellContains(r, "累计") && core.HasCell(r, "科类")
}

// ParseYiFenYiDuan 解析安徽一分一段 xlsx（单文件含物理/历史，本科批），按 年×科类 分组。
// 表头带单位后缀（分数(分)/本段人数(人)/累计人数(人)），列定位走 ColContains。
func ParseYiFenYiDuan(path, province string, year int) ([]*core.YiFenYiDuan, error) {
	s, err := core.OpenSheet(path, yfdHeader)
	if err != nil {
		return nil, err
	}
	return parseYiFenYiDuan(s, province, year), nil
}

func parseYiFenYiDuan(s *core.Sheet, province string, year int) []*core.YiFenYiDuan {
	cTrack, cBatch := s.Col("科类"), s.Col("批次")
	cScore := s.ColContains("分数", "分段")
	cCount, cCum := s.ColContains("本段人数"), s.ColContains("累计")

	byTrack := map[string]*core.YiFenYiDuan{}
	var order []string
	for _, r := range s.Data {
		track := canonTrack(core.Cell(r, cTrack))
		if !keep[track] || !batchKeep(core.Cell(r, cBatch)) {
			continue
		}
		score, ok := core.ParseLeadingInt(core.Cell(r, cScore))
		if !ok {
			continue
		}
		cum, ok := core.ParseLeadingInt(core.Cell(r, cCum))
		if !ok {
			continue
		}
		count, _ := core.ParseLeadingInt(core.Cell(r, cCount))
		y := byTrack[track]
		if y == nil {
			y = &core.YiFenYiDuan{Province: province, Track: track, Year: year}
			byTrack[track] = y
			order = append(order, track)
		}
		y.Entries = append(y.Entries, core.FenduanEntry{Score: score, Count: count, Cumulative: cum})
	}
	out := make([]*core.YiFenYiDuan, 0, len(order))
	for _, t := range order {
		core.SortFenduanAscending(byTrack[t])
		out = append(out, byTrack[t])
	}
	return out
}
