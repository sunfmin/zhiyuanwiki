// Package shanxi 是山西（2025 首届 3+1+2 新高考·院校专业组）的专业录取分数解析器。
//
// 山西的源表与统一格式 group 省不同形，故不能套 internal/group3p12：
//   - 录取分数表**没有院校代码列**（列为 生源地/科类/批次/院校名称/专业组名称/专业全称/
//     专业层次/选科要求/录取最低分/录取最低位次），且首行是标题行、列名也异形（「录取最低分/
//     录取最低位次」而非「最低分数/最低位次」）。院校代码由 cmd 侧 importShanxi 按校名从招生
//     计划（有规范代码）回填，故本解析器产出的行 SchoolCode 留空。
//   - 专业名取「专业全称」并截去括号尾注（五年/八年/办学地点等），以便与招生计划的裸专业名
//     按 (院校,专业) 挂接——统一格式 group 省的录取分数本就是裸名，山西特有这层尾注。
//
// 招生计划列名规范（院校代码/专业名称/计划人数），直接复用 group3p12.ParsePlan；一分一段科类
// 编码在文件名（无科类列），由 core.ParseYiFenYiDuanXLSX 逐文件解析。见 ADR-0013/0014。
package shanxi

import (
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

// keep 是山西（2025 新高考）放行的科类。理科/文科（改革前老文理年份）、艺术/体育不在内。
var keep = map[string]bool{"物理": true, "历史": true}

// canonTrack 把源表科类归一为站点口径（物理类→物理）；山西 2025 本就是裸「物理/历史」，是恒等。
func canonTrack(s string) string {
	switch strings.TrimSpace(s) {
	case "物理类", "物理科目组合":
		return "物理"
	case "历史类", "历史科目组合":
		return "历史"
	}
	return strings.TrimSpace(s)
}

// scoreHeader 定位录取分数表头行（首行为标题，故按谓词找真表头）：有「院校名称」「科类」且有含
// 「最低位次」的列。
func scoreHeader(r []string) bool {
	return core.HasCell(r, "科类") && core.HasCell(r, "院校名称") && core.HasCellContains(r, "最低位次")
}

// ParseScores 解析山西「专业录取分数线」xlsx（仅物理/历史本科、含最低位次）。year 由调用方给定
// （源表无年份列；山西首届新高考为 2025，文件名标 2024 实为 2025）。SchoolCode 留空，由 importShanxi
// 按校名从招生计划回填。
func ParseScores(path string, year int) ([]core.MajorScoreRow, error) {
	s, err := core.OpenSheet(path, scoreHeader)
	if err != nil {
		return nil, err
	}
	return parseScores(s, year), nil
}

func parseScores(s *core.Sheet, year int) []core.MajorScoreRow {
	cTrack, cBatch := s.Col("科类"), s.Col("批次")
	cName, cGroup := s.Col("院校名称"), s.Col("专业组名称")
	cMajor, cSelKe := s.Col("专业全称", "专业名称", "专业"), s.Col("选科要求")
	cMin, cRank := s.ColContains("最低分"), s.ColContains("最低位次")

	var out []core.MajorScoreRow
	for _, r := range s.Data {
		track := canonTrack(core.Cell(r, cTrack))
		if !keep[track] || !strings.Contains(core.Cell(r, cBatch), "本科") {
			continue
		}
		minRank, hasRank := core.ParseLeadingInt(core.Cell(r, cRank))
		if !hasRank {
			continue
		}
		// 专业全称带「(五年)/(八年)/(办学地点…)」尾注；截断按裸名挂接招生计划（计划侧也是裸名）。
		name := core.StripParenTail(core.Cell(r, cMajor))
		if name == "" {
			continue
		}
		minScore, _ := core.ParseLeadingInt(core.Cell(r, cMin))
		out = append(out, core.MajorScoreRow{
			Year:       year,
			Track:      track,
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
