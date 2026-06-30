package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
	"github.com/sunfmin/zhiyuanwiki/internal/zj"
)

type majorSchool struct {
	SchoolCode string `json:"sc"`
	SchoolName string `json:"sn"`
	MajorName  string `json:"mn,omitempty"` // 叶子全名（含方向后缀，用于专业页内消歧）
	MajorKey   string `json:"mk"`           // 叶子键（含方向），#z 锚点用
	MinRank    int    `json:"minRank"`
	MinScore   int    `json:"minScore,omitempty"` // 最近年最低分（只有分数省=西藏用它横向比较；位次省省略）
	Year       int    `json:"year"`
	Track      string `json:"track"`
}

type majorDetail struct {
	Key     string        `json:"key"`
	Name    string        `json:"name"`
	Schools []majorSchool `json:"schools"`
}

type majorIndexEntry struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	SchoolCount int    `json:"schoolCount"`
}

// zhuanyeCmd 按归一化专业名（majorKey）跨校聚合院校×专业叶子 → 专业索引与专业详情（按省份分目录）。
// 依赖先跑过 `yuanxiao -prov <slug>`。科类无关，黑龙江/浙江通用。
func zhuanyeCmd(args []string) {
	fs := flag.NewFlagSet("zhuanye", flag.ExitOnError)
	in := fs.String("in", filepath.Join("src", "data"), "src/data 目录（其下按省份 slug 分目录）")
	out := fs.String("out", filepath.Join("src", "data"), "JSON 输出目录（其下按省份 slug 分目录）")
	provSlug := fs.String("prov", "hlj", "省份 slug：hlj / zj")
	_ = fs.Parse(args)
	p := mustProv(*provSlug)

	schoolsDir := filepath.Join(srcDir(*in, p), "schools")
	files, err := filepath.Glob(filepath.Join(schoolsDir, "*.json"))
	if err != nil || len(files) == 0 {
		fatal(fmt.Errorf("未找到 %s 的 school 详情（先跑 yuanxiao -prov %s）：%v", p.name, p.slug, err))
	}

	details := map[string]*majorDetail{}
	for _, fp := range files {
		b, err := os.ReadFile(fp)
		if err != nil {
			fatal(err)
		}
		var d schoolDetail
		if err := json.Unmarshal(b, &d); err != nil {
			fatal(fmt.Errorf("%s: %w", fp, err))
		}
		for _, lf := range d.Leaves {
			if len(lf.Years) == 0 {
				continue
			}
			latest := lf.Years[0]
			for _, y := range lf.Years {
				if y.Year >= latest.Year {
					latest = y
				}
			}
			// 专业索引按「基名」跨校归并（浙江大类方向折叠回基名）；锚点仍用叶子键。
			baseName := lf.MajorName
			if p.slug == "zj" {
				baseName = zj.BaseMajorName(lf.MajorName)
			}
			baseKey := core.MajorKey(baseName)
			md := details[baseKey]
			if md == nil {
				md = &majorDetail{Key: baseKey, Name: baseName}
				details[baseKey] = md
			}
			md.Schools = append(md.Schools, majorSchool{
				SchoolCode: d.Code, SchoolName: d.Name,
				MajorName: lf.MajorName, MajorKey: lf.MajorKey,
				MinRank: latest.MinRank, MinScore: latest.MinScore, Year: latest.Year, Track: latest.Track,
			})
		}
	}

	outBase := srcDir(*out, p)
	detailDir := filepath.Join(outBase, "majors")
	if err := os.MkdirAll(detailDir, 0o755); err != nil {
		fatal(err)
	}
	index := make([]majorIndexEntry, 0, len(details))
	for _, md := range details {
		// 有位次省按最低位次升序（最难在前）；只有分数省（西藏，MinRank 全 0）按最低分降序。
		sort.Slice(md.Schools, func(i, j int) bool {
			a, b := md.Schools[i], md.Schools[j]
			if a.MinRank > 0 || b.MinRank > 0 {
				return a.MinRank < b.MinRank
			}
			return a.MinScore > b.MinScore
		})
		writeJSON(filepath.Join(detailDir, md.Key+".json"), md)
		index = append(index, majorIndexEntry{Key: md.Key, Name: md.Name, SchoolCount: len(md.Schools)})
	}
	sort.Slice(index, func(i, j int) bool {
		if index[i].SchoolCount != index[j].SchoolCount {
			return index[i].SchoolCount > index[j].SchoolCount
		}
		return index[i].Name < index[j].Name
	})
	writeJSON(filepath.Join(outBase, "majors.json"), index)

	fmt.Printf("✓ %s · 专业 %d 个（跨校聚合）→ %s\n", p.name, len(index), outBase)
}
