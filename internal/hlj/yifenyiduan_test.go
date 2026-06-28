package hlj

import "testing"

func TestTrackFromName(t *testing.T) {
	tests := map[string]string{
		"2024年黑龙江省高考（物理类）一分一段表.xlsx": "物理",
		"2024年黑龙江省高考（历史类）一分一段表.xlsx": "历史",
		"黑龙江2026物理类一分一段表.xlsx":       "物理",
		"2023年黑龙江省高考（理科）一分一段表.xlsx":  "", // 旧高考，跳过
		"2020年黑龙江省高考（文科）一分一段表.xlsx":  "",
	}
	for name, want := range tests {
		if got := trackFromName(name); got != want {
			t.Errorf("trackFromName(%q) = %q，想要 %q", name, got, want)
		}
	}
}

func TestIsCombinedYFD(t *testing.T) {
	combined := [][]string{{"年份", "科类", "批次", "控制线(分)", "分数(分)", "累计人数(人)"}}
	if !isCombinedYFD(combined) {
		t.Error("含「科类」列应判为合表")
	}
	perTrack := [][]string{{"分数", "人数", "累计人数"}}
	if isCombinedYFD(perTrack) {
		t.Error("逐科类表（无科类列）不应判为合表")
	}
}

func TestParseCombinedYFD(t *testing.T) {
	// 合表：物理/历史 各有 本科批 + 专科批两段，控制线分别 360/405（本科批）与 160（专科批）。
	// 本科批覆盖高分段、专科批覆盖低分段，拼起来才是全分布。
	rows := [][]string{
		{"年份", "科类", "批次", "控制线(分)", "分数(分)", "本段人数(人)", "累计人数(人)"},
		{"2025", "物理类", "本科批", "360", "694-750", "34", "34"},
		{"2025", "物理类", "本科批", "360", "360", "200", "85313"},
		{"2025", "物理类", "专科批", "160", "359", "150", "85500"},
		{"2025", "物理类", "专科批", "160", "130", "9", "117407"},
		{"2025", "历史类", "本科批", "405", "659", "1", "1"},
		{"2025", "历史类", "本科批", "405", "405", "80", "22977"},
		{"2025", "历史类", "专科批", "160", "130", "5", "54707"},
	}
	got, err := parseCombinedYFD(rows, "黑龙江", 2025)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("解析到 %d 个科类，想要 2", len(got))
	}
	byTrack := map[string]int{} // track → index
	for i, y := range got {
		byTrack[y.Track] = i
	}
	wuli := got[byTrack["物理"]]
	if wuli.ControlLine != 360 {
		t.Errorf("物理控制线 = %d，想要 360（本科批，非专科批 160）", wuli.ControlLine)
	}
	if len(wuli.Entries) != 4 {
		t.Errorf("物理分段 = %d，想要 4（本科批+专科批拼全分布）", len(wuli.Entries))
	}
	// Entries 升序：首段最低分 130，累计 117407 = 全省物理总人数（Total 不变量）。
	if wuli.Entries[0].Score != 130 || wuli.Total() != 117407 {
		t.Errorf("物理首段 = %+v，Total = %d，想要 score=130 Total=117407", wuli.Entries[0], wuli.Total())
	}
	lishi := got[byTrack["历史"]]
	if lishi.ControlLine != 405 {
		t.Errorf("历史控制线 = %d，想要 405", lishi.ControlLine)
	}
	if lishi.Total() != 54707 {
		t.Errorf("历史 Total = %d，想要 54707", lishi.Total())
	}
}
