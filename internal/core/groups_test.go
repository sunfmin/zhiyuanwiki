package core

import "testing"

// 同校同组代码在两科类间复用时，必须拆成两个组（按科类一等），不能并进同一个组。
// 这是四川/安徽等省的真实形态，也修复了江苏/湖南/黑龙江的潜在串档。
func TestBuildGroups2026SplitsByTrack(t *testing.T) {
	plan := []PlanRow{
		{Year: 2025, Track: "物理", SchoolCode: "3415", SchoolName: "某大学", GroupCode: "101", GroupName: "101组", MajorName: "会计学", Plan: 5},
		{Year: 2025, Track: "历史", SchoolCode: "3415", SchoolName: "某大学", GroupCode: "101", GroupName: "101组", MajorName: "法学", Plan: 4},
	}
	ent := NormName("某大学")
	leaves := []MajorLeaf{
		// 会计学同时有物理(位次1000)、历史(位次2000)两年——挂接应取各自科类。
		{SchoolKey: ent, SchoolCode: "3415", MajorKey: MajorKey("会计学"), MajorName: "会计学", Years: []YearScore{
			{Year: 2024, Track: "物理", MinScore: 600, MinRank: 1000},
			{Year: 2024, Track: "历史", MinScore: 590, MinRank: 2000},
		}},
		{SchoolKey: ent, SchoolCode: "3415", MajorKey: MajorKey("法学"), MajorName: "法学", Years: []YearScore{
			{Year: 2024, Track: "历史", MinScore: 580, MinRank: 2500},
		}},
	}
	got := BuildGroups2026(plan, leaves, nil, nil)
	groups := got[ent]
	if len(groups) != 2 {
		t.Fatalf("同校同号跨科类应拆成 2 个组，got %d: %+v", len(groups), groups)
	}
	byTrack := map[string]Group2026{}
	for _, g := range groups {
		byTrack[g.Track] = g
	}
	wuli, ok := byTrack["物理"]
	if !ok || len(wuli.Majors) != 1 || wuli.Majors[0].MajorName != "会计学" {
		t.Fatalf("物理组应只含会计学: %+v", wuli)
	}
	// 会计学的物理组应挂物理位次(1000)，而非历史(2000)——track-aware 挂接。
	if wuli.Majors[0].PrevRank != 1000 {
		t.Errorf("物理组会计学 PrevRank=%d, want 1000（不应串到历史位次）", wuli.Majors[0].PrevRank)
	}
	lishi, ok := byTrack["历史"]
	if !ok || len(lishi.Majors) != 1 || lishi.Majors[0].MajorName != "法学" {
		t.Fatalf("历史组应只含法学: %+v", lishi)
	}
}
