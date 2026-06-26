// Package core 是各省份共用的、与省份无关的高考数据原语：专业名归一化与键、
// 行表聚合成院校×专业叶子、一分一段换算、等效位次、xlsx 单元格助手、院校属性小工具。
// 省份专属解析（黑龙江万师兄旧表、浙江一表联动等）放在各自的 internal/<省> 包里。
package core

import (
	"regexp"
	"strconv"
	"strings"
)

var digitsRe = regexp.MustCompile(`\d+`)

// ParseLeadingInt 抽取字符串中第一段连续数字。处理 "700以上" → 700、"693-750" → 693。
func ParseLeadingInt(s string) (int, bool) {
	m := digitsRe.FindString(strings.TrimSpace(s))
	if m == "" {
		return 0, false
	}
	n, err := strconv.Atoi(m)
	if err != nil {
		return 0, false
	}
	return n, true
}

// NormSchoolCode 归一化院校代码到 4 位（不足左补 0）。浙江官方院校代码为 4 位定宽，
// 但 2026 计划表把代码存成数字、丢了前导 0（如「0001」存成「1」）；据此对齐到分数表口径。
// 非纯数字或已 ≥4 位的代码原样返回（不截断）。
func NormSchoolCode(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || len(s) >= 4 {
		return s
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return s
		}
	}
	return strings.Repeat("0", 4-len(s)) + s
}

// Cell 安全取行 r 的第 i 个单元格（越界返回空串）。
func Cell(r []string, i int) string {
	if i < 0 || i >= len(r) {
		return ""
	}
	return r[i]
}

// HasCell 报告行 row 中是否有单元格 trim 后精确等于 s。
func HasCell(row []string, s string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) == s {
			return true
		}
	}
	return false
}

// FindCol 返回表头中精确等于任一候选名的列下标；找不到返回 -1。
func FindCol(header []string, names ...string) int {
	for i, c := range header {
		cc := strings.TrimSpace(c)
		for _, n := range names {
			if cc == n {
				return i
			}
		}
	}
	return -1
}
