package zj

import (
	"testing"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

func TestParsePlan2026Rows(t *testing.T) {
	rows := [][]string{
		{"年份", "院校名称", "院校代码", "科类", "批次", "招生类型", "专业名称", "专业代码", "所属专业组", "专业备注", "选科要求", "招生人数", "学制(年)", "学费(元)"},
		{"2026", "浙江大学", "0001", "综合", "普通类平行录取", "普通类", "计算机科学与技术", "01", "", "", "物理、化学(2科必选)", "30", "四年", "6000"},
		{"2026", "浙江大学", "0001", "综合", "普通类提前批", "三位一体", "医学试验班", "02", "", "", "物理、化学(2科必选)", "20", "五年", "6000"},
		// 艺术类应被科类过滤
		{"2026", "某美院", "0003", "艺术类", "艺术类本科批", "艺术类", "美术", "03", "", "", "不限", "5", "四年", "10000"},
	}
	got, err := parsePlan2026Rows(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("解析到 %d 行，想要 2（综合·收录批次）", len(got))
	}
	if got[0].MajorName != "计算机科学与技术" || got[0].Plan != 30 || got[0].Tuition != "6000" {
		t.Errorf("第一行 = %+v", got[0])
	}
}

func TestBuildPlan2026(t *testing.T) {
	plan := []PlanRow2026{
		{Year: 2026, SchoolCode: "0001", SchoolName: "浙江大学", AdmitType: "普通类", MajorName: "计算机科学与技术", SelKe: "物理、化学(2科必选)", Plan: 30, Tuition: "6000"},
		{Year: 2026, SchoolCode: "0001", SchoolName: "浙江大学", AdmitType: "普通类", MajorName: "汉语言文学", SelKe: "不限", Plan: 10},
		// 同 (专业,选科) 重复行：计划相加、不重复出现
		{Year: 2026, SchoolCode: "0001", SchoolName: "浙江大学", AdmitType: "普通类", MajorName: "计算机科学与技术", SelKe: "物理、化学(2科必选)", Plan: 5, Tuition: "6000"},
	}
	leaves := []core.MajorLeaf{
		{SchoolCode: "0001", MajorKey: core.MajorKey("计算机科学与技术"), MajorName: "计算机科学与技术",
			Years: []core.YearScore{
				{Year: 2024, Track: "综合", MinScore: 655, MinRank: 1500},
				{Year: 2025, Track: "综合", MinScore: 660, MinRank: 1200},
			}},
	}
	totals := map[core.YearTrack]int{} // 空 → 等效位次回退原位次
	got := BuildPlan2026(plan, leaves, totals, 2025, nil)

	list := got["0001"]
	if len(list) != 2 {
		t.Fatalf("0001 专业数 = %d，想要 2（计算机+汉语言，重复合并）", len(list))
	}
	// 计算机有位次应排在前
	cs := list[0]
	if cs.MajorName != "计算机科学与技术" {
		t.Fatalf("首位应为有位次的计算机，得 %+v", cs)
	}
	if cs.Plan != 35 {
		t.Errorf("计算机计划应合并为 35，得 %d", cs.Plan)
	}
	if cs.PrevYear != 2025 || cs.PrevRank != 1200 || cs.EquivRank != 1200 {
		t.Errorf("计算机挂接 = {%d,%d,%d}，想要 {2025,1200,1200}", cs.PrevYear, cs.PrevRank, cs.EquivRank)
	}
	// 汉语言无往年叶子，PrevRank 应为 0，排在后
	if list[1].MajorName != "汉语言文学" || list[1].PrevRank != 0 {
		t.Errorf("汉语言 = %+v", list[1])
	}
}
