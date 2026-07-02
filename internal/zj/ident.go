package zj

import (
	"regexp"
	"strings"
)

var trailingDirRe = regexp.MustCompile(`[（(][^（）()]*[）)]$`)

// BaseMajorName 去掉浙江大类招生并入的方向后缀（末尾一个括号），还原跨校可比的专业基名。
// 用于专业浏览（zhuanye）的跨校聚合——叶子按方向细分，但专业索引按基名归并，避免被
// 「工科试验班（信息）」这类院校特定方向塞爆。无方向后缀的名字原样返回。
func BaseMajorName(name string) string {
	return strings.TrimSpace(trailingDirRe.ReplaceAllString(strings.TrimSpace(name), ""))
}
