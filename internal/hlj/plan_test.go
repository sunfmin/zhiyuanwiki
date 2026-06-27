package hlj

import (
	"testing"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

func TestEquivRank(t *testing.T) {
	totals := map[core.YearTrack]int{
		{Year: 2025, Track: "物理"}: 100000,
		{Year: 2026, Track: "物理"}: 110000,
	}
	yt := func(y int, tr string) core.YearTrack { return core.YearTrack{Year: y, Track: tr} }
	tests := []struct {
		name     string
		rank     int
		from, to core.YearTrack
		want     int
	}{
		{"按总人数比例放大", 100, yt(2025, "物理"), yt(2026, "物理"), 110},
		{"同年同科类-原样", 100, yt(2026, "物理"), yt(2026, "物理"), 100},
		{"缺from总人数-回退原位次", 100, yt(2024, "物理"), yt(2026, "物理"), 100},
		{"缺to总人数-回退原位次", 100, yt(2025, "物理"), yt(2026, "历史"), 100},
		{"非正位次-原样", 0, yt(2025, "物理"), yt(2026, "物理"), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := core.EquivRank(tt.rank, tt.from, tt.to, totals); got != tt.want {
				t.Errorf("EquivRank = %d，想要 %d", got, tt.want)
			}
		})
	}
}

func TestSelKeAllows(t *testing.T) {
	wuhuasheng := map[string]bool{"物理": true, "化学": true, "生物": true}
	tests := []struct {
		req  string
		want bool
	}{
		{"不限", true},
		{"化学", true},
		{"生物", true},
		{"化学和生物", true},
		{"化学或生物", true},
		{"政治", false},
		{"地理", false},
		{"", true},
		{"政治或地理", false},
	}
	for _, tt := range tests {
		t.Run(tt.req, func(t *testing.T) {
			if got := SelKeAllows(tt.req, wuhuasheng); got != tt.want {
				t.Errorf("SelKeAllows(%q) = %v，想要 %v", tt.req, got, tt.want)
			}
		})
	}
}

func TestParsePlanRows(t *testing.T) {
	rows := [][]string{
		{"黑龙江省2026年招生计划", ""},
		{"年份", "生源地", "科类", "批次", "计划类别", "院校代码", "院校名称", "院校专业组代码", "专业组代码", "专业组名称", "专业代码", "专业全称", "专业名称", "专业备注", "选科要求", "计划人数", "学制", "学费"},
		{"2026", "黑龙江", "物理", "本科批", "普通", "1003", "清华大学", "1003009", "009", "第009组", "01", "计算机类", "计算机类", "", "化学", "5", "4", "5000"},
		{"2026", "黑龙江", "物理", "本科批", "普通", "1003", "清华大学", "1003009", "009", "第009组", "02", "法学", "法学", "", "化学", "3", "4", "5000"},
		// 高职专科批应排除
		{"2026", "黑龙江", "物理", "高职专科批", "普通", "4245", "某专科", "4245001", "001", "第001组", "01", "x", "工业机器人技术", "", "不限", "10", "3", "6000"},
	}
	s, err := core.NewSheet(rows, planHeader)
	if err != nil {
		t.Fatal(err)
	}
	got := parsePlanSheet(s)
	if len(got) != 2 {
		t.Fatalf("解析到 %d 行，想要 2（本科批）", len(got))
	}
	if got[0].GroupCode != "009" || got[0].Plan != 5 || got[0].SelKe != "化学" {
		t.Errorf("第一行 = %+v", got[0])
	}
}

func TestBuildGroups2026(t *testing.T) {
	plan := []PlanRow{
		{Year: 2026, Track: "物理", SchoolCode: "1003", SchoolName: "清华大学", GroupCode: "009", GroupName: "第009组", MajorName: "计算机类", SelKe: "化学", Plan: 5},
		{Year: 2026, Track: "物理", SchoolCode: "1003", SchoolName: "清华大学", GroupCode: "009", GroupName: "第009组", MajorName: "法学", SelKe: "化学", Plan: 3},
	}
	// 注意：往年叶子的组号是 012（2023），与 2026 的 009 不同——必须按院校+专业名挂接，不按组号。
	leaves := []core.MajorLeaf{
		{SchoolCode: "1003", MajorKey: core.MajorKey("计算机类"), MajorName: "计算机类",
			Years: []core.YearScore{{Year: 2025, Track: "物理", MinScore: 690, MinRank: 120}}},
	}
	totals := map[core.YearTrack]int{}
	groups := BuildGroups2026(plan, leaves, totals, nil)

	gs := groups["1003"]
	if len(gs) != 1 || gs[0].GroupCode != "009" {
		t.Fatalf("groups[1003] = %+v", gs)
	}
	if len(gs[0].Majors) != 2 {
		t.Fatalf("组内专业 = %d，想要 2", len(gs[0].Majors))
	}
	// 计算机类应挂上 2025 位次 120（按专业名，不管组号 012≠009）
	var cs *GroupMajor
	for i := range gs[0].Majors {
		if gs[0].Majors[i].MajorName == "计算机类" {
			cs = &gs[0].Majors[i]
		}
	}
	if cs == nil || cs.PrevRank != 120 || cs.PrevYear != 2025 {
		t.Errorf("计算机类挂接 = %+v", cs)
	}
	// 法学没有往年叶子，PrevRank 应为 0（未挂接）
	for _, m := range gs[0].Majors {
		if m.MajorName == "法学" && m.PrevRank != 0 {
			t.Errorf("法学不应挂接到位次，得 %d", m.PrevRank)
		}
	}
}
