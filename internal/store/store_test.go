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
