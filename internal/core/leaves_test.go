package core

import "testing"

// findLeaf 按归一专业名取叶子（测试辅助）。
func findLeaf(leaves []MajorLeaf, code, major string) *MajorLeaf {
	for i := range leaves {
		if leaves[i].SchoolCode == code && leaves[i].MajorName == major {
			return &leaves[i]
		}
	}
	return nil
}

// TestLeafKey：叶子复合键格式（schoolKey/channel/majorKey），且 MajorLeaf.Key() 与之一致。
// 挂接往年位次两侧（建索引 / 查表）都靠此键同形，格式一改必须只改这一处。
func TestLeafKey(t *testing.T) {
	cases := []struct {
		name                         string
		schoolKey, channel, majorKey string
		want                         string
	}{
		{"常规", "北京大学", "1101", "0a1b2c3d", "北京大学/1101/0a1b2c3d"},
		{"空字段", "", "", "", "//"},
		{"含空渠道", "清华大学", "", "deadbeef", "清华大学//deadbeef"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := LeafKey(c.schoolKey, c.channel, c.majorKey); got != c.want {
				t.Errorf("LeafKey(%q,%q,%q) = %q, want %q", c.schoolKey, c.channel, c.majorKey, got, c.want)
			}
			// MajorLeaf.Key() 必须与 LeafKey 逐字节一致（同为唯一格式定义）。
			l := MajorLeaf{SchoolKey: c.schoolKey, SchoolCode: c.channel, MajorKey: c.majorKey}
			if got := l.Key(); got != c.want {
				t.Errorf("MajorLeaf.Key() = %q, want %q", got, c.want)
			}
		})
	}
}

// TestAggregateLeavesDedupMinRank：同一 (院校,专业,年,科类) 多行取最低位次（最难那条）。
func TestAggregateLeavesDedupMinRank(t *testing.T) {
	_, leaves := AggregateLeaves([]MajorScoreRow{
		{Year: 2025, Track: "物理", SchoolCode: "1101", SchoolName: "测试大学", MajorName: "计算机", MinScore: 630, MinRank: 5000},
		{Year: 2025, Track: "物理", SchoolCode: "1101", SchoolName: "测试大学", MajorName: "计算机", MinScore: 640, MinRank: 3000},
	})
	lf := findLeaf(leaves, "1101", "计算机")
	if lf == nil {
		t.Fatal("缺计算机叶子")
	}
	if len(lf.Years) != 1 {
		t.Fatalf("同年同科类应合并为 1 个数据点，got %d: %+v", len(lf.Years), lf.Years)
	}
	if lf.Years[0].MinRank != 3000 {
		t.Errorf("应取最低位次 3000（最难），got %d", lf.Years[0].MinRank)
	}
}

// TestAggregateLeavesMultiTrackAndSort：同一专业在物理/历史各成一个数据点，Years 按 (年,科类) 升序。
func TestAggregateLeavesMultiTrackAndSort(t *testing.T) {
	_, leaves := AggregateLeaves([]MajorScoreRow{
		{Year: 2025, Track: "历史", SchoolCode: "1101", SchoolName: "测试大学", MajorName: "经济学", MinRank: 900},
		{Year: 2024, Track: "物理", SchoolCode: "1101", SchoolName: "测试大学", MajorName: "经济学", MinRank: 1200},
		{Year: 2025, Track: "物理", SchoolCode: "1101", SchoolName: "测试大学", MajorName: "经济学", MinRank: 1100},
	})
	lf := findLeaf(leaves, "1101", "经济学")
	if lf == nil {
		t.Fatal("缺经济学叶子")
	}
	if len(lf.Years) != 3 {
		t.Fatalf("物理(2024/2025)+历史(2025) 应为 3 个数据点，got %d", len(lf.Years))
	}
	// 升序：2024物理 → 2025历史 → 2025物理（同年按科类「历史」<「物理」）
	want := []struct {
		year  int
		track string
	}{{2024, "物理"}, {2025, "历史"}, {2025, "物理"}}
	for i, w := range want {
		if lf.Years[i].Year != w.year || lf.Years[i].Track != w.track {
			t.Errorf("Years[%d]=%d/%s, want %d/%s", i, lf.Years[i].Year, lf.Years[i].Track, w.year, w.track)
		}
	}
}

// TestAggregateLeavesLatestYearWins：校名与选科取最新年份——且与行输入顺序无关。
func TestAggregateLeavesLatestYearWins(t *testing.T) {
	// 故意把旧年份行排在新年份行之后，验证「取最新」不依赖输入顺序。
	schools, leaves := AggregateLeaves([]MajorScoreRow{
		{Year: 2025, Track: "物理", SchoolCode: "1101", SchoolName: "测试大学(新)", MajorName: "计算机", SelKe: "首选物理+化学", MinRank: 3000},
		{Year: 2024, Track: "物理", SchoolCode: "1101", SchoolName: "测试大学(旧)", MajorName: "计算机", SelKe: "首选物理", MinRank: 3500},
	})
	if len(schools) != 1 || schools[0].Name != "测试大学(新)" {
		t.Fatalf("校名应取最新年份，got %+v", schools)
	}
	lf := findLeaf(leaves, "1101", "计算机")
	if lf == nil || lf.SelKe != "首选物理+化学" {
		t.Errorf("选科应取最新年份，got %+v", lf)
	}
}

// TestAggregateLeavesSkipsBlankCode：无院校代码的行被跳过；院校与叶子按 院校实体键(归一化校名) 升序
// （ADR-0021 起主键是校名而非代号，故「乙大学」<「甲大学」按字符序排前）。
func TestAggregateLeavesSkipsBlankCode(t *testing.T) {
	schools, leaves := AggregateLeaves([]MajorScoreRow{
		{Year: 2025, Track: "物理", SchoolCode: "", SchoolName: "无代码校", MajorName: "X", MinRank: 1},
		{Year: 2025, Track: "物理", SchoolCode: "1102", SchoolName: "乙大学", MajorName: "数学", MinRank: 2000},
		{Year: 2025, Track: "物理", SchoolCode: "1101", SchoolName: "甲大学", MajorName: "物理学", MinRank: 1000},
	})
	if len(schools) != 2 {
		t.Fatalf("空院校代码行应跳过，got %d 校", len(schools))
	}
	if schools[0].Key != NormName("乙大学") || schools[1].Key != NormName("甲大学") {
		t.Errorf("院校应按实体键(校名)升序: %+v", schools)
	}
	if len(leaves) != 2 || leaves[0].SchoolKey != NormName("乙大学") || leaves[1].SchoolKey != NormName("甲大学") {
		t.Errorf("叶子应按 (院校实体键,专业名) 升序: %+v", leaves)
	}
}
