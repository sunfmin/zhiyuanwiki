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

// keep 收口本站新高考省的默认科类：3+1+2 省的 物理/历史，以及 3+3「综合+院校专业组」省（北京/上海/
// 海南）的 综合。理科/文科（老文理）与艺术/体育不在内，会被过滤掉。老文理省（新疆，internal/xj）
// 走 *With 变体传 {理科,文科}——不并入此默认集，否则会把重庆/贵州等省 22-24 年的老文理历史行
// 一并吸入（那些 major 省正靠默认 keep 过滤掉改革前年份）。见 issue #27、ADR-0014。
var keep = map[string]bool{"物理": true, "历史": true, "综合": true}

// canonTrack 把源表科类归一为站点口径：物理类→物理、历史类→历史；综合/裸物理/裸历史 原样保留；
// 河北招生计划用「物理科目组合/历史科目组合」也归一。其余（艺术类（物理）/理科/文科 等）原样
// 返回后被 keep 过滤掉。
func canonTrack(s string) string {
	s = strings.TrimSpace(s)
	switch s {
	case "物理类", "物理科目组合":
		return "物理"
	case "历史类", "历史科目组合":
		return "历史"
	}
	return s
}

// batchKeep：留本科批次及等价主批——本科（本科批/本科提前批）、综合改革省的普通类一段/二段
// （山东）、常规批（山东计划侧）。丢专科/高职/艺术/体育批（艺术/体育已由科类过滤）。
func batchKeep(batch string) bool {
	return strings.Contains(batch, "本科") ||
		strings.Contains(batch, "一段") ||
		strings.Contains(batch, "二段") ||
		strings.Contains(batch, "常规批")
}

func scoreHeader(r []string) bool {
	return core.HasCell(r, "院校代码") && core.HasCell(r, "最低位次")
}

// scoreHeaderScoreOnly 是「只有分数」省份（西藏）的表头探测：不要求「最低位次」列（西藏该列全空、
// 甚至可能整列缺失），只认 院校代码 + 最低分数（或裸「最低分」）。见 ParseScoresScoreOnly。
func scoreHeaderScoreOnly(r []string) bool {
	return core.HasCell(r, "院校代码") && (core.HasCell(r, "最低分数") || core.HasCell(r, "最低分"))
}

// ParseScores 解析「专业录取分数」xlsx → 行表（仅物理/历史/综合本科、含最低位次）。表头驱动。
func ParseScores(path string) ([]core.MajorScoreRow, error) {
	return ParseScoresWith(path, keep)
}

// ParseScoresWith 同 ParseScores，但用调用方给定的 keep 科类集合（老文理省 internal/xj 传 {理科,文科}）。
func ParseScoresWith(path string, keep map[string]bool) ([]core.MajorScoreRow, error) {
	s, err := core.OpenSheet(path, scoreHeader)
	if err != nil {
		return nil, err
	}
	return parseScores(s, keep, false, true), nil
}

// ParseScoresScoreOnly 解析「只有分数、没有位次」省份的专业录取分数（西藏：考试院不发布一分一段、
// 录取数据无最低位次列）。与 ParseScoresWith 同形，但**不丢无位次行**（MinRank 落 0），改以「最低分数
// 非空」为入库门槛。下游靠 PrevRank==0 && PrevScore>0 走分数域投影/定位（见 majorx/dingwei）。
func ParseScoresScoreOnly(path string, keep map[string]bool) ([]core.MajorScoreRow, error) {
	s, err := core.OpenSheet(path, scoreHeaderScoreOnly)
	if err != nil {
		return nil, err
	}
	return parseScores(s, keep, false, false), nil
}

// ParseScoresAvg 同 ParseScores，但取「平均分/平均位次」列作为录取参考分（MinScore/MinRank），
// 而非最低分/最低位次。上海专用：上海官方把本科批最低分封顶在 580（高分段不披露，580 分以上一律
// 记 580 分 / 4096 位），导致复旦/上交/浙大等所有 top 校最低分相同、600 分考生全挤进「过保」。
// 而源表「平均分/平均位次」逐专业真实且未封顶（如复旦工科试验班 平均 619 / 119 位），是上海唯一
// 能区分专业竞争度的口径。故上海以平均分入库，下游等效分/冲稳保排序据此分档。源表无平均列时
// 回退到最低分（不致整省丢行）。见 issue。
func ParseScoresAvg(path string) ([]core.MajorScoreRow, error) {
	s, err := core.OpenSheet(path, scoreHeader)
	if err != nil {
		return nil, err
	}
	return parseScores(s, keep, true, true), nil
}

// parseScores：useAvg 取平均分/平均位次（上海口径）；requireRank=true 时丢无位次行（多数省），
// requireRank=false 时保留无位次行（MinRank=0）、改以「最低分数非空」为门槛（西藏 only-score）。
func parseScores(s *core.Sheet, keep map[string]bool, useAvg, requireRank bool) []core.MajorScoreRow {
	col := s.Col
	cYear, cTrack, cBatch := col("年份"), col("科类"), col("批次")
	cCode, cName := col("院校代码"), col("院校名称")
	cGroup := col("所属专业组", "专业组代码") // 上海录取分数表组列名为「专业组代码」（无「所属专业组」）
	cMajor, cSelKe := col("专业", "专业名称"), col("选科要求")
	cMin, cRank := col("最低分数", "最低分"), col("最低位次")
	if useAvg { // 上海口径：改取平均分/平均位次（缺列则回退最低分，避免整省丢行）
		if c := col("平均分"); c >= 0 {
			cMin = c
		}
		if c := col("平均位次"); c >= 0 {
			cRank = c
		}
	}

	var out []core.MajorScoreRow
	for _, r := range s.Data {
		track := canonTrack(core.Cell(r, cTrack))
		if !keep[track] || !batchKeep(core.Cell(r, cBatch)) {
			continue
		}
		minRank, hasRank := core.ParseLeadingInt(core.Cell(r, cRank))
		if requireRank && !hasRank {
			continue
		}
		name := strings.TrimSpace(core.Cell(r, cMajor))
		code := core.NormSchoolCode(core.Cell(r, cCode))
		if name == "" || code == "" {
			continue
		}
		year, _ := core.ParseLeadingInt(core.Cell(r, cYear))
		minScore, hasScore := core.ParseLeadingInt(core.Cell(r, cMin))
		if !requireRank && !hasScore {
			continue // 只有分数省：无最低分数的行无用，丢弃
		}
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
		(core.HasCell(r, "招生人数") || core.HasCell(r, "计划人数") || core.HasCell(r, "计划数")) // 江西计划列名为「计划数」
}

// ParsePlan 解析「招生计划」xlsx → 计划行（仅物理/历史/综合本科）。表头驱动。GroupCode 取专业组代码
// 或所属专业组（双兜底）；专业名带括号尾注的用 StripParenTail 截断以按裸名挂接录取分数表。
func ParsePlan(path string) ([]core.PlanRow, error) {
	return ParsePlanWith(path, keep)
}

// ParsePlanWith 同 ParsePlan，但用调用方给定的 keep 科类集合（老文理省 internal/xj 传 {理科,文科}）。
func ParsePlanWith(path string, keep map[string]bool) ([]core.PlanRow, error) {
	s, err := core.OpenSheet(path, planHeader)
	if err != nil {
		return nil, err
	}
	return parsePlan(s, keep), nil
}

func parsePlan(s *core.Sheet, keep map[string]bool) []core.PlanRow {
	col := s.Col
	cYear, cTrack, cBatch := col("年份"), col("科类", "文理"), col("批次") // 甘肃计划表 track 列名为「文理」
	cCode, cName := col("院校代码"), col("院校名称")
	// 组代码列名各省不一：所属专业组/专业组代码（多数）、专业组（内蒙）、组代码（甘肃）；吉林只有
	// 「专业组名称」（如「第 001 组」），下方 gcode 为空时用组名兜底建组。
	cGroupCode, cGroupName := col("专业组代码", "所属专业组", "专业组", "组代码"), col("专业组名称")
	cMajor, cSelKe := col("专业名称", "专业"), col("选科要求", "选课要求", "选科") // 江西「选课要求」/河北「选科」
	cRemark := col("专业备注", "备注")
	cPlan := col("计划人数", "招生人数", "计划数") // 江西计划列名为「计划数」
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
		if gcode == "" {
			gcode = gname // 计划表无组代码列、只有组名（如吉林「第 001 组」）时用组名当组码建组
		}
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

// ParseYiFenYiDuan 解析一分一段 xlsx（单文件含物理/历史），按 年×科类 分组。
// 表头带单位后缀（分数(分)/本段人数(人)/累计人数(人)），列定位走 ColContains。
//
// 一分一段是「每科类一条全省连续排名」。多数省源表把它按分数区间切成「本科批」（本科线→最高分）
// 与「专科批」（最低分→本科线下一分）两个**互补不重叠**的段——二者拼起来才是该科类**全体统考排名
// 考生**。故这里**不**按批次过滤（批次过滤只用于录取分数/招生计划），否则 MAX(累计)=本科上线人数而非
// 高考人数（小省如青海会出现「本科计划 > 高考人数」的悖论）。科类已由 keep 把艺术/体育挡在外。
func ParseYiFenYiDuan(path, province string, year int) ([]*core.YiFenYiDuan, error) {
	return ParseYiFenYiDuanWith(path, province, year, keep)
}

// ParseYiFenYiDuanWith 同 ParseYiFenYiDuan，但用调用方给定的 keep 科类集合（老文理省 internal/xj
// 传 {理科,文科}）。
func ParseYiFenYiDuanWith(path, province string, year int, keep map[string]bool) ([]*core.YiFenYiDuan, error) {
	s, err := core.OpenSheet(path, yfdHeader)
	if err != nil {
		return nil, err
	}
	return parseYiFenYiDuan(s, province, year, keep), nil
}

func parseYiFenYiDuan(s *core.Sheet, province string, year int, keep map[string]bool) []*core.YiFenYiDuan {
	cTrack, cBatch := s.Col("科类"), s.Col("批次")
	cScore := s.ColContains("分数", "分段")
	cCount, cCum := s.ColContains("本段人数"), s.ColContains("累计")
	cControl := s.ColContains("控制线") // 本科批控制线（特控线），源表自带；缺列则 -1

	byTrack := map[string]*core.YiFenYiDuan{}
	seen := map[string]map[int]bool{} // 科类 → 已收分数：本科批/专科批两段互补不重叠，防边界同分重复
	var order []string
	for _, r := range s.Data {
		track := canonTrack(core.Cell(r, cTrack))
		// 不按批次过滤：本科批 + 专科批 两段拼成该科类的完整排名（见函数头注）。科类 keep 已挡艺术/体育。
		if !keep[track] {
			continue
		}
		batch := core.Cell(r, cBatch)
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
			seen[track] = map[int]bool{}
			order = append(order, track)
		}
		// 控制线只取主「本科批」（非提前批），同年同科类各行相同，取首个即可。
		if y.ControlLine == 0 && strings.Contains(batch, "本科批") {
			if cl, ok := core.ParseLeadingInt(core.Cell(r, cControl)); ok {
				y.ControlLine = cl
			}
		}
		if seen[track][score] {
			continue // 同分已收（本科/专科段边界处偶有重复），保留首个
		}
		seen[track][score] = true
		y.Entries = append(y.Entries, core.FenduanEntry{Score: score, Count: count, Cumulative: cum})
	}
	out := make([]*core.YiFenYiDuan, 0, len(order))
	for _, t := range order {
		core.SortFenduanAscending(byTrack[t])
		out = append(out, byTrack[t])
	}
	return out
}
