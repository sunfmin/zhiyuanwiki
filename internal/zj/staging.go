package zj

import "github.com/sunfmin/zhiyuanwiki/internal/core"

// Track 是浙江的唯一科类「综合」（3+3，不分物理/历史）——浙江一分一段解析与 major 报考视图的默认科类口径。
const Track = "综合"

// ParseYiFenYiDuan 解析浙江一分一段 xlsx（综合科类）→ 一张表，签名与统一 staging 的 YFD 解析器一致，
// 登记在 provParsers["zj"].YFD（浙江综合单列格式与通用 group3p12 表头不同，故留用本解析器）。
func ParseYiFenYiDuan(path, province string, year int) ([]*core.YiFenYiDuan, error) {
	y, err := core.ParseYiFenYiDuanXLSX(path, province, Track, year)
	if err != nil {
		return nil, err
	}
	return []*core.YiFenYiDuan{y}, nil
}
