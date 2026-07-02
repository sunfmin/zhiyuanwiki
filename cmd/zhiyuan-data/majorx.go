package main

import (
	"sort"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/zj"
)

// projectPlanMajors 是「专业平行志愿（无院校专业组）」省份的通用投影核心（重庆/贵州/辽宁/山东/河北/浙江…）：
// 招生计划(最新年)→院校×专业报考视图（按 (院校,专业,选科,科类) 合并、按 (院校,专业名,科类) 挂往年位次）。
// 综合(浙江/山东)与双科类(重庆/辽宁)同走此路：科类逐行带、单科类「综合」是其退化情形。
// 作为 projectFn 传入共享骨架 buildBundle（见 yuanxiao_db.go）。见 ADR-0014 / ADR-0022（浙江归一到此路）。
func projectPlanMajors(plan []core.PlanRow, leaves []core.MajorLeaf, r *core.SchoolResolver, totals map[core.YearTrack]int, scores []core.MajorScoreRow, menlei func(string) string) projection {
	refYear := planYear(plan)
	if refYear == 0 {
		refYear = latestScoreYear(scores)
	}
	planByEntity := buildPlanMajorsTracked(plan, leaves, r, totals, refYear, menlei)
	return projection{
		label: "院校×专业",
		assign: func(d *schoolDetail, key string) int {
			d.Plan2026 = planByEntity[key]
			return len(d.Plan2026)
		},
	}
}

// buildPlanMajorsTracked 把招生计划逐专业聚合成院校报考视图，按 (渠道,专业,选科,科类) 合并计划，
// 并按 院校实体键+渠道+专业名 在同科类下挂往年最低位次/等效位次。双科类省同一专业在物理/历史各成一条。
// 返回 院校实体键 → 列表（ADR-0021：按归一化校名归拢，同名多渠道并入一页）。
func buildPlanMajorsTracked(plan []core.PlanRow, leaves []core.MajorLeaf, r *core.SchoolResolver, totals map[core.YearTrack]int, refYear int, menlei func(string) string) map[string][]zj.PlanMajor {
	leafIdx := map[string]*core.MajorLeaf{}
	for i := range leaves {
		leafIdx[leaves[i].SchoolKey+"/"+leaves[i].SchoolCode+"/"+leaves[i].MajorKey] = &leaves[i]
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
		if lf := leafIdx[ent+"/"+ch+"/"+key]; lf != nil {
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
