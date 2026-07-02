package main

import (
	"database/sql"
	"testing"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/store"
	"github.com/sunfmin/zhiyuanwiki/internal/zj"
)

// memStore 起一个纯内存 staging 库（无磁盘落地），驱动 buildBundle / 投影核心单测。
// SetMaxOpenConns(1)：modernc :memory: 每连接各自一份内存库，连接池多连接会看不到彼此的写，
// 锁到单连接后读写同库。见 store.OpenDB（注入已打开 *sql.DB 的构造器）。
func memStore(t *testing.T) *store.DB {
	t.Helper()
	raw, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	raw.SetMaxOpenConns(1)
	db, err := store.OpenDB(raw)
	if err != nil {
		t.Fatalf("store.OpenDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// planMajorByName 在某校 major 视图里按 (专业名, 科类) 取一条（测试辅助）。
func planMajorByName(d schoolDetail, major, track string) (zj.PlanMajor, bool) {
	for _, m := range d.Plan2026 {
		if m.MajorName == major && m.Track == track {
			return m, true
		}
	}
	return zj.PlanMajor{}, false
}

// TestBuildBundleMajorPathInMemory 端到端跑此前零测试的 major 投影路径（buildBundle + projectPlanMajors），
// 全程喂内存 *store.DB：分数(双科类·多年)→计划(2026·同专业多行求和)→院校×专业报考视图，
// 断言 计划求和 / 往年位次按科类挂接取最近年 / 门类码 / 院校属性命中 / 排序。
func TestBuildBundleMajorPathInMemory(t *testing.T) {
	db := memStore(t)

	// 分数：物理·计算机 2024/2025（PrevRank 应取最近年 2025 的 3000，非 2024 的 3500）；历史·法学 2025。
	if err := db.ReplaceScores("cqtest", []core.MajorScoreRow{
		{Year: 2025, Track: "物理", SchoolCode: "5001", SchoolName: "测试大学", MajorName: "计算机科学与技术", MinScore: 640, MinRank: 3000},
		{Year: 2024, Track: "物理", SchoolCode: "5001", SchoolName: "测试大学", MajorName: "计算机科学与技术", MinScore: 630, MinRank: 3500},
		{Year: 2025, Track: "历史", SchoolCode: "5001", SchoolName: "测试大学", MajorName: "法学", MinScore: 600, MinRank: 8000},
	}); err != nil {
		t.Fatal(err)
	}
	// 计划 2026：计算机拆两行（同 渠道/专业/选科/科类）应求和 30+5=35；法学 20。
	if err := db.ReplacePlan("cqtest", []core.PlanRow{
		{Year: 2026, Track: "物理", SchoolCode: "5001", SchoolName: "测试大学", MajorName: "计算机科学与技术", SelKe: "首选物理", Plan: 30, Schooling: "四年", Tuition: "5000"},
		{Year: 2026, Track: "物理", SchoolCode: "5001", SchoolName: "测试大学", MajorName: "计算机科学与技术", SelKe: "首选物理", Plan: 5, Schooling: "四年", Tuition: "5000"},
		{Year: 2026, Track: "历史", SchoolCode: "5001", SchoolName: "测试大学", MajorName: "法学", SelKe: "首选历史", Plan: 20, Schooling: "四年", Tuition: "5000"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.ReplaceCatalog([]store.CatalogRow{
		{SchoolName: "测试大学", Major: "计算机科学与技术", Menlei: "工学"},
		{SchoolName: "测试大学", Major: "法学", Menlei: "法学"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.ReplaceSchools([]store.SchoolInfo{
		{Name: "测试大学", Province: "重庆", City: "重庆", Kind: "理工类", Is985: true, Is211: true, Syl: true},
	}); err != nil {
		t.Fatal(err)
	}

	p := province{slug: "cqtest", name: "测试重庆", tracks: []string{"物理", "历史"}, model: "major"}
	b := buildBundle(db, p, projectPlanMajors)

	if len(b.schools) != 1 || b.schools[0].Name != "测试大学" {
		t.Fatalf("院校聚合错误: %+v", b.schools)
	}
	key := b.schools[0].Key
	d := b.details[key]
	if len(d.Plan2026) != 2 {
		t.Fatalf("应有 2 条院校×专业（计算机·物理 + 法学·历史），got %d", len(d.Plan2026))
	}

	cs, ok := planMajorByName(d, "计算机科学与技术", "物理")
	if !ok {
		t.Fatalf("缺 计算机·物理 视图: %+v", d.Plan2026)
	}
	if cs.Plan != 35 {
		t.Errorf("计算机 计划应求和 35（30+5），got %d", cs.Plan)
	}
	if cs.PrevYear != 2025 || cs.PrevRank != 3000 {
		t.Errorf("计算机 往年应取最近年 2025·位次 3000，got year=%d rank=%d", cs.PrevYear, cs.PrevRank)
	}
	if cs.Menlei != "工" {
		t.Errorf("计算机 门类码应为 工，got %q", cs.Menlei)
	}

	law, ok := planMajorByName(d, "法学", "历史")
	if !ok {
		t.Fatalf("缺 法学·历史 视图: %+v", d.Plan2026)
	}
	if law.PrevRank != 8000 || law.Plan != 20 {
		t.Errorf("法学 应 位次 8000·计划 20，got rank=%d plan=%d", law.PrevRank, law.Plan)
	}

	// 院校属性按校名命中：985/211/双一流 + 城市层级。
	if lv, ok := b.levels[key]; !ok || lv != [3]bool{true, true, true} {
		t.Errorf("院校层次应 985/211/双一流 全真，got %v ok=%v", lv, ok)
	}
	if m, ok := b.meta[key]; !ok || m.Province != "重庆" || m.CityTier == "" {
		t.Errorf("院校 meta 缺失或城市层级空: %+v ok=%v", m, ok)
	}
}

// TestBuildPlanMajorsTracked 直接测投影核心（纯函数）：预置分数/计划切片经 core 归并出 resolver+leaves，
// 再喂 buildPlanMajorsTracked，断言 双科类同专业各成一条 + 排序（有位次按 EquivRank 升序、难者在前）。
func TestBuildPlanMajorsTracked(t *testing.T) {
	scores := []core.MajorScoreRow{
		{Year: 2025, Track: "物理", SchoolCode: "5001", SchoolName: "测试大学", MajorName: "计算机科学与技术", MinScore: 640, MinRank: 3000},
		{Year: 2025, Track: "物理", SchoolCode: "5001", SchoolName: "测试大学", MajorName: "土木工程", MinScore: 600, MinRank: 9000},
	}
	plan := []core.PlanRow{
		{Year: 2026, Track: "物理", SchoolCode: "5001", SchoolName: "测试大学", MajorName: "土木工程", SelKe: "首选物理", Plan: 40},
		{Year: 2026, Track: "物理", SchoolCode: "5001", SchoolName: "测试大学", MajorName: "计算机科学与技术", SelKe: "首选物理", Plan: 30},
	}
	resolver := core.BuildSchoolResolver(append(core.IdentRowsFromScores(scores), core.IdentRowsFromPlan(plan)...))
	_, leaves := core.AggregateLeavesR(scores, resolver)

	out := buildPlanMajorsTracked(plan, leaves, resolver, map[core.YearTrack]int{}, 2026, nil)
	if len(out) != 1 {
		t.Fatalf("应只 1 个院校实体，got %d", len(out))
	}
	var list []zj.PlanMajor
	for _, v := range out {
		list = v
	}
	if len(list) != 2 {
		t.Fatalf("应有 2 条专业，got %d", len(list))
	}
	// 排序：有位次者按 EquivRank 升序（位次小=更难在前）。计算机(3000) 应排在 土木(9000) 前。
	if list[0].MajorName != "计算机科学与技术" || list[1].MajorName != "土木工程" {
		t.Errorf("应按位次升序排：计算机 在 土木 前，got %q, %q", list[0].MajorName, list[1].MajorName)
	}
	if list[0].EquivRank != 3000 {
		t.Errorf("空 totals 时 EquivRank 应等于原位次 3000，got %d", list[0].EquivRank)
	}
	// menlei 传 nil 时不设门类码。
	if list[0].Menlei != "" {
		t.Errorf("menlei=nil 时门类码应空，got %q", list[0].Menlei)
	}
}
