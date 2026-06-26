package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type majorSchool struct {
	SchoolCode string `json:"sc"`
	SchoolName string `json:"sn"`
	MajorKey   string `json:"mk"`
	MinRank    int    `json:"minRank"`
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

// zhuanyeCmd 按归一化专业名（majorKey）跨校聚合院校×专业叶子 → 专业索引与专业详情。
// 依赖先跑过 `yuanxiao`。
func zhuanyeCmd(args []string) {
	fs := flag.NewFlagSet("zhuanye", flag.ExitOnError)
	in := fs.String("in", filepath.Join("src", "data", "schools"), "school 详情 JSON 目录")
	out := fs.String("out", filepath.Join("src", "data"), "JSON 输出目录")
	_ = fs.Parse(args)

	files, err := filepath.Glob(filepath.Join(*in, "*.json"))
	if err != nil || len(files) == 0 {
		fatal(fmt.Errorf("未找到 school 详情（先跑 yuanxiao）：%v", err))
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
			// 最近年份数据点
			latest := lf.Years[0]
			for _, y := range lf.Years {
				if y.Year >= latest.Year {
					latest = y
				}
			}
			md := details[lf.MajorKey]
			if md == nil {
				md = &majorDetail{Key: lf.MajorKey, Name: lf.MajorName}
				details[lf.MajorKey] = md
			}
			md.Schools = append(md.Schools, majorSchool{
				SchoolCode: d.Code, SchoolName: d.Name, MajorKey: lf.MajorKey,
				MinRank: latest.MinRank, Year: latest.Year, Track: latest.Track,
			})
		}
	}

	detailDir := filepath.Join(*out, "majors")
	if err := os.MkdirAll(detailDir, 0o755); err != nil {
		fatal(err)
	}
	index := make([]majorIndexEntry, 0, len(details))
	for _, md := range details {
		sort.Slice(md.Schools, func(i, j int) bool { return md.Schools[i].MinRank < md.Schools[j].MinRank })
		writeJSON(filepath.Join(detailDir, md.Key+".json"), md)
		index = append(index, majorIndexEntry{Key: md.Key, Name: md.Name, SchoolCount: len(md.Schools)})
	}
	sort.Slice(index, func(i, j int) bool {
		if index[i].SchoolCount != index[j].SchoolCount {
			return index[i].SchoolCount > index[j].SchoolCount
		}
		return index[i].Name < index[j].Name
	})
	writeJSON(filepath.Join(*out, "majors.json"), index)

	fmt.Printf("✓ 专业 %d 个（跨校聚合）→ %s\n", len(index), *out)
}
