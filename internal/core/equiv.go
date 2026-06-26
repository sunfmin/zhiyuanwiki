package core

import "math"

// YearTrack 标识某年某科类，用作一分一段总人数表的键。
type YearTrack struct {
	Year  int
	Track string
}

// EquivRank 等效位次：把 from 年的位次 rank 按一分一段总人数比例缩放到 to 年口径。
// 缺任一年的总人数则返回原位次——位次本身即是跨年比较的近似口径（见 CONTEXT「位次」）。
func EquivRank(rank int, from, to YearTrack, totals map[YearTrack]int) int {
	if rank <= 0 || from == to {
		return rank
	}
	tf, okf := totals[from]
	tt, okt := totals[to]
	if !okf || !okt || tf == 0 {
		return rank
	}
	return int(math.Round(float64(rank) * float64(tt) / float64(tf)))
}
