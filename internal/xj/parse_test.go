package xj_test

import (
	"path/filepath"
	"testing"

	"github.com/sunfmin/zhiyuanwiki/internal/xj"
	"github.com/xuri/excelize/v2"
)

// writeXLSX 把 rows 写成一张单 sheet 的 .xlsx 到临时目录，返回路径（供解析器读真实文件，
// 覆盖 OpenSheet→表头定位→keep 过滤 的整条路径）。
func writeXLSX(t *testing.T, rows [][]string) string {
	t.Helper()
	f := excelize.NewFile()
	t.Cleanup(func() { f.Close() })
	for i, r := range rows {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			t.Fatalf("CoordinatesToCellName: %v", err)
		}
		row := make([]any, len(r))
		for j, v := range r {
			row[j] = v
		}
		if err := f.SetSheetRow("Sheet1", cell, &row); err != nil {
			t.Fatalf("SetSheetRow: %v", err)
		}
	}
	path := filepath.Join(t.TempDir(), "in.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	return path
}

// 新疆专业录取分数：仅理科/文科本科、含位次留；物理（新高考科类）/艺术理/专科批/无位次 丢。
func TestParseScores(t *testing.T) {
	header := []string{"年份", "院校名称", "院校代码", "科类", "批次", "专业", "所属专业组", "选科要求", "最低分数", "最低位次"}
	rows := [][]string{
		header,
		{"2025", "新疆大学", "6501", "理科", "本科一批", "计算机科学与技术", "", "", "560", "8000"},
		{"2025", "新疆大学", "6501", "文科", "本科一批", "汉语言文学", "", "", "540", "3000"},
		{"2025", "某校", "1101", "物理", "本科批", "X", "", "", "600", "1000"},        // 新高考科类 理科/文科 之外 → 丢
		{"2025", "某艺院", "8801", "艺术理", "艺术类本科批", "音乐", "", "", "450", "5000"}, // 艺术科类 → 丢
		{"2025", "某专科", "9999", "理科", "专科批", "护理", "", "", "300", "50000"},     // 专科批 → 丢
		{"2025", "无位次校", "7777", "理科", "本科二批", "Y", "", "", "400", ""},        // 无位次 → 丢
	}
	got, err := xj.ParseScores(writeXLSX(t, rows))
	if err != nil {
		t.Fatalf("ParseScores: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2（仅理科/文科本科含位次）: %+v", len(got), got)
	}
	if got[0].Track != "理科" || got[0].MinRank != 8000 || got[0].MajorName != "计算机科学与技术" {
		t.Errorf("理科行解析错误: %+v", got[0])
	}
	if got[1].Track != "文科" || got[1].MinRank != 3000 {
		t.Errorf("文科行解析错误: %+v", got[1])
	}
}

// 新疆招生计划：所属专业组列全空（→ major 模型），仅理科/文科本科留。
func TestParsePlan(t *testing.T) {
	header := []string{"年份", "院校名称", "院校代码", "科类", "批次", "招生类型", "专业名称", "专业代码", "所属专业组", "专业备注", "选科要求", "招生人数", "学制(年)", "学费(元)"}
	rows := [][]string{
		header,
		{"2025", "新疆大学", "6501", "理科", "本科一批", "普通类", "软件工程", "01", "", "", "", "60", "四年", "5000"},
		{"2025", "新疆大学", "6501", "文科", "本科一批", "普通类", "法学", "02", "", "", "", "30", "四年", "5000"},
		{"2025", "某专科", "9999", "理科", "专科批", "普通类", "护理", "03", "", "", "", "50", "三年", "6000"}, // 专科 → 丢
	}
	got, err := xj.ParsePlan(writeXLSX(t, rows))
	if err != nil {
		t.Fatalf("ParsePlan: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2（仅理科/文科本科）: %+v", len(got), got)
	}
	if got[0].GroupCode != "" {
		t.Errorf("新疆无院校专业组，GroupCode 应为空: %q", got[0].GroupCode)
	}
	if got[0].Track != "理科" || got[0].MajorName != "软件工程" || got[0].Plan != 60 {
		t.Errorf("理科计划行解析错误: %+v", got[0])
	}
}

// 新疆一分一段无「批次」列：不能被批次过滤清零，理科/文科两科类都应留下分段。
func TestParseYiFenYiDuanNoBatch(t *testing.T) {
	header := []string{"年份", "科类", "分数(分)", "本段人数(人)", "累计人数(人)", "排名区间"}
	rows := [][]string{
		header,
		{"2025", "理科", "670~750", "9", "9", "1~9"},
		{"2025", "理科", "669", "5", "14", "10~14"},
		{"2025", "文科", "640~750", "3", "3", "1~3"},
	}
	got, err := xj.ParseYiFenYiDuan(writeXLSX(t, rows), "新疆", 2025)
	if err != nil {
		t.Fatalf("ParseYiFenYiDuan: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2（理科+文科，无批次列不应清零）", len(got))
	}
	byTrack := map[string]int{}
	for _, y := range got {
		byTrack[y.Track] = len(y.Entries)
	}
	if byTrack["理科"] != 2 {
		t.Errorf("理科分段数 = %d, want 2", byTrack["理科"])
	}
	if byTrack["文科"] != 1 {
		t.Errorf("文科分段数 = %d, want 1", byTrack["文科"])
	}
}
