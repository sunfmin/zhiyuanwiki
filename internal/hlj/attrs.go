package hlj

import "strings"

// IsCoop 判定中外合作办学：专业名/全称/备注 任一含「中外」或「合作办学」。见 ADR-0008。
func IsCoop(parts ...string) bool {
	for _, s := range parts {
		if strings.Contains(s, "中外") || strings.Contains(s, "合作办学") {
			return true
		}
	}
	return false
}

// ParseTuition 取学费字符串中的首段数字（元/年）；「待定」或无数字返回 0（即不计入高收费）。
func ParseTuition(s string) int {
	n, _ := parseLeadingInt(s)
	return n
}
