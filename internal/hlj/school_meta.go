package hlj

import (
	"strings"

	"github.com/xuri/excelize/v2"
)

// SchoolMeta 是院校级属性（与年份无关），用于位次定位结果过滤。来自万师兄旧格式表，
// 按校名挂接（含 985/211/双一流，与 tags.go 同源）。城市层级由 CityTier 另算。见 ADR-0008。
type SchoolMeta struct {
	Province      string // 省份（院校所在地，非考生生源地）
	City          string // 城市
	Ownership     string // 办学性质：公办 / 民办
	Kind          string // 学校类别：综合类 / 理工类 / 师范类 …
	Is985         bool
	Is211         bool
	IsShuangYiLiu bool
}

func (m SchoolMeta) any() bool {
	return m.Province != "" || m.City != "" || m.Ownership != "" || m.Kind != "" ||
		m.Is985 || m.Is211 || m.IsShuangYiLiu
}

// Levels 返回该校的层次标签数组（985/211/双一流），仅含为真者。
func (m SchoolMeta) Levels() []string {
	var lv []string
	if m.Is985 {
		lv = append(lv, "985")
	}
	if m.Is211 {
		lv = append(lv, "211")
	}
	if m.IsShuangYiLiu {
		lv = append(lv, "双一流")
	}
	return lv
}

// SchoolMetaIndex 按校名查院校属性：先精确（归一化），再退到去括号基名（分校继承母体）。
// 多文件合并：字符串取首个非空、布尔按位 OR。
type SchoolMetaIndex struct {
	byNorm map[string]*SchoolMeta
	byBase map[string]*SchoolMeta
}

func NewSchoolMetaIndex() *SchoolMetaIndex {
	return &SchoolMetaIndex{byNorm: map[string]*SchoolMeta{}, byBase: map[string]*SchoolMeta{}}
}

func (si *SchoolMetaIndex) merge(m map[string]*SchoolMeta, key string, v SchoolMeta) {
	if key == "" {
		return
	}
	cur := m[key]
	if cur == nil {
		cur = &SchoolMeta{}
		m[key] = cur
	}
	if cur.Province == "" {
		cur.Province = v.Province
	}
	if cur.City == "" {
		cur.City = v.City
	}
	if cur.Ownership == "" {
		cur.Ownership = v.Ownership
	}
	if cur.Kind == "" {
		cur.Kind = v.Kind
	}
	cur.Is985 = cur.Is985 || v.Is985
	cur.Is211 = cur.Is211 || v.Is211
	cur.IsShuangYiLiu = cur.IsShuangYiLiu || v.IsShuangYiLiu
}

// AddRows 解析一张旧格式表（首行即表头）的院校属性列并并入索引。无相关列则跳过。
func (si *SchoolMetaIndex) AddRows(rows [][]string) {
	headerIdx := -1
	for i := 0; i < len(rows) && i < 4; i++ {
		if hasCell(rows[i], "学校") &&
			(hasCell(rows[i], "省份") || hasCell(rows[i], "办学性质") || hasCell(rows[i], "双一流")) {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return
	}
	h := rows[headerIdx]
	cName := findCol(h, "学校", "院校名称")
	if cName < 0 {
		return
	}
	cProv := findCol(h, "省份")
	cCity := findCol(h, "城市")
	cOwner := findCol(h, "办学性质")
	cKind := findCol(h, "学校类别")
	c985 := findCol(h, "_985", "985")
	c211 := findCol(h, "_211", "211")
	cSyl := findCol(h, "双一流")
	for _, r := range rows[headerIdx+1:] {
		name := strings.TrimSpace(cell(r, cName))
		if name == "" {
			continue
		}
		v := SchoolMeta{
			Province:      strings.TrimSpace(cell(r, cProv)),
			City:          strings.TrimSpace(cell(r, cCity)),
			Ownership:     strings.TrimSpace(cell(r, cOwner)),
			Kind:          strings.TrimSpace(cell(r, cKind)),
			Is985:         cell(r, c985) == "是",
			Is211:         cell(r, c211) == "是",
			IsShuangYiLiu: cell(r, cSyl) == "是",
		}
		if !v.any() {
			continue
		}
		si.merge(si.byNorm, normName(name), v)
		si.merge(si.byBase, baseName(name), v)
	}
}

// Lookup 查院校属性：精确名优先，再退到去括号基名。
func (si *SchoolMetaIndex) Lookup(name string) (SchoolMeta, bool) {
	if v, ok := si.byNorm[normName(name)]; ok {
		return *v, true
	}
	if v, ok := si.byBase[baseName(name)]; ok {
		return *v, true
	}
	return SchoolMeta{}, false
}

// Len 返回已收录（归一化名）院校数，用于日志。
func (si *SchoolMetaIndex) Len() int { return len(si.byNorm) }

// LoadSchoolMeta 打开多个 xlsx，解析首个 sheet 的院校属性列并合并。打不开/无相关列的静默跳过。
func LoadSchoolMeta(paths []string) *SchoolMetaIndex {
	si := NewSchoolMetaIndex()
	for _, p := range paths {
		f, err := excelize.OpenFile(p)
		if err != nil {
			continue
		}
		if sheets := f.GetSheetList(); len(sheets) > 0 {
			if rows, err := f.GetRows(sheets[0]); err == nil {
				si.AddRows(rows)
			}
		}
		_ = f.Close()
	}
	return si
}
