package main

import (
	"fmt"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/store"
)

// buildJSBundle 从 SQLite staging 投影江苏院校数据（见 ADR-0014）：专业录取分数→院校×专业叶子、
// 招生计划(最新年)→院校专业组报考视图、全国院校属性(按校名)→过滤属性、全国专业门类→门类码。
// 与 buildHLJBundle 同形，只是数据来源从 xlsx 换成 DB。
func buildJSBundle(dbPath string, p province) schoolBundle {
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
		fatal(fmt.Errorf("DB 无江苏分数行——先跑 `zhiyuan-data import -prov js`"))
	}
	fmt.Printf("  专业录取分数：%d 行（物理/历史·本科·含位次）\n", len(scores))
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
	groupsByCode := core.BuildGroups2026(plan, leaves, totals, menlei.Code)

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
	groupCount, matched := 0, 0
	for _, s := range schools {
		d := schoolDetail{School: s, Leaves: byCode[s.Code], Groups2026: groupsByCode[s.Code]}
		groupCount += len(d.Groups2026)
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
	fmt.Printf("  报考视图：院校专业组 %d 个（计划年 %d）· 院校属性命中 %d/%d\n",
		groupCount, planYear(plan), matched, len(schools))
	return b
}

// levelsOf 把全国院校属性的层次布尔转成数组（与 zj/hlj 的 Levels() 同形）。
func levelsOf(m store.SchoolInfo) []string {
	var lv []string
	if m.Is985 {
		lv = append(lv, "985")
	}
	if m.Is211 {
		lv = append(lv, "211")
	}
	if m.Syl {
		lv = append(lv, "双一流")
	}
	return lv
}

// latestPlanYear 只保留计划里最新年份的行（院校专业组逐年变，组视图是单年视图）。
func latestPlanYear(rows []core.PlanRow) []core.PlanRow {
	maxY := 0
	for _, r := range rows {
		if r.Year > maxY {
			maxY = r.Year
		}
	}
	var out []core.PlanRow
	for _, r := range rows {
		if r.Year == maxY {
			out = append(out, r)
		}
	}
	return out
}

func planYear(rows []core.PlanRow) int {
	if len(rows) == 0 {
		return 0
	}
	return rows[0].Year
}
