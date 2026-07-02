package main

import (
	"fmt"
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/store"
	"github.com/sunfmin/zhiyuanwiki/internal/zj"
)

// buildDBBundle 从 SQLite staging 投影某省院校数据（构建期 staging 管线省份共用，见 ADR-0014）：
// 专业录取分数→院校×专业叶子、招生计划(最新年)→院校专业组报考视图、全国院校属性(按校名)→过滤属性、
// 全国专业门类→门类码。与 buildHLJBundle 同形，只是数据来源从 xlsx 换成 DB（江苏/湖南/四川/安徽…通用）。
func buildDBBundle(dbPath string, p province) schoolBundle {
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
	fmt.Printf("  专业录取分数：%d 行（%s·本科·含位次）\n", len(scores), strings.Join(p.tracks, "/"))

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
	// 并按归一化校名归并、按渠道拆分（ADR-0021）。schools=并集，leaves/groups 均按实体键挂接。
	resolver := core.BuildSchoolResolver(append(core.IdentRowsFromScores(scores), core.IdentRowsFromPlan(planAll)...))
	schools, leaves := core.AggregateLeavesR(scores, resolver)
	groups2026 := core.BuildGroups2026R(plan, leaves, resolver, totals, menlei.Code)
	if rn := resolver.Renames(); len(rn) > 0 {
		fmt.Printf("  改名/转设归并 %d 处（人工可复核）：\n", len(rn))
		for _, s := range rn {
			fmt.Printf("    · %s\n", s)
		}
	}

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
	groupCount, matched := 0, 0
	for _, s := range schools {
		d := schoolDetail{School: s, Leaves: nonNilLeaves(byCode[schoolKey(s)]), Groups2026: groups2026[schoolKey(s)]}
		groupCount += len(d.Groups2026)
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
	fmt.Printf("  报考视图：院校专业组 %d 个（计划年 %d）· 院校属性命中 %d/%d\n",
		groupCount, planYear(plan), matched, len(schools))
	return b
}

// buildDBBundleMajor 从 SQLite staging 投影 major 模型省份（浙江：综合·专业平行志愿，无组）的院校
// 数据，与 buildDBBundle 同形，只是报考视图走 zj.BuildPlan2026 产 plan2026（院校×专业）而非
// groups2026。计划行经 zj.FromCorePlan 从统一 plan 表还原成 PlanRow2026（招生类型从 Batch 列取回）。
// 见 ADR-0014 / issue #20。院校属性此处仍按全国 school 表挂接；浙江特有 city_tier/按码属性的保全在 #21。
func buildDBBundleMajor(dbPath string, p province) schoolBundle {
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

	menlei, err := db.Menlei()
	if err != nil {
		fatal(err)
	}
	// 院校属性走省专属按码表（浙江「一表联动」：含 city_tier、按院校代码挂接），不用全国 school 表
	// （按校名、无 city_tier，且 城市/类型 命名与浙江源大相径庭，会整体回归）。见 #21。
	attrs, err := db.SchoolAttrs(p.slug)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("  院校属性（按代码）%d 所 · 专业→门类 %d 条\n", len(attrs), menlei.Len())

	totals, err := db.LoadTotals(p.slug)
	if err != nil {
		fatal(err)
	}
	planAll, err := db.LoadPlan(p.slug)
	if err != nil {
		fatal(err)
	}

	// 身份归并覆盖 分数∪计划（同 buildDBBundle）：院校全集含计划独有校、按校名归并（ADR-0021）。
	resolver := core.BuildSchoolResolver(append(core.IdentRowsFromScores(scores), core.IdentRowsFromPlan(planAll)...))
	schools, leaves := core.AggregateLeavesR(scores, resolver)
	if rn := resolver.Renames(); len(rn) > 0 {
		fmt.Printf("  改名/转设归并 %d 处（人工可复核）：\n", len(rn))
		for _, s := range rn {
			fmt.Printf("    · %s\n", s)
		}
	}
	// 浙江报考视图（院校×专业）仍按代号建，然后按 code→实体键 归拢到院校页（同名多渠道并入一页）。
	planByCode := zj.BuildPlan2026(zj.FromCorePlan(planAll), leaves, totals, zjRefYear, menlei)
	codeToEntity := map[string]string{}
	for _, r := range planAll {
		codeToEntity[r.SchoolCode] = resolver.Entity(r.SchoolName)
	}
	planByEntity := map[string][]zj.PlanMajor{}
	for code, majors := range planByCode {
		ent := codeToEntity[code]
		if ent == "" {
			ent = resolver.Entity(code) // 兜底
		}
		planByEntity[ent] = append(planByEntity[ent], majors...)
	}

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
		if a, ok := attrs[s.Code]; ok {
			matched++
			b.levels[schoolKey(s)] = [3]bool{a.Is985, a.Is211, a.Syl}
			b.meta[schoolKey(s)] = schoolMetaOut{
				Province: a.Province, City: a.City, CityTier: a.CityTier,
				Owner: a.Ownership, Kind: a.Kind, Levels: a.Levels(),
			}
		}
	}
	fmt.Printf("  报考视图：院校×专业 %d 个 · 院校属性命中 %d/%d\n", planCount, matched, len(schools))
	return b
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
