package main

import (
	"testing"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

// TestDedupGroupMajors 覆盖定位视图的组内去重：同 MajorKey 合并为一条、计划求和、保留首现顺序。
// 复现上海场景——复旦工科试验班在源计划拆成多行子方向（同名同 mk），定位列不应重复刷屏。
func TestDedupGroupMajors(t *testing.T) {
	in := []core.GroupMajor{
		{MajorKey: "a", MajorName: "工科试验班", Plan: 2, PrevRank: 119},
		{MajorKey: "a", MajorName: "工科试验班", Plan: 2, PrevRank: 119},
		{MajorKey: "b", MajorName: "数学类", Plan: 1, PrevRank: 800},
		{MajorKey: "a", MajorName: "工科试验班", Plan: 3, PrevRank: 119},
	}
	got := dedupGroupMajors(in)
	if len(got) != 2 {
		t.Fatalf("want 2 个唯一专业，got %d: %+v", len(got), got)
	}
	if got[0].MajorKey != "a" || got[0].Plan != 7 {
		t.Errorf("a 应合并计划 2+2+3=7、居首，got key=%s plan=%d", got[0].MajorKey, got[0].Plan)
	}
	if got[1].MajorKey != "b" || got[1].Plan != 1 {
		t.Errorf("b 应保留原样，got key=%s plan=%d", got[1].MajorKey, got[1].Plan)
	}
}

// 无重复时原样返回（不改顺序、不动计划）。
func TestDedupGroupMajorsNoDup(t *testing.T) {
	in := []core.GroupMajor{{MajorKey: "x", Plan: 1}, {MajorKey: "y", Plan: 2}}
	got := dedupGroupMajors(in)
	if len(got) != 2 || got[0].MajorKey != "x" || got[1].MajorKey != "y" {
		t.Errorf("无重复应原样返回，got %+v", got)
	}
}
