package store

import (
	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

// LoadScores 读某省全部专业录取分数行（投影回 core.MajorScoreRow）。
func (d *DB) LoadScores(prov string) ([]core.MajorScoreRow, error) {
	rows, err := d.sql.Query(`SELECT year,track,school_code,school_name,group_code,major,sel_ke,min_score,min_rank,max_score
		FROM major_score WHERE prov=?`, prov)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.MajorScoreRow
	for rows.Next() {
		var r core.MajorScoreRow
		if err := rows.Scan(&r.Year, &r.Track, &r.SchoolCode, &r.SchoolName, &r.GroupCode,
			&r.MajorName, &r.SelKe, &r.MinScore, &r.MinRank, &r.MaxScore); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LoadPlan 读某省全部招生计划行。
func (d *DB) LoadPlan(prov string) ([]core.PlanRow, error) {
	rows, err := d.sql.Query(`SELECT year,track,school_code,school_name,group_code,group_name,major,full_name,remark,sel_ke,plan,schooling,tuition
		FROM plan WHERE prov=?`, prov)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.PlanRow
	for rows.Next() {
		var r core.PlanRow
		if err := rows.Scan(&r.Year, &r.Track, &r.SchoolCode, &r.SchoolName, &r.GroupCode, &r.GroupName,
			&r.MajorName, &r.FullName, &r.Remark, &r.SelKe, &r.Plan, &r.Schooling, &r.Tuition); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LoadTotals 读某省各年×科类的考生总人数（= 该年该科类一分一段的最大累计），用于等效位次缩放。
func (d *DB) LoadTotals(prov string) (map[core.YearTrack]int, error) {
	rows, err := d.sql.Query(`SELECT year,track,MAX(cum) FROM yifenyiduan WHERE prov=? GROUP BY year,track`, prov)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[core.YearTrack]int{}
	for rows.Next() {
		var yt core.YearTrack
		var total int
		if err := rows.Scan(&yt.Year, &yt.Track, &total); err != nil {
			return nil, err
		}
		out[yt] = total
	}
	return out, rows.Err()
}

// LoadYiFenYiDuan 读某省全部一分一段，按 年×科类 分组成 YiFenYiDuan（升序），province 为中文省名。
func (d *DB) LoadYiFenYiDuan(prov, province string) ([]*core.YiFenYiDuan, error) {
	rows, err := d.sql.Query(`SELECT year,track,score,count,cum FROM yifenyiduan WHERE prov=?
		ORDER BY year,track,score`, prov)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byKey := map[core.YearTrack]*core.YiFenYiDuan{}
	var order []core.YearTrack
	for rows.Next() {
		var yt core.YearTrack
		var e core.FenduanEntry
		if err := rows.Scan(&yt.Year, &yt.Track, &e.Score, &e.Count, &e.Cumulative); err != nil {
			return nil, err
		}
		y := byKey[yt]
		if y == nil {
			y = &core.YiFenYiDuan{Province: province, Track: yt.Track, Year: yt.Year}
			byKey[yt] = y
			order = append(order, yt)
		}
		y.Entries = append(y.Entries, e)
	}
	out := make([]*core.YiFenYiDuan, 0, len(order))
	for _, yt := range order {
		out = append(out, byKey[yt])
	}
	return out, rows.Err()
}

// Menlei 从全国专业门类表重建分类器（精确映射 + 关键词兜底由 core 提供）。
func (d *DB) Menlei() (*core.MenleiClassifier, error) {
	rows, err := d.sql.Query(`SELECT major,menlei FROM major_catalog`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	mc := core.NewMenleiClassifier()
	for rows.Next() {
		var major, menlei string
		if err := rows.Scan(&major, &menlei); err != nil {
			return nil, err
		}
		mc.Learn(major, menlei)
	}
	return mc, rows.Err()
}

// SchoolIndex 按校名查全国院校属性：先精确（归一化名），再退到去括号基名（分校继承母体）。
type SchoolIndex struct {
	byNorm map[string]SchoolInfo
	byBase map[string]SchoolInfo
}

// Lookup 查院校属性：精确名优先，再退到去括号基名。
func (si *SchoolIndex) Lookup(name string) (SchoolInfo, bool) {
	if v, ok := si.byNorm[core.NormName(name)]; ok {
		return v, true
	}
	if v, ok := si.byBase[core.BaseName(name)]; ok {
		return v, true
	}
	return SchoolInfo{}, false
}

// Len 返回收录院校数。
func (si *SchoolIndex) Len() int { return len(si.byNorm) }

// SchoolIndex 载入全国院校属性并建归一化名 / 基名两级索引。
func (d *DB) SchoolIndex() (*SchoolIndex, error) {
	rows, err := d.sql.Query(`SELECT name,province,city,ownership,kind,is985,is211,syl,rank FROM school`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	si := &SchoolIndex{byNorm: map[string]SchoolInfo{}, byBase: map[string]SchoolInfo{}}
	for rows.Next() {
		var s SchoolInfo
		var is985, is211, syl int
		if err := rows.Scan(&s.Name, &s.Province, &s.City, &s.Ownership, &s.Kind, &is985, &is211, &syl, &s.Rank); err != nil {
			return nil, err
		}
		s.Is985, s.Is211, s.Syl = is985 == 1, is211 == 1, syl == 1
		si.byNorm[core.NormName(s.Name)] = s
		if base := core.BaseName(s.Name); base != "" {
			if _, ok := si.byBase[base]; !ok {
				si.byBase[base] = s
			}
		}
	}
	return si, rows.Err()
}
