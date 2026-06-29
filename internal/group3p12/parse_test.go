package group3p12

import (
	"testing"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

func sheet(t *testing.T, rows [][]string, isHeader func([]string) bool) *core.Sheet {
	t.Helper()
	s, err := core.NewSheet(rows, isHeader)
	if err != nil {
		t.Fatalf("NewSheet: %v", err)
	}
	return s
}

func TestParseScores(t *testing.T) {
	header := []string{"年份", "院校名称", "院校代码", "科类", "批次", "专业", "所属专业组", "选科要求", "最低分数", "最低位次"}
	rows := [][]string{
		header,
		{"2025", "郑州大学", "1101", "物理类", "本科批", "人工智能", "（05）", "首选物理", "640", "1196"},
		{"2025", "河南大学", "1102", "历史类", "本科批", "汉语言文学", "（01）", "首选历史", "610", "2300"},
		{"2025", "某专科", "9999", "物理类", "专科批", "护理", "", "", "400", "90000"},        // 专科批 → 丢
		{"2025", "某艺院", "8801", "艺术类（物理）", "艺术类本科批", "音乐", "", "", "500", "5000"}, // 艺术科类 → 丢
		{"2025", "无位次校", "7777", "物理类", "本科批", "X", "", "", "500", ""},            // 无位次 → 丢
	}
	got := parseScores(sheet(t, rows, scoreHeader))
	if len(got) != 2 {
		t.Fatalf("want 2 行（仅物理/历史本科含位次），got %d: %+v", len(got), got)
	}
	if got[0].Track != "物理" || got[0].GroupCode != "（05）" || got[0].MinRank != 1196 {
		t.Errorf("第一行解析错误: %+v", got[0])
	}
	if got[1].Track != "历史" { // 历史类 → 历史
		t.Errorf("科类未归一: %q", got[1].Track)
	}
}

// TestParsePlanGroupCodeColumn 覆盖超集解析的两点（相对 js/hn 简单版的额外能力）：
// 组代码列名走「专业组代码」、专业名带尾注用 StripParenTail 截断。
func TestParsePlanGroupCodeColumn(t *testing.T) {
	header := []string{"年份", "院校名称", "院校代码", "科类", "批次", "专业名称", "专业组代码", "专业组名称", "选科要求", "计划人数", "学制(年)", "学费(元)"}
	rows := [][]string{
		header,
		{"2025", "郑州大学", "1101", "物理类", "本科批", "计算机类（包含专业：软件工程、网络工程）（南校区）", "05", "第05组", "首选物理", "30", "四年", "5800"},
		{"2025", "某专科", "9999", "物理类", "专科批", "护理", "", "", "", "50", "三年", "6000"}, // 专科 → 丢
	}
	got := parsePlan(sheet(t, rows, planHeader))
	if len(got) != 1 {
		t.Fatalf("want 1 行，got %d", len(got))
	}
	p := got[0]
	if p.GroupCode != "05" || p.GroupName != "第05组" {
		t.Errorf("组代码/组名未走专业组代码/专业组名称列: %+v", p)
	}
	if p.MajorName != "计算机类" { // StripParenTail 截断尾注
		t.Errorf("专业名未截断尾注: %q", p.MajorName)
	}
	if p.Plan != 30 || p.Tuition != "5800" {
		t.Errorf("计划行解析错误: %+v", p)
	}
}

// TestParsePlanGroupCodeFallback 覆盖：无「专业组代码」列时回退「所属专业组」，无「专业组名称」时
// 组名用组代码兜底（与 js/hn 旧行为等价）。
func TestParsePlanGroupCodeFallback(t *testing.T) {
	header := []string{"年份", "院校名称", "院校代码", "科类", "批次", "专业名称", "所属专业组", "选科要求", "招生人数"}
	rows := [][]string{
		header,
		{"2025", "南昌大学", "2201", "历史类", "本科批", "法学", "（03）", "首选历史", "12"},
	}
	got := parsePlan(sheet(t, rows, planHeader))
	if len(got) != 1 {
		t.Fatalf("want 1 行，got %d", len(got))
	}
	p := got[0]
	if p.GroupCode != "（03）" || p.GroupName != "（03）" { // 组名兜底为组代码
		t.Errorf("组代码兜底错误: %+v", p)
	}
}

// TestParsePlanMajorColAndUnitSuffix 覆盖湖北/云南计划表形：列名为「专业」（非「专业名称」）、
// 学费表头带「(元/年)」单位后缀、科类为裸「物理/历史」（非「物理类」）。
func TestParsePlanMajorColAndUnitSuffix(t *testing.T) {
	header := []string{"年份", "院校名称", "院校代码", "科类", "批次", "专业", "专业代码", "所属专业组", "专业备注", "选科要求", "招生人数", "学制", "学费(元/年)"}
	rows := [][]string{
		header,
		{"2025", "武汉大学", "3301", "物理", "本科批", "临床医学", "01", "（801）", "", "首选物理", "20", "五年", "5850"},
	}
	got := parsePlan(sheet(t, rows, planHeader))
	if len(got) != 1 {
		t.Fatalf("want 1 行，got %d", len(got))
	}
	p := got[0]
	if p.Track != "物理" || p.MajorName != "临床医学" || p.GroupCode != "（801）" || p.Plan != 20 {
		t.Errorf("「专业」列/裸科类解析错误: %+v", p)
	}
	if p.Tuition != "5850" || p.Schooling != "五年" { // 学费(元/年)/学制 经 ColContains 命中
		t.Errorf("带单位后缀的学费/学制未命中: tuition=%q schooling=%q", p.Tuition, p.Schooling)
	}
}

func TestParseYiFenYiDuan(t *testing.T) {
	header := []string{"年份", "科类", "批次", "控制线(分)", "分数(分)", "本段人数(人)", "累计人数(人)"}
	rows := [][]string{
		header,
		{"2025", "物理类", "本科批", "422", "693-750", "126", "126"},
		{"2025", "物理类", "本科批", "422", "692", "18", "144"},
		{"2025", "历史类", "本科批", "438", "660", "10", "10"},
		{"2025", "物理类", "专科批", "200", "300", "5", "9999"}, // 专科 → 丢
	}
	got := parseYiFenYiDuan(sheet(t, rows, yfdHeader), "河南", 2025)
	if len(got) != 2 { // 物理 + 历史
		t.Fatalf("want 2 个(科类)，got %d", len(got))
	}
	var wuli *core.YiFenYiDuan
	for _, y := range got {
		if y.Track == "物理" {
			wuli = y
		}
	}
	if wuli == nil {
		t.Fatal("缺物理段")
	}
	if wuli.Total() != 144 { // 升序后最低分(692)累计=144
		t.Errorf("Total()=%d, want 144", wuli.Total())
	}
	if r, _ := wuli.ScoreToRank(693); r != 126 { // 693-750 顶段取前导数 693 → 累计 126
		t.Errorf("ScoreToRank(693)=%d, want 126", r)
	}
	if wuli.ControlLine != 422 {
		t.Errorf("控制线未取本科批: %d", wuli.ControlLine)
	}
}
