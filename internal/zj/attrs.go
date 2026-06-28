package zj

import (
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/xuri/excelize/v2"
)

// SchoolAttr 是院校级属性（按院校代码挂接），用于位次定位结果过滤与院校页展示。
// 浙江源数据已自带这些列，无需像黑龙江那样靠校名挂接第三方表。
type SchoolAttr struct {
	Province      string // 所在省
	City          string // 城市
	CityTier      string // 城市层级（来自「城市水平标签」）
	Ownership     string // 公办 / 民办
	Kind          string // 学校类别（类型）
	Is985         bool
	Is211         bool
	IsShuangYiLiu bool
}

func (a SchoolAttr) any() bool {
	return a.Province != "" || a.City != "" || a.CityTier != "" || a.Ownership != "" ||
		a.Kind != "" || a.Is985 || a.Is211 || a.IsShuangYiLiu
}

// Levels 返回该校的层次标签数组（985/211/双一流），仅含为真者。
func (a SchoolAttr) Levels() []string {
	var lv []string
	if a.Is985 {
		lv = append(lv, "985")
	}
	if a.Is211 {
		lv = append(lv, "211")
	}
	if a.IsShuangYiLiu {
		lv = append(lv, "双一流")
	}
	return lv
}

// AttrIndex 按院校代码查院校属性；多源合并：字符串取首个非空、布尔按位 OR。
type AttrIndex struct {
	byCode map[string]*SchoolAttr
}

func NewAttrIndex() *AttrIndex { return &AttrIndex{byCode: map[string]*SchoolAttr{}} }

func (ai *AttrIndex) Len() int { return len(ai.byCode) }

func (ai *AttrIndex) Lookup(code string) (SchoolAttr, bool) {
	if v, ok := ai.byCode[core.NormSchoolCode(code)]; ok {
		return *v, true
	}
	return SchoolAttr{}, false
}

// All 返回全部 院校代码 → 属性（已归一化代码键），供入库 staging 落 school_attr。
func (ai *AttrIndex) All() map[string]SchoolAttr {
	out := make(map[string]SchoolAttr, len(ai.byCode))
	for k, v := range ai.byCode {
		out[k] = *v
	}
	return out
}

func (ai *AttrIndex) merge(code string, v SchoolAttr) {
	code = core.NormSchoolCode(code)
	if code == "" || !v.any() {
		return
	}
	cur := ai.byCode[code]
	if cur == nil {
		cur = &SchoolAttr{}
		ai.byCode[code] = cur
	}
	if cur.Province == "" {
		cur.Province = v.Province
	}
	if cur.City == "" {
		cur.City = v.City
	}
	if cur.CityTier == "" {
		cur.CityTier = v.CityTier
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

// cityTierOf 从「城市水平标签」（如 "新一线城市/省会城市"）取层级 "新一线"。
func cityTierOf(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	first := strings.Split(label, "/")[0]
	return strings.TrimSuffix(strings.TrimSpace(first), "城市")
}

// tagsOf 从「院校标签」（如 "985/211/双一流/国重点/保研资格"）解析层次。
func tagsOf(tags string) (is985, is211, syl bool) {
	return strings.Contains(tags, "985"), strings.Contains(tags, "211"), strings.Contains(tags, "双一流")
}

// addScoreRows 从「专业录取分数」表的内联院校列补充属性（覆盖全部院校）。
func (ai *AttrIndex) addScoreRows(rows [][]string) {
	headerIdx := -1
	for i, r := range rows {
		if core.HasCell(r, "院校代码") && core.HasCell(r, "学校所在") {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return
	}
	h := rows[headerIdx]
	cCode := core.FindCol(h, "院校代码")
	cProv := core.FindCol(h, "学校所在")
	cOwner := core.FindCol(h, "学校性质")
	c985 := core.FindCol(h, "是否985")
	c211 := core.FindCol(h, "是否211")
	for _, r := range rows[headerIdx+1:] {
		ai.merge(core.Cell(r, cCode), SchoolAttr{
			Province:  strings.TrimSpace(core.Cell(r, cProv)),
			Ownership: strings.TrimSpace(core.Cell(r, cOwner)),
			Is985:     strings.TrimSpace(core.Cell(r, c985)) == "是",
			Is211:     strings.TrimSpace(core.Cell(r, c211)) == "是",
		})
	}
}

// addLianRows 从「一表联动」表补充更全属性（城市/城市层级/类型/双一流）。表头在前几行内（含超表头）。
func (ai *AttrIndex) addLianRows(rows [][]string) {
	headerIdx := -1
	for i := 0; i < len(rows) && i < 6; i++ {
		if core.HasCell(rows[i], "院校代码") && core.HasCell(rows[i], "城市水平标签") {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return
	}
	h := rows[headerIdx]
	cCode := core.FindCol(h, "院校代码")
	cProv := core.FindCol(h, "所在省")
	cCity := core.FindCol(h, "城市")
	cTier := core.FindCol(h, "城市水平标签")
	cKind := core.FindCol(h, "类型")
	cTags := core.FindCol(h, "院校标签")
	for _, r := range rows[headerIdx+1:] {
		is985, is211, syl := tagsOf(core.Cell(r, cTags))
		prov := strings.TrimSpace(core.Cell(r, cProv))
		ai.merge(core.Cell(r, cCode), SchoolAttr{
			Province:      prov,
			City:          core.NormCity(prov, core.Cell(r, cCity)),
			CityTier:      cityTierOf(core.Cell(r, cTier)),
			Kind:          strings.TrimSpace(core.Cell(r, cKind)),
			Is985:         is985,
			Is211:         is211,
			IsShuangYiLiu: syl,
		})
	}
}

// LoadAttrs 合并「专业录取分数」（内联列，全覆盖）与「一表联动」（更全属性）→ 院校属性索引。
func LoadAttrs(scorePaths []string, lianPath string) *AttrIndex {
	ai := NewAttrIndex()
	// 先读一表联动（更全），再读分数表（兜底补 省/性质/985/211）。
	if lianPath != "" {
		if rows := readFirstSheet(lianPath); rows != nil {
			ai.addLianRows(rows)
		}
	}
	for _, p := range scorePaths {
		if rows := readFirstSheet(p); rows != nil {
			ai.addScoreRows(rows)
		}
	}
	return ai
}

func readFirstSheet(path string) [][]string {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil
	}
	return rows
}
