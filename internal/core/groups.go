package core

import "sort"

// PlanRow 是招生计划里的一行（某年某院校专业组下的一个专业）。3+1+2 省份（江苏/黑龙江…）
// 共用此结构；各省自己的 xlsx 解析循环（因省而异）产出它，聚合（BuildGroups2026）共用。
type PlanRow struct {
	Year       int
	Track      string
	Batch      string // 批次/招生类型：组模型省份留空；浙江(major)落「招生类型」以还原中外合作判定
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

	fn string // 内部：原始 full_name，供组内同名碰撞消歧（追加包含专业提示），不序列化
}

// Group2026 是一个院校专业组的单年报考视图。
type Group2026 struct {
	GroupCode string       `json:"groupCode"`
	GroupName string       `json:"groupName"`
	Track     string       `json:"track"`
	SelKe     string       `json:"selKe"` // 组内统一选科要求（不统一则为空）
	Majors    []GroupMajor `json:"majors"`
}

// disambiguateGroupMajors 就地消解同组内的显示名碰撞——只对真碰撞生效，裸名唯一的专业不动：
//  一级：同名行各自追加「书院/方向/班名」（清华 9 书院、中科大各英才班）；
//  二级：追加后仍同名的（同班名、仅包含专业不同，如中科大两个拔尖计划科技英才班）再追加「包含」首专业
//        → 理科试验班类(拔尖计划科技英才班·数学类) / (…·化学类)。
func disambiguateGroupMajors(majors []GroupMajor) {
	collisions := func() [][]int {
		byName := map[string][]int{}
		for i := range majors {
			byName[majors[i].MajorName] = append(byName[majors[i].MajorName], i)
		}
		var out [][]int
		for _, idxs := range byName {
			if len(idxs) > 1 {
				out = append(out, idxs)
			}
		}
		return out
	}
	for _, idxs := range collisions() { // 一级：书院/方向/班名
		for _, i := range idxs {
			majors[i].MajorName = augmentReportName(majors[i].MajorName, DirectionQualifier(majors[i].fn))
		}
	}
	for _, idxs := range collisions() { // 二级：包含首专业
		for _, i := range idxs {
			majors[i].MajorName = augmentReportName(majors[i].MajorName, containHead(majors[i].fn))
		}
	}
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

// LeafLatestForTrack 返回叶子在指定科类下最近年份的数据点；该科类无往年线则回退到全科类最近点
// （3+1+2 下同名专业可能仅在另一科类有往年录取线，回退好过完全无位次）。组模型（BuildGroups2026）
// 与专业平行志愿模型（cmd 的 buildPlanMajorsTracked）共用这一往年位次挂接逻辑。
func LeafLatestForTrack(l *MajorLeaf, track string) *YearScore {
	var best *YearScore
	for i := range l.Years {
		if l.Years[i].Track != track {
			continue
		}
		if best == nil || l.Years[i].Year >= best.Year {
			best = &l.Years[i]
		}
	}
	if best == nil {
		return leafLatest(l)
	}
	return best
}

// BuildGroups2026 用计划行自建身份归并器聚合组视图。详见 BuildGroups2026R。
func BuildGroups2026(plan []PlanRow, leaves []MajorLeaf, totals map[YearTrack]int, menlei func(string) string) map[string][]Group2026 {
	return BuildGroups2026R(plan, leaves, BuildSchoolResolver(IdentRowsFromPlan(plan)), totals, menlei)
}

// BuildGroups2026R 把招生计划行按 院校实体→渠道→组 聚合成单年视图，并用 leaves（按
// 院校实体键+渠道+专业名）挂接每个组内专业的往年最低位次与等效位次。menlei（可为 nil）把专业名
// 归到学科门类码。返回 院校实体键 → 组列表（ADR-0021：主键是归一化校名，不是院校代号）。
func BuildGroups2026R(plan []PlanRow, leaves []MajorLeaf, r *SchoolResolver, totals map[YearTrack]int, menlei func(string) string) map[string][]Group2026 {
	leafIdx := map[string]*MajorLeaf{}
	for i := range leaves {
		leafIdx[leaves[i].SchoolKey+"/"+leaves[i].SchoolCode+"/"+leaves[i].MajorKey] = &leaves[i]
	}

	// 组在 3+1+2 下按 (渠道, 科类, 组代码) 一等：同校同号的物理组与历史组是两个组；普通/专项两渠道
	// 的同号组也须分开（故键含渠道代表代号）。个别省组代码在两科类间复用，缺科类会把历史专业并进物理组。
	type gkey struct{ channel, track, group string }
	order := []gkey{}
	groups := map[gkey]*Group2026{}
	gkeyEnt := map[gkey]string{} // gkey -> 院校实体键（输出归拢用）

	for _, row := range plan {
		ent := r.Entity(row.SchoolName)
		ch := r.Channel(row.SchoolName, row.SchoolCode)
		k := gkey{ch, row.Track, row.GroupCode}
		g := groups[k]
		if g == nil {
			g = &Group2026{GroupCode: row.GroupCode, GroupName: row.GroupName, Track: row.Track, SelKe: row.SelKe}
			groups[k] = g
			gkeyEnt[k] = ent
			order = append(order, k)
		}
		if g.SelKe != row.SelKe {
			g.SelKe = "" // 组内选科不统一
		}
		gm := GroupMajor{
			MajorName: NormalizeMajorName(row.MajorName),
			MajorKey:  MajorKey(row.MajorName),
			SelKe:     row.SelKe,
			Plan:      row.Plan,
			Tuition:   row.Tuition,
			Coop:      IsCoop(row.MajorName, row.FullName, row.Remark),
			fn:        row.FullName,
		}
		if menlei != nil {
			gm.Menlei = menlei(row.MajorName)
		}
		if lf := leafIdx[ent+"/"+ch+"/"+gm.MajorKey]; lf != nil {
			if p := LeafLatestForTrack(lf, row.Track); p != nil {
				gm.PrevYear = p.Year
				gm.PrevRank = p.MinRank
				gm.EquivRank = EquivRank(p.MinRank,
					YearTrack{Year: p.Year, Track: p.Track}, YearTrack{Year: row.Year, Track: row.Track}, totals)
			}
		}
		g.Majors = append(g.Majors, gm)
	}

	// 组内同名碰撞消歧：首括号相同、仅「包含专业」不同的行（中科大「拔尖计划科技英才班」数理 vs 化生地），
	// 追加包含专业提示使其可区分。
	for _, g := range groups {
		disambiguateGroupMajors(g.Majors)
	}

	out := map[string][]Group2026{}
	for _, k := range order {
		ent := gkeyEnt[k]
		out[ent] = append(out[ent], *groups[k])
	}
	for ent := range out {
		sort.Slice(out[ent], func(i, j int) bool {
			if out[ent][i].Track != out[ent][j].Track {
				return out[ent][i].Track < out[ent][j].Track
			}
			return out[ent][i].GroupCode < out[ent][j].GroupCode
		})
	}
	return out
}
