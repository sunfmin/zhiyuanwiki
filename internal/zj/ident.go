package zj

import (
	"regexp"
	"strings"
)

var firstParenRe = regexp.MustCompile(`[（(]([^）)]*)[）)]`)

// majorIdent 把浙江大类招生的「方向」并入专业名，使每个方向成为独立的院校×专业单位。
// 方向取自专业备注的**首个括号**——但「（含…）」是"包含专业"的说明性括号、不是方向名，跳过。
//
// 浙大「工科试验班」一个名字下分图灵班/信息/海洋/材料…多个方向，各方向选科相同、
// 但计划与录取位次天差地别（2025：图灵班 202 名、海洋 6085 名）。若只按专业名聚合，
// 会把各方向计划求和（1096）却挂上最难方向的位次（202），自相矛盾。按方向拆分后，
// 每个方向的计划与往年位次各自对应；同名同方向逐年仍能合并（「（信息）」各年稳定）。
func majorIdent(name, remark string) string {
	name = strings.TrimSpace(name)
	m := firstParenRe.FindStringSubmatch(strings.TrimSpace(remark))
	if m == nil {
		return name
	}
	inner := strings.TrimSpace(m[1])
	if inner == "" || strings.HasPrefix(inner, "含") {
		return name // 「（含…）」是包含专业的说明，非方向名
	}
	return name + m[0]
}

var trailingDirRe = regexp.MustCompile(`[（(][^（）()]*[）)]$`)

// BaseMajorName 去掉 majorIdent 并入的方向后缀（末尾一个括号），还原跨校可比的专业基名。
// 用于专业浏览（zhuanye）的跨校聚合——叶子按方向细分，但专业索引按基名归并，避免被
// 「工科试验班（信息）」这类院校特定方向塞爆。无方向后缀的名字原样返回。
func BaseMajorName(name string) string {
	return strings.TrimSpace(trailingDirRe.ReplaceAllString(strings.TrimSpace(name), ""))
}
