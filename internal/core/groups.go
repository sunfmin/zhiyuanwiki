package core

import "sort"

// PlanRow 是招生计划里的一行（某年某院校专业组下的一个专业）。3+1+2 省份（江苏/黑龙江…）
// 共用此结构；各省自己的 xlsx 解析循环（因省而异）产出它，聚合（BuildGroups2026）共用。
type PlanRow struct {
	Year       int
	Track      string
	SchoolCode string
	SchoolName string
	GroupCode  string // 专业组代码（逐年变）
	GroupName  string
	MajorName  string
	FullName   string // 专业全称（用于中外合作判定）
	Remark     string // 专业备注（用于中外合作判定）
	SelKe      string
	Plan       int
	Schooling  string // 学制
	Tuition    string // 学费
}

// ── 院校专业组报考视图（组 = 单年视图；历史由组内专业按院校+专业名挂接）──

// GroupMajor 是组内的一个专业，挂接了往年最低位次。
type GroupMajor struct {
	MajorName string `json:"majorName"`
	MajorKey  string `json:"majorKey"`
	SelKe     string `json:"selKe"`
	Plan      int    `json:"plan"`
	Tuition   string `json:"tuition"`
	Menlei    string `json:"menlei,omitempty"`    // 学科门类 1 字码，未命中省略
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
func leafLatest(l *MajorLeaf) *YearScore {
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
func BuildGroups2026(plan []PlanRow, leaves []MajorLeaf, totals map[YearTrack]int, menlei func(string) string) map[string][]Group2026 {
	leafIdx := map[string]*MajorLeaf{}
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
			MajorName: NormalizeMajorName(r.MajorName),
			MajorKey:  MajorKey(r.MajorName),
			SelKe:     r.SelKe,
			Plan:      r.Plan,
			Tuition:   r.Tuition,
			Coop:      IsCoop(r.MajorName, r.FullName, r.Remark),
		}
		if menlei != nil {
			gm.Menlei = menlei(r.MajorName)
		}
		if lf := leafIdx[r.SchoolCode+"/"+gm.MajorKey]; lf != nil {
			if p := leafLatest(lf); p != nil {
				gm.PrevYear = p.Year
				gm.PrevRank = p.MinRank
				gm.EquivRank = EquivRank(p.MinRank,
					YearTrack{Year: p.Year, Track: p.Track}, YearTrack{Year: r.Year, Track: r.Track}, totals)
			}
		}
		g.Majors = append(g.Majors, gm)
	}

	out := map[string][]Group2026{}
	for _, k := range order {
		out[k.school] = append(out[k.school], *groups[k])
	}
	for code := range out {
		sort.Slice(out[code], func(i, j int) bool { return out[code][i].GroupCode < out[code][j].GroupCode })
	}
	return out
}
