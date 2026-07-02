package store

import (
	"path/filepath"
	"testing"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

func openTmp(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestScoresRoundTripAndIdempotent(t *testing.T) {
	db := openTmp(t)
	rows := []core.MajorScoreRow{
		{Year: 2025, Track: "物理", SchoolCode: "1101", SchoolName: "南京大学", GroupCode: "（05）", MajorName: "人工智能", MinScore: 680, MinRank: 196},
		{Year: 2025, Track: "历史", SchoolCode: "1101", SchoolName: "南京大学", MajorName: "汉语言文学", MinScore: 640, MinRank: 300},
	}
	if err := db.ReplaceScores("js", rows); err != nil {
		t.Fatal(err)
	}
	// 重导（幂等）：再写一次不应翻倍。
	if err := db.ReplaceScores("js", rows); err != nil {
		t.Fatal(err)
	}
	got, err := db.LoadScores("js")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("幂等重导后应 2 行，got %d", len(got))
	}
}

func TestTotalsFromYiFenYiDuan(t *testing.T) {
	db := openTmp(t)
	yds := []*core.YiFenYiDuan{
		{Province: "江苏", Track: "物理", Year: 2025, Entries: []core.FenduanEntry{
			{Score: 683, Cumulative: 126}, {Score: 682, Cumulative: 144}, {Score: 400, Cumulative: 205975},
		}},
	}
	if err := db.ReplaceYiFenYiDuan("js", yds); err != nil {
		t.Fatal(err)
	}
	totals, err := db.LoadTotals("js")
	if err != nil {
		t.Fatal(err)
	}
	if got := totals[core.YearTrack{Year: 2025, Track: "物理"}]; got != 205975 {
		t.Errorf("总人数应取最大累计 205975，got %d", got)
	}
}

func TestPlanTotalsLatestExcludesXinjiangDanlie(t *testing.T) {
	db := openTmp(t)
	// 新疆普通类与「单列类」（民语言/民考民）计划混在一张表里。单列类考生走独立考试、独立一分一段，
	// 不在普通类高考人数分母里，故本科计划总数须剔除单列类，否则本科计划＞考生会显成 >100%。
	if err := db.ReplacePlan("xj", []core.PlanRow{
		{Year: 2025, Track: "理科", SchoolCode: "1001", SchoolName: "北京大学", MajorName: "法学", Plan: 3},
		{Year: 2025, Track: "理科", SchoolCode: "1001", SchoolName: "北京大学", MajorName: "法学", Remark: "（民族班）（单列类）", Plan: 5}, // 单列类：剔除
		{Year: 2025, Track: "文科", SchoolCode: "1001", SchoolName: "北京大学", MajorName: "哲学", Remark: "（南疆单列计划）", Plan: 2}, // 普通类南疆单列：保留
		{Year: 2024, Track: "理科", SchoolCode: "1001", SchoolName: "北京大学", MajorName: "法学", Plan: 9},                       // 旧年：不应被选中
	}); err != nil {
		t.Fatal(err)
	}
	// 别省无单列类标记，照常全计——验证全表过滤不误伤其他省。
	if err := db.ReplacePlan("js", []core.PlanRow{
		{Year: 2025, Track: "物理", SchoolCode: "1101", SchoolName: "南京大学", MajorName: "软件工程", Plan: 7},
	}); err != nil {
		t.Fatal(err)
	}
	totals, err := db.PlanTotalsLatest()
	if err != nil {
		t.Fatal(err)
	}
	// 新疆取最新年 2025：普通 3 + 普通类南疆单列 2 = 5（剔除单列类 5、不取旧年 2024 的 9）。
	if got := totals["xj"]; got.Year != 2025 || got.Total != 5 {
		t.Errorf("新疆应取 2025 普通类计划 5（剔除单列类），got %+v", got)
	}
	// 别省不受单列类过滤影响。
	if got := totals["js"]; got.Total != 7 {
		t.Errorf("江苏计划应 7（无单列类），got %+v", got)
	}
}

func TestSchoolIndexNameAndBaseLookup(t *testing.T) {
	db := openTmp(t)
	if err := db.ReplaceSchools([]SchoolInfo{
		{Name: "南京大学", Province: "江苏", City: "南京", Kind: "综合类", Is985: true, Is211: true, Syl: true, Rank: 6},
	}); err != nil {
		t.Fatal(err)
	}
	idx, err := db.SchoolIndex()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := idx.Lookup("南京大学"); !ok {
		t.Error("精确名应命中")
	}
	// 分校去括号退到母体基名。
	if info, ok := idx.Lookup("南京大学（苏州校区）"); !ok || !info.Is985 {
		t.Errorf("基名应继承母体 985，got %+v ok=%v", info, ok)
	}
}

func TestHomeSchoolCounts(t *testing.T) {
	db := openTmp(t)
	if err := db.ReplaceSchools([]SchoolInfo{
		{Name: "南京大学", Province: "江苏"},
		{Name: "东南大学", Province: "江苏"},
		{Name: "浙江大学", Province: "浙江"},
		{Name: "无省份校", Province: ""}, // 空省份不计入
	}); err != nil {
		t.Fatal(err)
	}
	counts, err := db.HomeSchoolCounts()
	if err != nil {
		t.Fatal(err)
	}
	if counts["江苏"] != 2 {
		t.Errorf("江苏 本省院校应 2，got %d", counts["江苏"])
	}
	if counts["浙江"] != 1 {
		t.Errorf("浙江 本省院校应 1，got %d", counts["浙江"])
	}
	if _, ok := counts[""]; ok {
		t.Error("空省份不应入表")
	}
}

func TestMenleiFromCatalog(t *testing.T) {
	db := openTmp(t)
	if err := db.ReplaceCatalog([]CatalogRow{
		{SchoolName: "南京大学", Major: "软件工程", Menlei: "工学"},
	}); err != nil {
		t.Fatal(err)
	}
	mc, err := db.Menlei()
	if err != nil {
		t.Fatal(err)
	}
	if got := mc.Code("软件工程"); got != "工" {
		t.Errorf("软件工程 门类码应为 工，got %q", got)
	}
}
