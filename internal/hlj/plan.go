package hlj

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

// PlanRow 是招生计划里的一行（某年某院校专业组下的一个专业）。
type PlanRow struct {
	Year       int
	Track      string
	SchoolCode string
	SchoolName string
	GroupCode  string // 专业组代码（3 位，逐年变）
	GroupName  string
	MajorName  string
	FullName   string // 专业全称（用于中外合作判定）
	Remark     string // 专业备注（用于中外合作判定）
	SelKe      string
	Plan       int
	Schooling  string // 学制
	Tuition    string // 学费
}

// ParsePlanXLSX 解析招生计划 xlsx，只返回新科类（物理/历史）本科批行。表头驱动。
func ParsePlanXLSX(path string) ([]PlanRow, error) {
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
	return parsePlanRows(rows)
}

func parsePlanRows(rows [][]string) ([]PlanRow, error) {
	headerIdx := -1
	for i, r := range rows {
		if core.HasCell(r, "院校代码") && core.HasCell(r, "专业名称") && core.HasCell(r, "计划人数") {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return nil, fmt.Errorf("未找到含\"院校代码/专业名称/计划人数\"的表头行")
	}
	h := rows[headerIdx]
	col := func(names ...string) int { return core.FindCol(h, names...) }
	cYear, cTrack, cBatch := col("年份"), col("科类"), col("批次")
	cSchoolCode, cSchoolName := col("院校代码"), col("院校名称")
	cGroup, cGroupName := col("专业组代码"), col("专业组名称")
	cMajor, cSelKe := col("专业名称"), col("选科要求")
	cFull, cRemark := col("专业全称"), col("专业备注")
	cPlan, cSchooling, cTuition := col("计划人数"), col("学制"), col("学费")

	var out []PlanRow
	for _, r := range rows[headerIdx+1:] {
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
	return out, nil
}

// ── 2026 院校专业组视图（组 = 单年视图；历史由组内专业按院校+专业名挂接）──

// GroupMajor 是 2026 组内的一个专业，挂接了往年最低位次。
type GroupMajor struct {
	MajorName string `json:"majorName"`
	MajorKey  string `json:"majorKey"`
	SelKe     string `json:"selKe"`
	Plan      int    `json:"plan"`
	Tuition   string `json:"tuition"`
	Menlei    string `json:"menlei,omitempty"`    // 学科门类 1 字码（见 menlei.go），未命中省略
	Coop      bool   `json:"coop,omitempty"`      // 中外合作办学
	PrevYear  int    `json:"prevYear,omitempty"`  // 挂接到的最近年份
	PrevRank  int    `json:"prevRank,omitempty"`  // 该年最低位次
	EquivRank int    `json:"equivRank,omitempty"` // 等效到 planYear
}

// Group2026 是一个院校专业组的单年报考视图。
type Group2026 struct {
	GroupCode string       `json:"groupCode"`
	GroupName string       `json:"groupName"`
	Track     string       `json:"track"`
	SelKe     string       `json:"selKe"` // 组内统一选科要求（不统一则为空）
	Majors    []GroupMajor `json:"majors"`
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

// BuildGroups2026 把招生计划行按院校→组聚合成单年视图，并用 leaves（按院校代码+专业名）
// 挂接每个组内专业的往年最低位次与等效位次。menlei（可为 nil）把专业名归到学科门类码。
// 返回 院校代码 → 组列表。
func BuildGroups2026(plan []PlanRow, leaves []core.MajorLeaf, totals map[core.YearTrack]int, menlei func(string) string) map[string][]Group2026 {
	leafIdx := map[string]*core.MajorLeaf{}
	for i := range leaves {
		leafIdx[leaves[i].SchoolCode+"/"+leaves[i].MajorKey] = &leaves[i]
	}

	type gkey struct{ school, group string }
	order := []gkey{}
	groups := map[gkey]*Group2026{}

	for _, r := range plan {
		k := gkey{r.SchoolCode, r.GroupCode}
		g := groups[k]
		if g == nil {
			g = &Group2026{GroupCode: r.GroupCode, GroupName: r.GroupName, Track: r.Track, SelKe: r.SelKe}
			groups[k] = g
			order = append(order, k)
		}
		if g.SelKe != r.SelKe {
			g.SelKe = "" // 组内选科不统一
		}
		gm := GroupMajor{
			MajorName: core.NormalizeMajorName(r.MajorName),
			MajorKey:  core.MajorKey(r.MajorName),
			SelKe:     r.SelKe,
			Plan:      r.Plan,
			Tuition:   r.Tuition,
			Coop:      core.IsCoop(r.MajorName, r.FullName, r.Remark),
		}
		if menlei != nil {
			gm.Menlei = menlei(r.MajorName)
		}
		if lf := leafIdx[r.SchoolCode+"/"+gm.MajorKey]; lf != nil {
			if p := leafLatest(lf); p != nil {
				gm.PrevYear = p.Year
				gm.PrevRank = p.MinRank
				gm.EquivRank = core.EquivRank(p.MinRank,
					core.YearTrack{Year: p.Year, Track: p.Track}, core.YearTrack{Year: r.Year, Track: r.Track}, totals)
			}
		}
		g.Majors = append(g.Majors, gm)
	}

	out := map[string][]Group2026{}
	for _, k := range order {
		out[k.school] = append(out[k.school], *groups[k])
	}
	// 组内按代码、组按代码排序，稳定输出
	for code := range out {
		sort.Slice(out[code], func(i, j int) bool { return out[code][i].GroupCode < out[code][j].GroupCode })
	}
	return out
}
