package hlj

import (
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

// SchoolMeta 是院校级属性（与年份无关），用于位次定位结果过滤。来自万师兄旧格式表，
// 按校名挂接（含 985/211/双一流，与 tags.go 同源）。城市层级由 core.CityTier 另算。见 ADR-0008。
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
		if core.HasCell(rows[i], "学校") &&
			(core.HasCell(rows[i], "省份") || core.HasCell(rows[i], "办学性质") || core.HasCell(rows[i], "双一流")) {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return
	}
	h := rows[headerIdx]
	cName := core.FindCol(h, "学校", "院校名称")
	if cName < 0 {
		return
	}
	cProv := core.FindCol(h, "省份")
	cCity := core.FindCol(h, "城市")
	cOwner := core.FindCol(h, "办学性质")
	cKind := core.FindCol(h, "学校类别")
	c985 := core.FindCol(h, "_985", "985")
	c211 := core.FindCol(h, "_211", "211")
	cSyl := core.FindCol(h, "双一流")
	for _, r := range rows[headerIdx+1:] {
		name := strings.TrimSpace(core.Cell(r, cName))
		if name == "" {
			continue
		}
		prov := strings.TrimSpace(core.Cell(r, cProv))
		v := SchoolMeta{
			Province:      prov,
			City:          core.NormCity(prov, core.Cell(r, cCity)),
			Ownership:     strings.TrimSpace(core.Cell(r, cOwner)),
			Kind:          strings.TrimSpace(core.Cell(r, cKind)),
			Is985:         core.Cell(r, c985) == "是",
			Is211:         core.Cell(r, c211) == "是",
			IsShuangYiLiu: core.Cell(r, cSyl) == "是",
		}
		if !v.any() {
			continue
		}
		si.merge(si.byNorm, core.NormName(name), v)
		si.merge(si.byBase, core.BaseName(name), v)
	}
}

// Lookup 查院校属性：精确名优先，再退到去括号基名。
func (si *SchoolMetaIndex) Lookup(name string) (SchoolMeta, bool) {
	if v, ok := si.byNorm[core.NormName(name)]; ok {
		return *v, true
	}
	if v, ok := si.byBase[core.BaseName(name)]; ok {
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
