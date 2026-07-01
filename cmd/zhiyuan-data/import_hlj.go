package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/hlj"
	"github.com/sunfmin/zhiyuanwiki/internal/store"
)

// importHLJ 把黑龙江入库 staging。黑龙江源是异构的、且分布在两棵树，故不能套用 importProvince
// 的「各省份/<省>」统一 glob（见 ADR-0014/issue #19）：
//   - 专业录取分数（2023-2025 物理/历史·含位次）与 2026 招生计划：万师兄树（defaultSrc），与旧
//     buildHLJBundle 同源，保证 院校/叶子/组视图 不回归。
//   - 一分一段：各省份树（gefenSrc），含 2024/2025 物理+历史（2025 还带本科线），点亮高考人数/本科线；
//     再并入万师兄 2026 物理（保留旧 fenduan wuli-2026，2026 是当届考生的换算基准）。
//
// 全国院校属性表已由 importCmd 的 importNational 先行刷新（各省份树），下游 buildDBBundle 据此挂接。
func importHLJ(db *store.DB, gefenSrc string, p province) {
	wan := defaultSrc() // 黑龙江分数/计划源根（24-万师兄…，原万师兄树，已归整到 高考志愿/）

	// 专业录取分数：万师兄 2023/2024/2025（逐年文件，与旧 buildHLJBundle 同源同解析）。
	scoreDir := filepath.Join(wan,
		"24-万师兄-黑龙江2026年高考志愿填报大数据",
		"04-万师兄-黑龙江高考-专业录取分数线-2020-2025")
	var scores []core.MajorScoreRow
	for _, y := range []int{2023, 2024, 2025} { // 新科类（物理/历史 + 位次），见 ADR-0007
		path := findHLJScoreFile(scoreDir, y)
		if path == "" {
			fmt.Fprintf(os.Stderr, "⚠ 黑龙江 %d 专业分数线：未找到文件，跳过\n", y)
			continue
		}
		logSrc(fmt.Sprintf("录取分数·%d", y), path)
		rows, err := hlj.ParseMajorScoresXLSX(path)
		if err != nil {
			fatal(err)
		}
		scores = append(scores, rows...)
	}
	if len(scores) == 0 {
		fatal(fmt.Errorf("黑龙江：未解析到任何专业分数线行"))
	}
	if err := db.ReplaceScores(p.slug, scores); err != nil {
		fatal(err)
	}
	fmt.Printf("  专业录取分数：%d 行（物理/历史·本科·含位次）→ major_score\n", len(scores))

	// 招生计划：万师兄 2026（组视图是单年视图，取 2026）。
	planPath := filepath.Join(wan,
		"24-万师兄-黑龙江2026年高考志愿填报大数据",
		"01-万师兄-黑龙江高考-招生计划-2020-2026", "黑龙江_招生计划_2026.xlsx")
	logSrc("招生计划·2026", planPath)
	if plan, err := hlj.ParsePlanXLSX(planPath); err != nil {
		fmt.Fprintf(os.Stderr, "⚠ 黑龙江 2026 招生计划解析失败（%v），跳过（组视图将为空）\n", err)
	} else {
		if err := db.ReplacePlan(p.slug, plan); err != nil {
			fatal(err)
		}
		fmt.Printf("  招生计划：%d 行（2026·物理/历史·本科批）→ plan\n", len(plan))
	}

	// 一分一段：各省份树 2024/2025 物理+历史 + 万师兄 2026 物理。按 年×科类 去重（findFiles 按
	// 体积降序，保留首个最全副本；2025 合表在两处各有一份）。理科/文科逐科类表由解析器返回 nil 跳过。
	var allYfd []*core.YiFenYiDuan
	seenYT := map[core.YearTrack]bool{}
	add := func(yds []*core.YiFenYiDuan) {
		for _, y := range yds {
			yt := core.YearTrack{Year: y.Year, Track: y.Track}
			if seenYT[yt] {
				continue
			}
			seenYT[yt] = true
			allYfd = append(allYfd, y)
		}
	}
	hljRoot := filepath.Join(gefenSrc, provDirName[p.slug])
	for _, yf := range findFiles(hljRoot, []string{"一分一段表"}, []string{"艺术", "艺考"}) {
		year := yearFromName(filepath.Base(yf))
		if year == 0 {
			continue
		}
		yds, err := hlj.ParseYiFenYiDuan(yf, p.name, year)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ 一分一段 %s 解析失败：%v\n", filepath.Base(yf), err)
			continue
		}
		logSrc(fmt.Sprintf("一分一段·%d", year), yf)
		add(yds)
	}
	// 万师兄 2026 物理（各省份树暂无 2026；保留旧 fenduan wuli-2026）。
	wan2026 := filepath.Join(wan, "黑龙江2026物理类一分一段表.xlsx")
	if _, err := os.Stat(wan2026); err == nil {
		if yds, err := hlj.ParseYiFenYiDuan(wan2026, p.name, 2026); err == nil {
			logSrc("一分一段·2026", wan2026)
			add(yds)
		}
	}
	if err := db.ReplaceYiFenYiDuan(p.slug, allYfd); err != nil {
		fatal(err)
	}
	fmt.Printf("  一分一段：%d 个(年×科类) → yifenyiduan\n", len(allYfd))
	fmt.Printf("✓ %s入库完成 → %s\n", p.name, "out/zhiyuan.db")
}

// findHLJScoreFile 在万师兄目录里找某年的专业分数线文件（连字符或下划线命名）。
func findHLJScoreFile(dir string, year int) string {
	for _, name := range []string{
		fmt.Sprintf("黑龙江-专业分数线-%d.xlsx", year),
		fmt.Sprintf("黑龙江_专业分数线_%d.xlsx", year),
	} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
