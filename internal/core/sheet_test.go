package core

import "testing"

// 表头谓词：含「院校代码」+「专业名称」即为表头行。
func isPlanHeader(r []string) bool { return HasCell(r, "院校代码") && HasCell(r, "专业名称") }

func TestNewSheet(t *testing.T) {
	rows := [][]string{
		{"某省2026招生计划", ""},
		{"年份", "院校代码", "专业名称", "计划人数"},
		{"2026", "1003", "计算机", "5"},
		{"2026", "1003", "法学", "3"},
	}
	s, err := NewSheet(rows, isPlanHeader)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Data) != 2 {
		t.Fatalf("数据行 = %d，想要 2（表头之后）", len(s.Data))
	}
	if s.Col("专业名称") != 2 || s.Col("计划人数") != 3 {
		t.Errorf("列定位错：专业名称=%d 计划人数=%d", s.Col("专业名称"), s.Col("计划人数"))
	}
	if s.Col("不存在列") != -1 {
		t.Errorf("未命中列应为 -1，得 %d", s.Col("不存在列"))
	}
	if got := Cell(s.Data[0], s.Col("专业名称")); got != "计算机" {
		t.Errorf("首数据行专业名称 = %q，想要 计算机", got)
	}
}

func TestNewSheetNoHeader(t *testing.T) {
	rows := [][]string{{"无关", "数据"}, {"也无关", "行"}}
	if _, err := NewSheet(rows, isPlanHeader); err == nil {
		t.Error("找不到表头应返回错误")
	}
}
