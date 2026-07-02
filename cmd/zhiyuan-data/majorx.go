package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/store"
	"github.com/sunfmin/zhiyuanwiki/internal/zj"
)

// buildMajorBundle 是「专业平行志愿（无院校专业组）」省份的通用投影（重庆/贵州/辽宁/山东/河北/浙江…）：
// 录取分数→院校×专业叶子、招生计划(最新年)→院校×专业报考视图（按 (院校,专业,选科,科类) 合并、
// 按 (院校,专业名,科类) 挂往年位次）、全国院校属性(按校名)→过滤属性。综合(浙江/山东)与双科类(重庆/辽宁)
// 同走此路：科类逐行带、单科类「综合」是其退化情形。见 ADR-0014 / ADR-0022（浙江归一到此路）。
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

	// 身份归并覆盖 分数∪计划（同 buildDBBundle，ADR-0021）：院校全集、按校名归并、按渠道拆分。
	resolver := core.BuildSchoolResolver(append(core.IdentRowsFromScores(scores), core.IdentRowsFromPlan(planAll)...))
	schools, leaves := core.AggregateLeavesR(scores, resolver)
	if rn := resolver.Renames(); len(rn) > 0 {
		fmt.Printf("  改名/转设归并 %d 处（人工可复核）：\n", len(rn))
		for _, s := range rn {
			fmt.Printf("    · %s\n", s)
		}
	}
	planByEntity := buildPlanMajorsTracked(plan, leaves, resolver, totals, refYear, menlei.Code)

	byCode := map[string][]core.MajorLeaf{}
	for _, lf := range leaves {
		byCode[leafGroupKey(lf)] = append(byCode[leafGroupKey(lf)], lf)
	}

	b := schoolBundle{
		schools: schools, leaves: leaves,
		details: map[string]schoolDetail{},
		meta:    map[string]schoolMetaOut{},
		levels:  map[string][3]bool{},
	}
	planCount, matched := 0, 0
	for _, s := range schools {
		d := schoolDetail{School: s, Leaves: nonNilLeaves(byCode[schoolKey(s)]), Plan2026: planByEntity[schoolKey(s)]}
		planCount += len(d.Plan2026)
		b.details[schoolKey(s)] = d
		if m, ok := idx.Lookup(s.Name); ok {
			matched++
			b.levels[schoolKey(s)] = [3]bool{m.Is985, m.Is211, m.Syl}
			b.meta[schoolKey(s)] = schoolMetaOut{
				Province: m.Province, City: m.City, CityTier: core.CityTier(m.City),
				Owner: m.Ownership, Kind: m.Kind, Levels: levelsOf(m),
			}
		}
	}
	fmt.Printf("  报考视图：院校×专业 %d 个（计划年 %d）· 院校属性命中 %d/%d\n",
		planCount, planYear(plan), matched, len(schools))
	return b
}

// buildPlanMajorsTracked 把招生计划逐专业聚合成院校报考视图，按 (渠道,专业,选科,科类) 合并计划，
// 并按 院校实体键+渠道+专业名 在同科类下挂往年最低位次/等效位次。双科类省同一专业在物理/历史各成一条。
// 返回 院校实体键 → 列表（ADR-0021：按归一化校名归拢，同名多渠道并入一页）。
func buildPlanMajorsTracked(plan []core.PlanRow, leaves []core.MajorLeaf, r *core.SchoolResolver, totals map[core.YearTrack]int, refYear int, menlei func(string) string) map[string][]zj.PlanMajor {
	leafIdx := map[string]*core.MajorLeaf{}
	for i := range leaves {
		leafIdx[leaves[i].Key()] = &leaves[i]
	}

	type mkey struct{ channel, major, selke, track string }
	order := []mkey{}
	mkeyEnt := map[mkey]string{}
	seen := map[mkey]*zj.PlanMajor{}

	for _, row := range plan {
		ent := r.Entity(row.SchoolName)
		ch := r.Channel(row.SchoolName, row.SchoolCode)
		key := core.MajorKey(row.MajorName)
		k := mkey{ch, key, row.SelKe, row.Track}
		if pm := seen[k]; pm != nil {
			pm.Plan += row.Plan
			continue
		}
		pm := &zj.PlanMajor{
			MajorName: core.NormalizeMajorName(row.MajorName),
			MajorKey:  key,
			Track:     row.Track,
			SelKe:     row.SelKe,
			Plan:      row.Plan,
			Tuition:   row.Tuition,
			Schooling: row.Schooling,
			Coop:      core.IsCoop(row.MajorName, row.FullName, row.Remark),
		}
		if menlei != nil {
			pm.Menlei = menlei(row.MajorName)
		}
		if lf := leafIdx[core.LeafKey(ent, ch, key)]; lf != nil {
			if ys := core.LeafLatestForTrack(lf, row.Track); ys != nil {
				pm.PrevYear = ys.Year
				pm.PrevRank = ys.MinRank
				// 只有分数省（西藏）的定位基准。仅在「同科类」有录取史时才挂分——理/文分数不可比，
				// LeafLatestForTrack 在本科类无史时会回退到另一科类，那条分数对本科类无意义、不能用作定位。
				// （有位次省的 PrevRank/EquivRank 维持原回退行为不变；PrevScore 它们用不到。）
				if ys.Track == row.Track {
					pm.PrevScore = ys.MinScore
				}
				pm.EquivRank = core.EquivRank(ys.MinRank,
					core.YearTrack{Year: ys.Year, Track: ys.Track},
					core.YearTrack{Year: refYear, Track: row.Track}, totals)
			}
		}
		seen[k] = pm
		mkeyEnt[k] = ent
		order = append(order, k)
	}

	out := map[string][]zj.PlanMajor{}
	byEnt := map[string][]mkey{}
	for _, k := range order {
		byEnt[mkeyEnt[k]] = append(byEnt[mkeyEnt[k]], k)
	}
	for ent, keys := range byEnt {
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
			// 无位次（西藏「只有分数」）：按最低分降序（分高=最难在前），与有位次省的「位次升序」同序意。
			if si, sj := list[i].PrevScore, list[j].PrevScore; ri == 0 && si != sj {
				return si > sj
			}
			if list[i].Track != list[j].Track {
				return list[i].Track < list[j].Track
			}
			return list[i].MajorName < list[j].MajorName
		})
		out[ent] = list
	}
	return out
}
