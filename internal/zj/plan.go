package zj

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/xuri/excelize/v2"
)

// PlanRow2026 是 2026 招生计划里的一行（某院校某专业，浙江按专业平行志愿逐专业列出）。
type PlanRow2026 struct {
	Year       int
	SchoolCode string
	SchoolName string
	AdmitType  string // 招生类型（普通类/中外合作办学/综合评价…）
	MajorName  string
	Remark     string // 专业备注（方向，用于中外合作判定）
	SelKe      string
	Plan       int
	Schooling  string // 学制
	Tuition    string // 学费
}

// ParsePlan2026XLSX 解析 2026 浙江招生计划 xlsx，只返回科类=综合、收录批次的行。表头驱动。
func ParsePlan2026XLSX(path string) ([]PlanRow2026, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("打开 %s: %w", path, err)
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("%s: 无 sheet", path)
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("读 %s: %w", path, err)
	}
	return parsePlan2026Rows(rows)
}

func parsePlan2026Rows(rows [][]string) ([]PlanRow2026, error) {
	headerIdx := -1
	for i, r := range rows {
		if core.HasCell(r, "院校代码") && core.HasCell(r, "专业名称") && core.HasCell(r, "招生人数") {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return nil, fmt.Errorf("未找到含\"院校代码/专业名称/招生人数\"的表头行")
	}
	h := rows[headerIdx]
	col := func(names ...string) int { return core.FindCol(h, names...) }
	cYear := col("年份")
	cTrack := col("科类")
	cBatch := col("批次")
	cSchoolCode := col("院校代码")
	cSchoolName := col("院校名称")
	cAdmit := col("招生类型")
	cMajor := col("专业名称", "专业")
	cRemark := col("专业备注")
	cSelKe := col("选科要求")
	cPlan := col("招生人数", "计划人数")
	cSchooling := col("学制(年)", "学制")
	cTuition := col("学费(元)", "学费")

	var out []PlanRow2026
	for _, r := range rows[headerIdx+1:] {
		if strings.TrimSpace(core.Cell(r, cTrack)) != Track {
			continue
		}
		if !batchKeep(core.Cell(r, cBatch)) {
			continue
		}
		name := strings.TrimSpace(core.Cell(r, cMajor))
		code := core.NormSchoolCode(core.Cell(r, cSchoolCode))
		if name == "" || code == "" {
			continue
		}
		year, _ := core.ParseLeadingInt(core.Cell(r, cYear))
		plan, _ := core.ParseLeadingInt(core.Cell(r, cPlan))
		out = append(out, PlanRow2026{
			Year:       year,
			SchoolCode: code,
			SchoolName: strings.TrimSpace(core.Cell(r, cSchoolName)),
			AdmitType:  strings.TrimSpace(core.Cell(r, cAdmit)),
			MajorName:  name,
			Remark:     strings.TrimSpace(core.Cell(r, cRemark)),
			SelKe:      strings.TrimSpace(core.Cell(r, cSelKe)),
			Plan:       plan,
			Schooling:  strings.TrimSpace(core.Cell(r, cSchooling)),
			Tuition:    strings.TrimSpace(core.Cell(r, cTuition)),
		})
	}
	return out, nil
}

// ── 2026 院校×专业报考视图（专业平行志愿：每个专业独立投档，无组）──

// PlanMajor 是 2026 某院校的一个可填报专业，挂接了往年最低位次与等效位次。
type PlanMajor struct {
	MajorName string `json:"majorName"`
	MajorKey  string `json:"majorKey"`
	SelKe     string `json:"selKe"`
	Plan      int    `json:"plan"`
	Tuition   string `json:"tuition,omitempty"`
	Schooling string `json:"schooling,omitempty"`
	Menlei    string `json:"menlei,omitempty"` // 学科门类 1 字码
	Coop      bool   `json:"coop,omitempty"`   // 中外合作办学
	PrevYear  int    `json:"prevYear,omitempty"`
	PrevRank  int    `json:"prevRank,omitempty"`  // 最近年份最低位次
	EquivRank int    `json:"equivRank,omitempty"` // 等效到 refYear
}

// leafLatest 返回叶子最近年份的数据点。
func leafLatest(l *core.MajorLeaf) *core.YearScore {
	if len(l.Years) == 0 {
		return nil
	}
	best := &l.Years[0]
	for i := range l.Years {
		if l.Years[i].Year >= best.Year {
			best = &l.Years[i]
		}
	}
	return best
}

// BuildPlan2026 把 2026 计划逐专业聚合成院校报考视图，并按院校代码+专业名挂接往年最低位次。
// 同一院校内 (专业名,选科) 相同的多行合并（计划人数相加）。refYear 是等效位次的目标年口径
// （浙江取最近有一分一段的年份，如 2025；2026 高考未考、无一分一段）。返回 院校代码 → 专业列表。
func BuildPlan2026(plan []PlanRow2026, leaves []core.MajorLeaf, totals map[core.YearTrack]int, refYear int, menlei *core.MenleiClassifier) map[string][]PlanMajor {
	leafIdx := map[string]*core.MajorLeaf{}
	for i := range leaves {
		leafIdx[leaves[i].SchoolCode+"/"+leaves[i].MajorKey] = &leaves[i]
	}

	type mkey struct{ school, major, selke string }
	order := map[string][]mkey{} // school -> ordered keys
	seen := map[mkey]*PlanMajor{}

	for _, r := range plan {
		key := core.MajorKey(r.MajorName)
		k := mkey{r.SchoolCode, key, r.SelKe}
		if pm := seen[k]; pm != nil {
			pm.Plan += r.Plan
			continue
		}
		pm := &PlanMajor{
			MajorName: core.NormalizeMajorName(r.MajorName),
			MajorKey:  key,
			SelKe:     r.SelKe,
			Plan:      r.Plan,
			Tuition:   r.Tuition,
			Schooling: r.Schooling,
			Coop:      core.IsCoop(r.MajorName, r.Remark, r.AdmitType),
		}
		if menlei != nil {
			pm.Menlei = menlei.Code(r.MajorName)
		}
		if lf := leafIdx[r.SchoolCode+"/"+key]; lf != nil {
			if p := leafLatest(lf); p != nil {
				pm.PrevYear = p.Year
				pm.PrevRank = p.MinRank
				pm.EquivRank = core.EquivRank(p.MinRank,
					core.YearTrack{Year: p.Year, Track: Track},
					core.YearTrack{Year: refYear, Track: Track}, totals)
			}
		}
		seen[k] = pm
		order[r.SchoolCode] = append(order[r.SchoolCode], k)
	}

	out := map[string][]PlanMajor{}
	for school, keys := range order {
		list := make([]PlanMajor, 0, len(keys))
		for _, k := range keys {
			list = append(list, *seen[k])
		}
		// 有等效位次的按位次升序（最难在前），无位次的按专业名排在后。
		sort.SliceStable(list, func(i, j int) bool {
			ri, rj := list[i].EquivRank, list[j].EquivRank
			if (ri > 0) != (rj > 0) {
				return ri > 0
			}
			if ri != rj && ri > 0 {
				return ri < rj
			}
			return list[i].MajorName < list[j].MajorName
		})
		out[school] = list
	}
	return out
}
