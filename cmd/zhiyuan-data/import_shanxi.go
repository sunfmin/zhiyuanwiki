package main

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/group3p12"
	"github.com/sunfmin/zhiyuanwiki/internal/shanxi"
	"github.com/sunfmin/zhiyuanwiki/internal/store"
)

// shanxiYear 是山西首届 3+1+2 新高考年份。录取分数文件名标 2024，但其科类是物理/历史、含院校
// 专业组（仅 2025 起存在；2024 及以前山西是老文理理科/文科），实为 2025 数据，统一按 2025 入库。
const shanxiYear = 2025

// importShanxi 把山西（2025 新高考·院校专业组）入库 staging。山西源表与统一 group 省不同形，
// 故走专用 import（同 importHLJ/importZJ，见 ADR-0014）：
//   - 招生计划列名规范（院校代码/专业名称/计划人数·物理/历史），复用 group3p12.ParsePlan；
//     源无年份列，统一置 2025。
//   - 专业录取分数表无院校代码列（internal/shanxi 专属解析），按校名从招生计划回填规范代码——
//     院校页/叶子/组视图都按院校代码挂接，缺码行会被 AggregateLeaves 丢弃，故必须回填。
//   - 一分一段科类编码在文件名（无科类列），逐文件按文件名定科类，用 core.ParseYiFenYiDuanXLSX 解析。
//
// 全国院校属性表已由 importCmd 的 importNational 先行刷新（按校名挂接），下游 buildDBBundle 据此挂属性。
func importShanxi(db *store.DB, gefenSrc string, p province) {
	root := filepath.Join(gefenSrc, provDirName[p.slug])

	// 招生计划：2025·物理/历史·院校专业组。无「年份」列 → 统一置 2025（与录取分数同年，归一安全）。
	planPath := findFile(root, []string{"山西2025招生计划"}, []string{"艺术", "艺考"})
	if planPath == "" {
		fatal(fmt.Errorf("山西：未找到 2025 招生计划 xlsx（在 %s 下）", root))
	}
	logSrc("招生计划", planPath)
	plan, err := group3p12.ParsePlan(planPath)
	if err != nil {
		fatal(err)
	}
	for i := range plan {
		plan[i].Year = shanxiYear
	}
	if err := db.ReplacePlan(p.slug, plan); err != nil {
		fatal(err)
	}
	fmt.Printf("  招生计划：%d 行（%d·物理/历史·院校专业组）→ plan\n", len(plan), shanxiYear)

	// 校名→院校代码：计划侧有规范代码，分数表无代码，按校名回填。计划给同名校加了尾部「(城市)」
	// （中国人民大学(北京)、中国石油大学(华东)(青岛)），分数表用裸名/校区名，故除精确匹配外再按
	// 「去尾部城市括号后唯一」回填——多校区歧义（中国矿业大学 北京/徐州）不强配，留给合成码。
	resolveCode := shanxiCodeResolver(plan)

	// 专业录取分数：2025·物理/历史·含位次。回填代码；计划里查不到规范代码的院校用合成码（叶子仍在，
	// 仅无 2026 报考视图），不丢数据。
	scorePath := findFile(root, []string{"专业录取分数线"}, []string{"艺术", "艺考"})
	if scorePath == "" {
		fatal(fmt.Errorf("山西：未找到专业录取分数线 xlsx（在 %s 下）", root))
	}
	logSrc("录取分数", scorePath)
	rawScores, err := shanxi.ParseScores(scorePath, shanxiYear)
	if err != nil {
		fatal(err)
	}
	var scores []core.MajorScoreRow
	matchedSchools, synthSchools := map[string]bool{}, map[string]bool{}
	for _, r := range rawScores {
		if code := resolveCode(r.SchoolName); code != "" {
			r.SchoolCode = code
			matchedSchools[r.SchoolName] = true
		} else {
			r.SchoolCode = shanxiSyntheticCode(r.SchoolName)
			synthSchools[r.SchoolName] = true
		}
		scores = append(scores, r)
	}
	if len(scores) == 0 {
		fatal(fmt.Errorf("山西：录取分数解析为空"))
	}
	if err := db.ReplaceScores(p.slug, scores); err != nil {
		fatal(err)
	}
	fmt.Printf("  专业录取分数：%d 行（物理/历史·本科·含位次）· 院校代码回填命中 %d 校",
		len(scores), len(matchedSchools))
	if n := len(synthSchools); n > 0 {
		fmt.Printf("，%d 校计划无规范代码用合成码（无 2026 报考视图，例：%s）", n, joinSomeNames(synthSchools, 5))
	}
	fmt.Println(" → major_score")

	// 一分一段：2025 物理/历史（科目组合）。科类在文件名，逐文件定科类。
	var allYfd []*core.YiFenYiDuan
	for _, yf := range findFiles(root, []string{"一分一段", "2025"}, []string{"艺术", "艺考"}) {
		track := shanxiTrackFromName(filepath.Base(yf))
		if track == "" {
			continue
		}
		logSrc(fmt.Sprintf("一分一段·%d·%s", shanxiYear, track), yf)
		y, err := core.ParseYiFenYiDuanXLSX(yf, p.name, track, shanxiYear)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ 一分一段 %s 解析失败：%v\n", filepath.Base(yf), err)
			continue
		}
		allYfd = append(allYfd, y)
	}
	// 2026 起：一分一段由官方图片版表 OCR 重塑成万师兄格式（科类列 · 物理/历史同表），走通用
	// group3p12 解析（自带 2026 年份），与 2025 老单科类文件并存入库。见 CLAUDE.md 下载数据约定。
	for _, yf := range findFiles(root, []string{"一分一段表", "2026"}, []string{"艺术", "艺考"}) {
		yds, err := group3p12.ParseYiFenYiDuan(yf, p.name, 2026)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ 一分一段(2026) %s 解析失败：%v\n", filepath.Base(yf), err)
			continue
		}
		allYfd = append(allYfd, yds...)
	}
	if err := db.ReplaceYiFenYiDuan(p.slug, allYfd); err != nil {
		fatal(err)
	}
	fmt.Printf("  一分一段：%d 个(年×科类) → yifenyiduan\n", len(allYfd))
	fmt.Printf("✓ %s入库完成 → %s\n", p.name, "out/zhiyuan.db")
}

// shanxiCodeResolver 据招生计划建「校名→院校代码」解析器：先精确（归一校名）匹配，再按「去尾部
// (城市) 括号后唯一」匹配。后者吸收计划给同名校加的城市后缀（中国人民大学(北京)↔中国人民大学、
// 中国石油大学(华东)(青岛)↔中国石油大学(华东)）；多校区落到同一基名时（中国矿业大学 北京/徐州）
// 视为歧义、不返回，避免把分数错挂到另一校区。
func shanxiCodeResolver(plan []core.PlanRow) func(name string) string {
	exact := map[string]string{}
	base := map[string]map[string]bool{} // 去尾部城市括号后的基名 → 代码集合（判唯一）
	for _, r := range plan {
		if r.SchoolCode == "" {
			continue
		}
		nm := core.NormName(r.SchoolName)
		if exact[nm] == "" {
			exact[nm] = r.SchoolCode
		}
		if b := stripTrailingCity(nm); b != nm {
			if base[b] == nil {
				base[b] = map[string]bool{}
			}
			base[b][r.SchoolCode] = true
		}
	}
	return func(name string) string {
		nm := core.NormName(name)
		if c := exact[nm]; c != "" {
			return c
		}
		if codes := base[nm]; len(codes) == 1 {
			for c := range codes {
				return c
			}
		}
		return ""
	}
}

// stripTrailingCity 去掉院校名末尾的「(城市)」括号（仅当括号内是纯地名、不含 校区/分校/学院 时），
// 否则原样返回。入参须已 NormName（括号为半角）。
func stripTrailingCity(nm string) string {
	if !strings.HasSuffix(nm, ")") {
		return nm
	}
	i := strings.LastIndex(nm, "(")
	if i <= 0 {
		return nm
	}
	inner := nm[i+1 : len(nm)-1]
	if strings.Contains(inner, "校区") || strings.Contains(inner, "分校") || strings.Contains(inner, "学院") {
		return nm
	}
	return nm[:i]
}

// shanxiSyntheticCode 为计划里查不到规范代码的院校生成稳定合成码（S+7 位十六进制，不与 4 位数字
// 院校代码冲突）。这类院校的叶子仍按此码成页（含历年录取走势），只是无 2026 报考视图。
func shanxiSyntheticCode(name string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(core.NormName(name)))
	return fmt.Sprintf("S%07x", h.Sum32()&0x0fffffff)
}

// shanxiTrackFromName 从山西一分一段文件名取科类（「物理科目组合」→物理，「历史科目组合」→历史）。
func shanxiTrackFromName(name string) string {
	switch {
	case strings.Contains(name, "物理"):
		return "物理"
	case strings.Contains(name, "历史"):
		return "历史"
	}
	return ""
}

// joinSomeNames 取集合里前 n 个名字（排序后）拼成逗号串，供告警示例用。
func joinSomeNames(set map[string]bool, n int) string {
	names := make([]string, 0, len(set))
	for s := range set {
		names = append(names, s)
	}
	sort.Strings(names)
	if len(names) > n {
		names = names[:n]
	}
	return strings.Join(names, "、")
}
