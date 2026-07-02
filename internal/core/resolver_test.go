package core

import "testing"

// TestResolverReformMerge：同校跨老/新高考换号（年份不相交）归为一个渠道，历史不劈断。
// 深圳大学 2023 用 2046、2024-25 用 2044 —— 一个普通渠道。
func TestResolverReformMerge(t *testing.T) {
	r := BuildSchoolResolver([]IdentRow{
		{"2046", "深圳大学", 2023},
		{"2044", "深圳大学", 2024},
		{"2044", "深圳大学", 2025},
	})
	if e := r.Entity("深圳大学"); e != NormName("深圳大学") {
		t.Fatalf("entity=%q", e)
	}
	if c1, c2 := r.Channel("深圳大学", "2046"), r.Channel("深圳大学", "2044"); c1 != c2 {
		t.Fatalf("深圳 2046/2044 应同渠道，得 %q vs %q", c1, c2)
	}
	if rc := r.RepCode(NormName("深圳大学")); rc != "2044" {
		t.Fatalf("代表代号应为最新年 2044，得 %q", rc)
	}
}

// TestResolverCodeReuseSplits：同一代号跨年被两所无关学校复用（非相邻、前导不同）→ 分成两个实体。
// 2046：2023 深圳、2026 广州。
func TestResolverCodeReuseSplits(t *testing.T) {
	r := BuildSchoolResolver([]IdentRow{
		{"2046", "深圳大学", 2023},
		{"2044", "深圳大学", 2024},
		{"2046", "广州大学", 2026},
	})
	if r.Entity("深圳大学") == r.Entity("广州大学") {
		t.Fatal("深圳与广州复用 2046，绝不能并成一个实体")
	}
	if c := r.Channel("广州大学", "2046"); c != "2046" {
		t.Fatalf("广州的 2046 自成渠道，得 %q", c)
	}
}

// TestResolverMultiChannel：同校同年多代号（普通/专项，年份重叠）分成不同渠道；老高考码并入普通。
// 黑龙江大学：1405@2023(老普通)、1408@2024-25(普通)、6001@2024-25(专项)。
func TestResolverMultiChannel(t *testing.T) {
	r := BuildSchoolResolver([]IdentRow{
		{"1405", "黑龙江大学", 2023},
		{"1408", "黑龙江大学", 2024},
		{"1408", "黑龙江大学", 2025},
		{"6001", "黑龙江大学", 2024},
		{"6001", "黑龙江大学", 2025},
	})
	pu, zx := r.Channel("黑龙江大学", "1408"), r.Channel("黑龙江大学", "6001")
	if pu == zx {
		t.Fatalf("普通 1408 与专项 6001 同年并存，应分渠道，得同为 %q", pu)
	}
	if lao := r.Channel("黑龙江大学", "1405"); lao != pu {
		t.Fatalf("老高考 1405 应并入普通渠道 %q，得 %q", pu, lao)
	}
}

// TestResolverUnionIncludesPlanOnly：只在计划里出现、无历史录取的新招生校（广州大学）
// 也进院校全集——这是「广州大学在黑龙江消失」的根治（院校一览曾只由分数构建）。
func TestResolverUnionIncludesPlanOnly(t *testing.T) {
	rows := append(
		IdentRowsFromScores([]MajorScoreRow{ // 深圳有历史，广州无
			{Year: 2023, SchoolCode: "2046", SchoolName: "深圳大学"},
			{Year: 2024, SchoolCode: "2044", SchoolName: "深圳大学"},
		}),
		IdentRowsFromPlan([]PlanRow{ // 广州只在 2026 计划出现
			{Year: 2026, SchoolCode: "2046", SchoolName: "广州大学"},
			{Year: 2026, SchoolCode: "2044", SchoolName: "深圳大学"},
		})...,
	)
	r := BuildSchoolResolver(rows)
	has := map[string]bool{}
	for _, e := range r.Entities() {
		has[r.Name(e)] = true
	}
	if !has["广州大学"] {
		t.Fatal("广州大学（计划独有、无历史）必须进院校全集")
	}
	if r.Entity("深圳大学") == r.Entity("广州大学") {
		t.Fatal("深圳与广州复用代号 2046，不能并成一个实体")
	}
}

// TestResolverRename：同代号相邻年、校名共享前缀（升格）→ 并入最新名的实体。
func TestResolverRename(t *testing.T) {
	r := BuildSchoolResolver([]IdentRow{
		{"1612", "湖州师范学院", 2025},
		{"1612", "湖州师范大学", 2026},
	})
	if r.Entity("湖州师范学院") != r.Entity("湖州师范大学") {
		t.Fatal("湖州师范学院/大学 应并成同一实体")
	}
	if n := r.Name(r.Entity("湖州师范学院")); n != "湖州师范大学" {
		t.Fatalf("规范名应取最新的『湖州师范大学』，得 %q", n)
	}
}
