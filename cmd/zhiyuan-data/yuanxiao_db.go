package main

import (
	"fmt"
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/store"
)

// projection 是「计划/叶子 → 每校报考视图」的投影结果：assign 把视图挂进某校 detail 并返回其条目数，
// label 是汇总行里的口径名。group 省投院校专业组（Groups2026）、major 省投院校×专业（Plan2026），
// 由 ADR-0017 model 分派在缝处（yuanxiaoCmd）选出对应 projectFn。
type projection struct {
	label  string
	assign func(d *schoolDetail, key string) int
}

// projectFn 消费已归并的计划/叶子/归并器/往年总人数/分数/门类分类器，产出 projection。
// 两种实现（projectGroups / projectPlanMajors）即原 buildDBBundle / buildMajorBundle 的「投影核心」，
// 作为参数传入共享骨架 buildBundle——骨架同形、核心发散（#32 已确认不合并核心）。
type projectFn func(plan []core.PlanRow, leaves []core.MajorLeaf, r *core.SchoolResolver, totals map[core.YearTrack]int, scores []core.MajorScoreRow, menlei func(string) string) projection

// buildBundle 是 group / major 两省族共用的单一投影骨架（构建期 staging 管线，见 ADR-0014）：
// LoadScores→空守卫→SchoolIndex→Menlei→LoadTotals→LoadPlan→计划年补齐→BuildSchoolResolver→
// AggregateLeavesR→projectFn→逐校 assemble（叶子 + 报考视图）+ meta/levels/CityTier→汇总。
// DB 句柄注入（不在函数内 Open），故可喂内存 *store.DB 或预置切片单测。投影核心由 project 参数决定。
func buildBundle(db *store.DB, p province, project projectFn) schoolBundle {
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
	// 个别省计划表无「年份」列（如内蒙），plan 行 Year=0——用录取分数最新年补齐，否则等效位次目标年为 0、
	// 报考视图「计划年」显示 0 且 EquivRank 退化。计划本就对应该 score 年，补齐是安全归一。
	if planYear(plan) == 0 {
		if sy := latestScoreYear(scores); sy > 0 {
			for i := range plan {
				plan[i].Year = sy
			}
		}
	}

	// 院校身份归并器覆盖 分数(全年) ∪ 计划(全年)：院校全集含只在计划里出现的新招生校（如广州大学），
	// 并按归一化校名归并、按渠道拆分（ADR-0021）。schools=并集，leaves/视图 均按实体键挂接。
	resolver := core.BuildSchoolResolver(append(core.IdentRowsFromScores(scores), core.IdentRowsFromPlan(planAll)...))
	schools, leaves := core.AggregateLeavesR(scores, resolver)
	if rn := resolver.Renames(); len(rn) > 0 {
		fmt.Printf("  改名/转设归并 %d 处（人工可复核）：\n", len(rn))
		for _, s := range rn {
			fmt.Printf("    · %s\n", s)
		}
	}
	proj := project(plan, leaves, resolver, totals, scores, menlei.Code)

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
	viewCount, matched := 0, 0
	for _, s := range schools {
		d := schoolDetail{School: s, Leaves: nonNilLeaves(byCode[schoolKey(s)])}
		viewCount += proj.assign(&d, schoolKey(s))
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
	fmt.Printf("  报考视图：%s %d 个（计划年 %d）· 院校属性命中 %d/%d\n",
		proj.label, viewCount, planYear(plan), matched, len(schools))
	return b
}

// projectGroups 是院校专业组投影（原 buildDBBundle 核心）：招生计划(最新年)→院校专业组报考视图。
// 黑龙江/江苏/湖南/四川/安徽… group 模型省共用（见 ADR-0014）。core.BuildGroups2026R 保持不变。
func projectGroups(plan []core.PlanRow, leaves []core.MajorLeaf, r *core.SchoolResolver, totals map[core.YearTrack]int, scores []core.MajorScoreRow, menlei func(string) string) projection {
	groups2026 := core.BuildGroups2026R(plan, leaves, r, totals, menlei)
	return projection{
		label: "院校专业组",
		assign: func(d *schoolDetail, key string) int {
			d.Groups2026 = groups2026[key]
			return len(d.Groups2026)
		},
	}
}

// nonNilLeaves 保证叶子数组非 nil（计划独有校如广州大学无历史录取 → nil，会序列化成 JSON null，
// 而前端 school.leaves.slice() 会在 null 上崩）。空叶子序列化成 []。
func nonNilLeaves(lvs []core.MajorLeaf) []core.MajorLeaf {
	if lvs == nil {
		return []core.MajorLeaf{}
	}
	return lvs
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

// latestScoreYear 返回录取分数行里的最大年份（用于给无「年份」列的计划表补齐年份）。
func latestScoreYear(rows []core.MajorScoreRow) int {
	maxY := 0
	for _, r := range rows {
		if r.Year > maxY {
			maxY = r.Year
		}
	}
	return maxY
}
