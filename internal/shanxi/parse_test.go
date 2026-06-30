package shanxi

import (
	"testing"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

func sheet(t *testing.T, rows [][]string) *core.Sheet {
	t.Helper()
	s, err := core.NewSheet(rows, scoreHeader)
	if err != nil {
		t.Fatalf("NewSheet: %v", err)
	}
	return s
}

// TestParseScores 覆盖山西录取分数表的特异点：首行标题（真表头在第二行）、无院校代码列、
// 列名异形（录取最低分/录取最低位次）、专业全称带括号尾注须截断、理科/专科批/无位次须丢、
// 年份由调用方注入（源表无年份列）。
func TestParseScores(t *testing.T) {
	title := []string{"山西2024专业录取分数线", "", "", "", "", "", "", "", "", ""}
	header := []string{"生源地", "科类", "批次", "院校名称", "专业组名称", "专业全称", "专业层次", "选科要求", "录取最低分", "录取最低位次"}
	rows := [][]string{
		title,
		header,
		{"山西", "物理", "本科批", "空军军医大学", "第303组", "口腔医学(五年)(色盲色弱不予录取)", "本科", "化学", "609", "6015"},
		{"山西", "历史", "本科批", "山西大学", "第101组", "汉语言文学", "本科", "不限", "560", "3200"},
		{"山西", "理科", "本科二批C", "某专科", "第999组", "护理", "专科", "", "400", "90000"}, // 老文理 理科 → 丢
		{"山西", "物理", "专科批", "某高职", "第888组", "机电", "专科", "", "380", "120000"},   // 专科批 → 丢
		{"山西", "物理", "本科批", "无位次校", "第777组", "X", "本科", "", "500", ""},        // 无位次 → 丢
	}
	got := parseScores(sheet(t, rows), 2025)
	if len(got) != 2 {
		t.Fatalf("want 2 行（仅物理/历史本科含位次），got %d: %+v", len(got), got)
	}
	r0 := got[0]
	if r0.Year != 2025 {
		t.Errorf("年份未注入: %d", r0.Year)
	}
	if r0.Track != "物理" || r0.SchoolName != "空军军医大学" || r0.MinScore != 609 || r0.MinRank != 6015 {
		t.Errorf("第一行解析错误: %+v", r0)
	}
	if r0.MajorName != "口腔医学" { // StripParenTail 截去 (五年)(…) 尾注，与招生计划裸名挂接
		t.Errorf("专业全称尾注未截断: %q", r0.MajorName)
	}
	if r0.SchoolCode != "" { // 源表无院校代码列；由 importShanxi 按校名回填
		t.Errorf("SchoolCode 应留空待回填，got %q", r0.SchoolCode)
	}
	if got[1].Track != "历史" || got[1].MajorName != "汉语言文学" {
		t.Errorf("第二行解析错误: %+v", got[1])
	}
}
