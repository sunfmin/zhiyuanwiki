package zj

import (
	"reflect"
	"testing"
)

func TestToCorePlanMapping(t *testing.T) {
	in := []PlanRow2026{{
		Year: 2026, SchoolCode: "0001", SchoolName: "浙江大学",
		AdmitType: "中外合作办学", MajorName: "计算机科学与技术", Remark: "中外合作",
		SelKe: "物理", Plan: 30, Schooling: "4", Tuition: "60000",
	}}
	got := ToCorePlan(in)
	if len(got) != 1 {
		t.Fatalf("len = %d，想要 1", len(got))
	}
	c := got[0]
	if c.Track != Track {
		t.Errorf("Track = %q，想要 %q（综合）", c.Track, Track)
	}
	if c.GroupCode != "" {
		t.Errorf("GroupCode = %q，想要空（major 模型无组）", c.GroupCode)
	}
	if c.Batch != "中外合作办学" {
		t.Errorf("Batch = %q，想要「中外合作办学」（招生类型落 Batch）", c.Batch)
	}
	if c.MajorName != "计算机科学与技术" || c.SelKe != "物理" || c.Plan != 30 {
		t.Errorf("字段映射错：%+v", c)
	}
}

func TestPlanRoundTrip(t *testing.T) {
	in := []PlanRow2026{
		{Year: 2026, SchoolCode: "0001", SchoolName: "浙江大学", AdmitType: "普通类",
			MajorName: "数学类", Remark: "", SelKe: "物理", Plan: 50, Schooling: "4", Tuition: "5300"},
		{Year: 2026, SchoolCode: "0002", SchoolName: "杭州电子科技大学", AdmitType: "中外合作办学",
			MajorName: "会计学", Remark: "中外合作", SelKe: "不限", Plan: 20, Schooling: "4", Tuition: "60000"},
	}
	got := FromCorePlan(ToCorePlan(in))
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip 不一致：\n got=%+v\nwant=%+v", got, in)
	}
}
