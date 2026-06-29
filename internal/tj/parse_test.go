package tj

import (
	"testing"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

func TestParsePlan(t *testing.T) {
	rows := [][]string{
		{"2025天津招生计划"}, // 标题行，OpenSheet 跳过
		{"批次", "专业组代码", "院校名称", "专业代码", "专业名称", "备注", "计划数", "选科", "学费", "专业组", "学制"},
		{"普通类本科批A阶段", "005603", "北京大学", "19", "中国语言文学类", "校本部", "1", "不限", "5000", "03", "四年"},
		{"普通类本科批A阶段", "161801", "中国人民大学", "14", "信息资源管理", "新媒体班", "1", "不限", "5000", "01", "四年"},
		{"艺考类本科统考A(美术与设计学)类", "990001", "某艺院", "01", "美术学", "", "5", "不限", "8000", "01", "四年"}, // 艺考 → 丢
		{"普通类高职高专批", "880010", "某高职", "01", "护理", "", "10", "不限", "6000", "10", "三年"},          // 高职 → 丢
	}
	s, err := core.NewSheet(rows, planHeader)
	if err != nil {
		t.Fatalf("NewSheet: %v", err)
	}
	got := parsePlan(s)
	if len(got) != 2 {
		t.Fatalf("want 2 行（仅普通类本科批），got %d: %+v", len(got), got)
	}
	// 院校代码 = 专业组代码 剥去专业组后缀
	if got[0].SchoolCode != "0056" || got[0].GroupCode != "03" || got[0].Track != "综合" {
		t.Errorf("北京大学行解析错误（院校代码应=0056）: %+v", got[0])
	}
	if got[1].SchoolCode != "1618" || got[1].GroupCode != "01" || got[1].Plan != 1 {
		t.Errorf("中国人民大学行解析错误（院校代码应=1618）: %+v", got[1])
	}
}
