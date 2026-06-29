// Package group3p12 是「统一格式 3+1+2 院校专业组省份」的共用解析器（各省份/ 干净树）。
//
// 这些省的源表同形（物理类/历史类 · 含最低位次 · 招生计划带院校专业组代码），逐行解析逻辑
// 逐字节相同——四川/安徽曾各自照抄一份（见 ADR-0014 旧配方「照抄 internal/hn」）。本包把这份
// 解析收成一处：凡格式与四川/安徽一致的 group 省（广西/江西/湖北/云南/河南…）都指向它，
// 不再每省一份拷贝。
//
// 与 ADR-0013「省份缝在 internal/<省>」不冲突：那条缝的意义是让**异构**省份能各自发散；
// 这里是**同构**省份共享一份，正是 single-source-of-truth 取向。真正因省而异的省份（老文理、
// 无组代码、特殊布局）仍各写各包。省份无关的聚合/组装/门类在 internal/core；入库/投影见 ADR-0014。
package group3p12

import (
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

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

// ParseScores 解析「专业录取分数」xlsx → 行表（仅物理/历史本科、含最低位次）。表头驱动。
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
	return core.HasCell(r, "院校代码") &&
		(core.HasCell(r, "专业名称") || core.HasCell(r, "专业")) && // 部分省计划表列名为「专业」而非「专业名称」
		(core.HasCell(r, "招生人数") || core.HasCell(r, "计划人数"))
}

// ParsePlan 解析「招生计划」xlsx → 计划行（仅物理/历史本科）。表头驱动。GroupCode 取专业组代码
// 或所属专业组（双兜底）；专业名带括号尾注的用 StripParenTail 截断以按裸名挂接录取分数表。
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
	// 学制/学费的表头带单位后缀且各省不一（学制 / 学制(年)；学费 / 学费(元) / 学费(元/年)），用 ColContains 容错。
	cSchooling, cTuition := s.ColContains("学制"), s.ColContains("学费")

	var out []core.PlanRow
	for _, r := range s.Data {
		track := canonTrack(core.Cell(r, cTrack))
		if !keep[track] || !batchKeep(core.Cell(r, cBatch)) {
			continue
		}
		// 招生计划专业名带「（包含专业：…）（XX校区）」等尾注，录取分数表用裸名——截断以挂接。
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
			gname = gcode // 源表无独立组名时用组代码兜底
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

// ParseYiFenYiDuan 解析一分一段 xlsx（单文件含物理/历史，本科批），按 年×科类 分组。
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
	cControl := s.ColContains("控制线") // 本科批控制线（特控线），源表自带；缺列则 -1

	byTrack := map[string]*core.YiFenYiDuan{}
	var order []string
	for _, r := range s.Data {
		track := canonTrack(core.Cell(r, cTrack))
		batch := core.Cell(r, cBatch)
		if !keep[track] || !batchKeep(batch) {
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
		// 控制线只取主「本科批」（非提前批），同年同科类各行相同，取首个即可。
		if y.ControlLine == 0 && strings.Contains(batch, "本科批") {
			if cl, ok := core.ParseLeadingInt(core.Cell(r, cControl)); ok {
				y.ControlLine = cl
			}
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
