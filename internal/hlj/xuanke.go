package hlj

import "strings"

// 新高考再选科目（首选物理/历史由科类承载，不在此判定）。
var reXuanSubjects = []string{"物理", "历史", "化学", "生物", "政治", "地理"}

func subjectsIn(req string) []string {
	var out []string
	for _, s := range reXuanSubjects {
		if strings.Contains(req, s) {
			out = append(out, s)
		}
	}
	return out
}

// SelKeAllows 判断选科要求 req 是否允许 chosen 选科组合报考。
// req 例："不限" / "化学" / "化学和生物" / "化学或生物" / "政治"。
// chosen 为考生已选科目集合（含首选与再选，如 物化生 = {物理,化学,生物}）。
// 无法识别的要求默认放行（保守，不误杀）。
func SelKeAllows(req string, chosen map[string]bool) bool {
	req = strings.TrimSpace(req)
	if req == "" || strings.Contains(req, "不限") {
		return true
	}
	subs := subjectsIn(req)
	if len(subs) == 0 {
		return true
	}
	if strings.Contains(req, "或") {
		for _, s := range subs {
			if chosen[s] {
				return true
			}
		}
		return false
	}
	// "和" 或单科：要求列出的科目全部已选。
	for _, s := range subs {
		if !chosen[s] {
			return false
		}
	}
	return true
}
