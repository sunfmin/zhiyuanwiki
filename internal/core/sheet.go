package core

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// Sheet 是一张 xlsx 首个 sheet 已定位到表头行后的视图：表头 + 其后的数据行 + 列定位。
//
// 它把每个省份解析器都要写的同一串样板——开文件 → 取首个 sheet → 读所有行 →
// 按谓词找表头 → 建列定位闭包——收进一处深模块。调用方只需给出「哪一行是表头」的判据，
// 拿回数据行游标与 Col() 列定位；省份专属的「读哪些列、怎么过滤、装成什么结构」仍各自留在
// 各省解析器里（那些是真正因省而异的，不该塞进一个共用大配置）。
type Sheet struct {
	Header []string   // 表头行
	Data   [][]string // 表头之后的数据行
}

// OpenSheet 打开 path 的首个 sheet，读出所有行，用 isHeader 定位表头，返回表头后的数据视图。
// 打不开 / 无 sheet / 找不到表头 → 错误。
func OpenSheet(path string, isHeader func(row []string) bool) (*Sheet, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("打开 %s: %w", path, err)
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("%s: 无 sheet", path)
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("读 %s: %w", path, err)
	}
	s, err := NewSheet(rows, isHeader)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

// NewSheet 在已读出的 rows 上定位表头（便于用合成数据做表驱动测试，免开真文件）。
func NewSheet(rows [][]string, isHeader func(row []string) bool) (*Sheet, error) {
	for i, r := range rows {
		if isHeader(r) {
			return &Sheet{Header: r, Data: rows[i+1:]}, nil
		}
	}
	return nil, fmt.Errorf("未找到表头行")
}

// Col 返回表头中首个匹配 names 之一的列下标；未命中 -1（同 FindCol）。
func (s *Sheet) Col(names ...string) int { return FindCol(s.Header, names...) }
