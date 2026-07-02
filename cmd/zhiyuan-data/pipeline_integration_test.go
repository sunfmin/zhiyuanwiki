package main

import (
	"path/filepath"
	"testing"

	"github.com/sunfmin/zhiyuanwiki/internal/group3p12"
	"github.com/sunfmin/zhiyuanwiki/internal/store"
	"github.com/xuri/excelize/v2"
)

// 接省回归的快速护栏：走真实 parse → store → project 三层（group3p12 解析 xlsx → SQLite staging →
// buildDBBundle 投影），喂极小 fixture、断言投影产出。新接一省若任一层回归，本测试即红——
// 无需起浏览器跑慢的 render 测。

func writeXLSXMain(t *testing.T, name string, rows [][]string) string {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	for i, r := range rows {
		for j, c := range r {
			cell, err := excelize.CoordinatesToCellName(j+1, i+1)
			if err != nil {
				t.Fatalf("CoordinatesToCellName: %v", err)
			}
			if err := f.SetCellValue(sheet, cell, c); err != nil {
				t.Fatalf("SetCellValue: %v", err)
			}
		}
	}
	path := filepath.Join(t.TempDir(), name)
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	return path
}

// groupByTrack 在某校的 2026 报考视图里按科类取组（测试辅助；Groups2026 元素类型经 hlj 别名指向
// core.Group2026，这里只读其字段、不命名类型）。
func groupByTrack(d schoolDetail, track string) (string, int, bool) {
	for _, g := range d.Groups2026 {
		if g.Track == track && len(g.Majors) > 0 {
			return g.Majors[0].MajorName, g.Majors[0].PrevRank, true
		}
	}
	return "", 0, false
}

func TestPipelineParseStoreProject(t *testing.T) {
	scoresPath := writeXLSXMain(t, "scores.xlsx", [][]string{
		{"年份", "院校名称", "院校代码", "科类", "批次", "专业", "所属专业组", "选科要求", "最低分数", "最低位次"},
		{"2025", "测试大学", "1101", "物理类", "本科批", "计算机", "（01）", "首选物理", "640", "3000"},
		{"2024", "测试大学", "1101", "物理类", "本科批", "计算机", "（01）", "首选物理", "630", "3500"},
		{"2025", "测试大学", "1101", "历史类", "本科批", "汉语言文学", "（02）", "首选历史", "600", "8000"},
	})
	planPath := writeXLSXMain(t, "plan.xlsx", [][]string{
		{"年份", "院校名称", "院校代码", "科类", "批次", "专业名称", "专业组代码", "专业组名称", "选科要求", "计划人数", "学制(年)", "学费(元)"},
		{"2025", "测试大学", "1101", "物理类", "本科批", "计算机", "01", "第01组", "首选物理", "30", "四年", "5000"},
		{"2025", "测试大学", "1101", "历史类", "本科批", "汉语言文学", "02", "第02组", "首选历史", "20", "四年", "5000"},
	})

	// parse（真实公共接口）
	scores, err := group3p12.ParseScores(scoresPath)
	if err != nil {
		t.Fatalf("ParseScores: %v", err)
	}
	plan, err := group3p12.ParsePlan(planPath)
	if err != nil {
		t.Fatalf("ParsePlan: %v", err)
	}

	// store（构建期 SQLite staging）
	dbPath := filepath.Join(t.TempDir(), "itest.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	if err := db.ReplaceScores("itest", scores); err != nil {
		t.Fatalf("ReplaceScores: %v", err)
	}
	if err := db.ReplacePlan("itest", plan); err != nil {
		t.Fatalf("ReplacePlan: %v", err)
	}
	db.Close()

	// project（真实投影路径，group 模型）
	p := province{slug: "itest", name: "集成测试省", tracks: []string{"物理", "历史"}, model: "group"}
	b := buildDBBundle(dbPath, p)

	if len(b.schools) != 1 || b.schools[0].Code != "1101" || b.schools[0].Name != "测试大学" {
		t.Fatalf("院校聚合错误: %+v", b.schools)
	}
	// 主键是归一化校名（ADR-0021），详情按 Key 取，不按代号。
	d := b.details[b.schools[0].Key]
	if len(d.Leaves) != 2 {
		t.Fatalf("应有 2 个院校×专业叶子（计算机+汉语言），got %d", len(d.Leaves))
	}
	if len(d.Groups2026) != 2 {
		t.Fatalf("应有 2 个院校专业组（物理/历史各一），got %d", len(d.Groups2026))
	}
	// 物理组：计算机，往年位次取最近年（2025）的 3000，而非 2024 的 3500。
	if name, rank, ok := groupByTrack(d, "物理"); !ok || name != "计算机" || rank != 3000 {
		t.Errorf("物理组挂接错误: name=%q rank=%d ok=%v", name, rank, ok)
	}
	// 历史组：汉语言文学，往年位次 8000。
	if name, rank, ok := groupByTrack(d, "历史"); !ok || name != "汉语言文学" || rank != 8000 {
		t.Errorf("历史组挂接错误: name=%q rank=%d ok=%v", name, rank, ok)
	}
}
