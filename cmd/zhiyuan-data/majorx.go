package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/store"
	"github.com/sunfmin/zhiyuanwiki/internal/zj"
)

// buildMajorBundle 是「专业平行志愿（无院校专业组）」省份的通用投影（重庆/贵州/辽宁/山东/河北）：
// 录取分数→院校×专业叶子、招生计划(最新年)→院校×专业报考视图（按 (院校,专业,选科,科类) 合并、
// 按 (院校,专业名,科类) 挂往年位次）、全国院校属性(按校名)→过滤属性。与浙江 buildDBBundleMajor 同形，
// 但 (1) 院校属性走全国 school 表（按校名）而非浙江一表联动按码表；(2) 双科类（物理/历史）逐行带科类，
// 不像浙江硬编码单科类「综合」。见 ADR-0014。
func buildMajorBundle(dbPath string, p province) schoolBundle {
	db, err := store.Open(dbPath)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	scores, err := db.LoadScores(p.slug)
	if err != nil {
		fatal(err)
	}
	if len(scores) == 0 {
		fatal(fmt.Errorf("DB 无%s分数行——先跑 `zhiyuan-data import -prov %s`", p.name, p.slug))
	}
	fmt.Printf("  专业录取分数：%d 行（%s·含位次）\n", len(scores), strings.Join(p.tracks, "/"))
	schools, leaves := core.AggregateLeaves(scores)

	idx, err := db.SchoolIndex()
	if err != nil {
		fatal(err)
	}
	menlei, err := db.Menlei()
	if err != nil {
		fatal(err)
	}
	fmt.Printf("  全国院校属性 %d 所 · 专业→门类 %d 条\n", idx.Len(), menlei.Len())

	totals, err := db.LoadTotals(p.slug)
	if err != nil {
		fatal(err)
	}
	planAll, err := db.LoadPlan(p.slug)
	if err != nil {
		fatal(err)
	}
	plan := latestPlanYear(planAll)
	if planYear(plan) == 0 {
		if sy := latestScoreYear(scores); sy > 0 {
			for i := range plan {
				plan[i].Year = sy
			}
		}
	}
	refYear := planYear(plan)
	if refYear == 0 {
		refYear = latestScoreYear(scores)
	}
	planByCode := buildPlanMajorsTracked(plan, leaves, totals, refYear, menlei.Code)

	byCode := map[string][]core.MajorLeaf{}
	for _, lf := range leaves {
		byCode[lf.SchoolCode] = append(byCode[lf.SchoolCode], lf)
	}

	b := schoolBundle{
		schools: schools, leaves: leaves,
		details: map[string]schoolDetail{},
		meta:    map[string]schoolMetaOut{},
		levels:  map[string][3]bool{},
	}
	planCount, matched := 0, 0
	for _, s := range schools {
		d := schoolDetail{School: s, Leaves: byCode[s.Code], Plan2026: planByCode[s.Code]}
		planCount += len(d.Plan2026)
		b.details[s.Code] = d
		if m, ok := idx.Lookup(s.Name); ok {
			matched++
			b.levels[s.Code] = [3]bool{m.Is985, m.Is211, m.Syl}
			b.meta[s.Code] = schoolMetaOut{
				Province: m.Province, City: m.City, CityTier: core.CityTier(m.City),
				Owner: m.Ownership, Kind: m.Kind, Levels: levelsOf(m),
			}
		}
	}
	fmt.Printf("  报考视图：院校×专业 %d 个（计划年 %d）· 院校属性命中 %d/%d\n",
		planCount, planYear(plan), matched, len(schools))
	return b
}

// buildPlanMajorsTracked 把招生计划逐专业聚合成院校报考视图，按 (院校,专业,选科,科类) 合并计划，
// 并按 (院校,专业名) 在同科类下挂往年最低位次/等效位次。双科类省同一专业在物理/历史各成一条。
func buildPlanMajorsTracked(plan []core.PlanRow, leaves []core.MajorLeaf, totals map[core.YearTrack]int, refYear int, menlei func(string) string) map[string][]zj.PlanMajor {
	leafIdx := map[string]*core.MajorLeaf{}
	for i := range leaves {
		leafIdx[leaves[i].SchoolCode+"/"+leaves[i].MajorKey] = &leaves[i]
	}

	type mkey struct{ school, major, selke, track string }
	order := map[string][]mkey{}
	seen := map[mkey]*zj.PlanMajor{}

	for _, r := range plan {
		key := core.MajorKey(r.MajorName)
		k := mkey{r.SchoolCode, key, r.SelKe, r.Track}
		if pm := seen[k]; pm != nil {
			pm.Plan += r.Plan
			continue
		}
		pm := &zj.PlanMajor{
			MajorName: core.NormalizeMajorName(r.MajorName),
			MajorKey:  key,
			Track:     r.Track,
			SelKe:     r.SelKe,
			Plan:      r.Plan,
			Tuition:   r.Tuition,
			Schooling: r.Schooling,
			Coop:      core.IsCoop(r.MajorName, r.FullName, r.Remark),
		}
		if menlei != nil {
			pm.Menlei = menlei(r.MajorName)
		}
		if lf := leafIdx[r.SchoolCode+"/"+key]; lf != nil {
			if ys := leafLatestForTrack(lf, r.Track); ys != nil {
				pm.PrevYear = ys.Year
				pm.PrevRank = ys.MinRank
				pm.EquivRank = core.EquivRank(ys.MinRank,
					core.YearTrack{Year: ys.Year, Track: ys.Track},
					core.YearTrack{Year: refYear, Track: r.Track}, totals)
			}
		}
		seen[k] = pm
		order[r.SchoolCode] = append(order[r.SchoolCode], k)
	}

	out := map[string][]zj.PlanMajor{}
	for school, keys := range order {
		list := make([]zj.PlanMajor, 0, len(keys))
		for _, k := range keys {
			list = append(list, *seen[k])
		}
		sort.SliceStable(list, func(i, j int) bool {
			ri, rj := list[i].EquivRank, list[j].EquivRank
			if (ri > 0) != (rj > 0) {
				return ri > 0
			}
			if ri != rj && ri > 0 {
				return ri < rj
			}
			if list[i].Track != list[j].Track {
				return list[i].Track < list[j].Track
			}
			return list[i].MajorName < list[j].MajorName
		})
		out[school] = list
	}
	return out
}

// leafLatestForTrack 返回叶子在指定科类下最近年份的数据点；该科类无往年线则回退到全科类最近点。
func leafLatestForTrack(l *core.MajorLeaf, track string) *core.YearScore {
	var best *core.YearScore
	for i := range l.Years {
		if l.Years[i].Track != track {
			continue
		}
		if best == nil || l.Years[i].Year >= best.Year {
			best = &l.Years[i]
		}
	}
	if best != nil {
		return best
	}
	for i := range l.Years {
		if best == nil || l.Years[i].Year >= best.Year {
			best = &l.Years[i]
		}
	}
	return best
}
