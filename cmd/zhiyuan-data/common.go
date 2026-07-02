package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/hlj"
	"github.com/sunfmin/zhiyuanwiki/internal/zj"
)

// schoolDetail 是写给 Astro 的每校详情：院校 + 其全部院校×专业叶子 + 2026 报考视图。
// 报考视图按省份的填报模型二选一：黑龙江=院校专业组(Groups2026)；浙江=院校×专业(Plan2026)。
type schoolDetail struct {
	core.School
	Leaves     []core.MajorLeaf `json:"leaves"`
	Groups2026 []hlj.Group2026  `json:"groups2026,omitempty"`
	Plan2026   []zj.PlanMajor   `json:"plan2026,omitempty"`
}

// trackRange 是某科类在该校最新有数据年份的录取线区间。
// MaxScore↔MinRank 是最难专业（分最高、位次最靠前）；MinScore↔MaxRank 是最易专业。
type trackRange struct {
	Year     int `json:"year"`
	MinScore int `json:"minScore"`
	MaxScore int `json:"maxScore"`
	MinRank  int `json:"minRank"`
	MaxRank  int `json:"maxRank"`
}

// schoolIndexEntry 是 schools.json 索引项。Ranges 按科类名给出各自录取线区间
// （黑龙江=物理/历史；浙江=综合），列表可按任一科类的 MaxScore 排序。
type schoolIndexEntry struct {
	Code      string                 `json:"code"`
	Name      string                 `json:"name"`
	LeafCount int                    `json:"leafCount"`
	Ranges    map[string]*trackRange `json:"ranges"`

	Is985         bool `json:"is985,omitempty"`
	Is211         bool `json:"is211,omitempty"`
	IsShuangYiLiu bool `json:"isShuangYiLiu,omitempty"`
}

// schoolMetaOut 是 public/data/<slug>/school-meta.json 的一条（按院校代码建键），承载位次定位
// 结果过滤用的院校级属性。紧凑键、空值省略；客户端一次性 fetch、按 sc 挂接。见 ADR-0008。
type schoolMetaOut struct {
	Province string   `json:"p,omitempty"`  // 省份（院校所在地）
	City     string   `json:"c,omitempty"`  // 城市
	CityTier string   `json:"ct,omitempty"` // 城市层级
	Owner    string   `json:"o,omitempty"`  // 办学性质
	Kind     string   `json:"k,omitempty"`  // 学校类别
	Levels   []string `json:"lv,omitempty"` // 层次：["985","211","双一流"] 中为真者
}

// rangeForTrack 汇总某科类在该校最新有数据年份的录取线区间；该科类无数据返回 nil。
func rangeForTrack(leaves []core.MajorLeaf, track string) *trackRange {
	year := 0
	for _, lf := range leaves {
		for _, ys := range lf.Years {
			if ys.Track == track && ys.MinScore > 0 && ys.Year > year {
				year = ys.Year
			}
		}
	}
	if year == 0 {
		return nil
	}
	r := &trackRange{Year: year}
	for _, lf := range leaves {
		for _, ys := range lf.Years {
			if ys.Track != track || ys.Year != year || ys.MinScore <= 0 {
				continue
			}
			if r.MinScore == 0 || ys.MinScore < r.MinScore {
				r.MinScore = ys.MinScore
			}
			if ys.MinScore > r.MaxScore {
				r.MaxScore = ys.MinScore
			}
			if ys.MinRank > 0 {
				if r.MinRank == 0 || ys.MinRank < r.MinRank {
					r.MinRank = ys.MinRank
				}
				if ys.MinRank > r.MaxRank {
					r.MaxRank = ys.MinRank
				}
			}
		}
	}
	return r
}

// schoolKey 是院校实体在投影层的**主键**——用作 details/meta/levels map 键、schools.json 索引 Code、
// 以及每校详情文件名 slug。当前 == 院校代号；ADR-0021 将其改为归一化校名派生（报考实体粒度）。
// 全站对「院校实体键」的读写收口到此处，便于 #37 单点替换而不散落改 s.Code。
func schoolKey(s core.School) string { return s.Code }

// leafGroupKey 是把院校×专业叶子归拢到其院校实体的键，须与 schoolKey **同源**。当前 == 叶子院校代号。
func leafGroupKey(lf core.MajorLeaf) string { return lf.SchoolCode }

// schoolBundle 是某省份院校数据的中间产物，由各省 builder 产出、由 emitSchoolData 落盘。
type schoolBundle struct {
	schools []core.School
	leaves  []core.MajorLeaf
	details map[string]schoolDetail  // 院校代码 → 详情（叶子 + 2026 视图）
	meta    map[string]schoolMetaOut // 院校代码 → 过滤属性
	levels  map[string][3]bool       // 院校代码 → {is985,is211,双一流}（写入索引）
}

// emitSchoolData 把 bundle 落盘：schools.json 索引、school-meta.json 过滤属性、schools/{code}.json 详情。
func emitSchoolData(p province, b schoolBundle, outDir, pubDir string) {
	byCode := map[string][]core.MajorLeaf{}
	for _, lf := range b.leaves {
		byCode[leafGroupKey(lf)] = append(byCode[leafGroupKey(lf)], lf)
	}

	index := make([]schoolIndexEntry, 0, len(b.schools))
	n985, n211, nSyl := 0, 0, 0
	for _, s := range b.schools {
		lvs := byCode[schoolKey(s)]
		ranges := map[string]*trackRange{}
		for _, tr := range p.tracks {
			if r := rangeForTrack(lvs, tr); r != nil {
				ranges[tr] = r
			}
		}
		e := schoolIndexEntry{Code: schoolKey(s), Name: s.Name, LeafCount: len(lvs), Ranges: ranges}
		if lv, ok := b.levels[schoolKey(s)]; ok {
			e.Is985, e.Is211, e.IsShuangYiLiu = lv[0], lv[1], lv[2]
			if lv[0] {
				n985++
			}
			if lv[1] {
				n211++
			}
			if lv[2] {
				nSyl++
			}
		}
		index = append(index, e)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatal(err)
	}
	writeJSON(filepath.Join(outDir, "schools.json"), index)
	fmt.Printf("  院校层次：985=%d 211=%d 双一流=%d\n", n985, n211, nSyl)

	if err := os.MkdirAll(pubDir, 0o755); err != nil {
		fatal(err)
	}
	writeJSON(filepath.Join(pubDir, "school-meta.json"), b.meta)
	fmt.Printf("  院校过滤属性：%d 所 → %s\n", len(b.meta), filepath.Join(pubDir, "school-meta.json"))

	detailDir := filepath.Join(outDir, "schools")
	if err := os.MkdirAll(detailDir, 0o755); err != nil {
		fatal(err)
	}
	for _, s := range b.schools {
		d := b.details[schoolKey(s)]
		writeJSON(filepath.Join(detailDir, schoolKey(s)+".json"), d)
	}
	fmt.Printf("✓ %s · 院校 %d 所 · 院校×专业叶子 %d 个 → %s\n",
		p.name, len(b.schools), len(b.leaves), outDir)
}

// defaultSrc 是浙江/黑龙江专用 import 的源根（其下有 09、浙江…、24-万师兄…、黑龙江2026物理类一分一段表.xlsx）。
// 原为万师兄树 ~/Developments/zhiyuan/官方数据，数据归整后统一到 ~/Downloads/高考志愿（与各省份同根，见 CLAUDE.md 数据来源审计）。
func defaultSrc() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Downloads", "高考志愿")
}

// logSrc 打印某次入库实际选中的源 xlsx（数据溯源/审计用）：因源树里常有多版本同名文件、
// 且按「子树内含关键子串、体积最大」选文件，打印真正用到的那份可随时核对是否为期望版本。
func logSrc(role, path string) {
	fmt.Printf("  📄 %s ← %s\n", role, path)
}

func writeJSON(path string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "错误:", err)
	os.Exit(1)
}
