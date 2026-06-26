package hlj

import (
	"regexp"
	"strings"

	"github.com/xuri/excelize/v2"
)

// SchoolTag 是院校层次标签（985 / 211 / 双一流），院校级属性，与年份无关。
// 来源：万师兄旧格式表（含「学校」「_985」「_211」「双一流」列，值为 是/否）。
type SchoolTag struct {
	Is985         bool `json:"is985,omitempty"`
	Is211         bool `json:"is211,omitempty"`
	IsShuangYiLiu bool `json:"isShuangYiLiu,omitempty"`
}

func (t SchoolTag) any() bool { return t.Is985 || t.Is211 || t.IsShuangYiLiu }

// TagIndex 按院校名查层次标签：先精确（归一化后），再退到去括号基名（覆盖
// 「中国石油大学(华东)」这类分校——继承母体标签）。多文件可合并，按位 OR。
type TagIndex struct {
	byNorm map[string]*SchoolTag
	byBase map[string]*SchoolTag
}

func NewTagIndex() *TagIndex {
	return &TagIndex{byNorm: map[string]*SchoolTag{}, byBase: map[string]*SchoolTag{}}
}

var parenRe = regexp.MustCompile(`[(（].*?[)）]`)

// normName 归一化院校名：全角括号/空格→半角、去空白。
func normName(s string) string {
	s = strings.NewReplacer("（", "(", "）", ")", "　", "", " ", "").Replace(strings.TrimSpace(s))
	return s
}

// baseName 去掉括号注记（分校/校区），用于母体标签继承。
func baseName(s string) string {
	return strings.TrimSpace(parenRe.ReplaceAllString(normName(s), ""))
}

func (ti *TagIndex) merge(m map[string]*SchoolTag, key string, t SchoolTag) {
	if key == "" {
		return
	}
	cur := m[key]
	if cur == nil {
		cur = &SchoolTag{}
		m[key] = cur
	}
	cur.Is985 = cur.Is985 || t.Is985
	cur.Is211 = cur.Is211 || t.Is211
	cur.IsShuangYiLiu = cur.IsShuangYiLiu || t.IsShuangYiLiu
}

// AddRows 解析一张旧格式表（首行即表头）的层次列并并入索引。无相关列则跳过。
func (ti *TagIndex) AddRows(rows [][]string) {
	if len(rows) == 0 {
		return
	}
	headerIdx := -1
	for i := 0; i < len(rows) && i < 4; i++ {
		if hasCell(rows[i], "学校") && hasCell(rows[i], "双一流") {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return
	}
	h := rows[headerIdx]
	cName := findCol(h, "学校", "院校名称")
	c985 := findCol(h, "_985", "985")
	c211 := findCol(h, "_211", "211")
	cSyl := findCol(h, "双一流")
	if cName < 0 {
		return
	}
	for _, r := range rows[headerIdx+1:] {
		name := strings.TrimSpace(cell(r, cName))
		if name == "" {
			continue
		}
		t := SchoolTag{
			Is985:         cell(r, c985) == "是",
			Is211:         cell(r, c211) == "是",
			IsShuangYiLiu: cell(r, cSyl) == "是",
		}
		if !t.any() {
			continue
		}
		ti.merge(ti.byNorm, normName(name), t)
		ti.merge(ti.byBase, baseName(name), t)
	}
}

// Lookup 查院校层次标签：精确名优先，再退到去括号基名。
func (ti *TagIndex) Lookup(name string) (SchoolTag, bool) {
	if t, ok := ti.byNorm[normName(name)]; ok {
		return *t, true
	}
	if t, ok := ti.byBase[baseName(name)]; ok {
		return *t, true
	}
	return SchoolTag{}, false
}

// Len 返回已收录的（归一化名）院校数，用于日志。
func (ti *TagIndex) Len() int { return len(ti.byNorm) }

// LoadSchoolTags 打开多个 xlsx，解析其首个 sheet 的层次列并合并。打不开/无相关列的文件静默跳过。
func LoadSchoolTags(paths []string) *TagIndex {
	ti := NewTagIndex()
	for _, p := range paths {
		f, err := excelize.OpenFile(p)
		if err != nil {
			continue
		}
		sheets := f.GetSheetList()
		if len(sheets) > 0 {
			if rows, err := f.GetRows(sheets[0]); err == nil {
				ti.AddRows(rows)
			}
		}
		_ = f.Close()
	}
	return ti
}
