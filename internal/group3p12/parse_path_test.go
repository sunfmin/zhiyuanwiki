package group3p12

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// 这些测试走**公共接口** ParseScores/ParsePlan/ParseYiFenYiDuan(path)，覆盖 OpenSheet 的接线
// （开文件 → 取首 sheet → 定位表头 → 解析），而非像 parse_test.go 那样直接喂 NewSheet 行。

// writeXLSX 把 rows 写成一个临时 xlsx 文件（首 sheet）。
func writeXLSX(t *testing.T, rows [][]string) string {
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
	path := filepath.Join(t.TempDir(), "fixture.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	return path
}

func TestParseScoresThroughPath(t *testing.T) {
	path := writeXLSX(t, [][]string{
		{"年份", "院校名称", "院校代码", "科类", "批次", "专业", "所属专业组", "选科要求", "最低分数", "最低位次"},
		{"2025", "测试大学", "1101", "物理类", "本科批", "计算机", "（01）", "首选物理", "640", "3000"},
		{"2025", "某专科", "9999", "物理类", "专科批", "护理", "", "", "400", "90000"}, // 专科 → 丢
	})
	got, err := ParseScores(path)
	if err != nil {
		t.Fatalf("ParseScores: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 行（仅本科含位次），got %d: %+v", len(got), got)
	}
	if got[0].Track != "物理" || got[0].SchoolCode != "1101" || got[0].MinRank != 3000 {
		t.Errorf("解析错误: %+v", got[0])
	}
}

func TestParsePlanThroughPath(t *testing.T) {
	path := writeXLSX(t, [][]string{
		{"年份", "院校名称", "院校代码", "科类", "批次", "专业名称", "专业组代码", "专业组名称", "选科要求", "计划人数", "学制(年)", "学费(元)"},
		{"2025", "测试大学", "1101", "历史类", "本科批", "汉语言文学（试验班）", "02", "第02组", "首选历史", "20", "四年", "5000"},
	})
	got, err := ParsePlan(path)
	if err != nil {
		t.Fatalf("ParsePlan: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 行，got %d", len(got))
	}
	p := got[0]
	if p.Track != "历史" || p.MajorName != "汉语言文学" || p.GroupCode != "02" || p.Plan != 20 || p.Tuition != "5000" {
		t.Errorf("解析错误（含 StripParenTail/单位后缀列）: %+v", p)
	}
}

func TestParseYiFenYiDuanThroughPath(t *testing.T) {
	path := writeXLSX(t, [][]string{
		{"年份", "科类", "批次", "控制线(分)", "分数(分)", "本段人数(人)", "累计人数(人)"},
		{"2025", "物理类", "本科批", "422", "700", "100", "100"},
		{"2025", "物理类", "本科批", "422", "699", "20", "120"},
	})
	got, err := ParseYiFenYiDuan(path, "测试", 2025)
	if err != nil {
		t.Fatalf("ParseYiFenYiDuan: %v", err)
	}
	if len(got) != 1 || got[0].Track != "物理" {
		t.Fatalf("want 1 个物理科类表，got %+v", got)
	}
	if got[0].Total() != 120 {
		t.Errorf("Total()=%d, want 120", got[0].Total())
	}
}

func TestParseScoresFileNotFound(t *testing.T) {
	if _, err := ParseScores(filepath.Join(t.TempDir(), "missing.xlsx")); err == nil {
		t.Fatal("文件不存在应返回错误")
	}
}

func TestParseScoresHeaderNotFound(t *testing.T) {
	// 表头行不含「院校代码/最低位次」→ OpenSheet 定位不到表头 → 错误（接线失败路径）。
	path := writeXLSX(t, [][]string{
		{"无关列A", "无关列B"},
		{"x", "y"},
	})
	if _, err := ParseScores(path); err == nil {
		t.Fatal("找不到表头应返回错误")
	}
}
