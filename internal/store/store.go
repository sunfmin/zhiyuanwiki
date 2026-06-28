// Package store 是高考志愿数据的「构建期 SQLite staging」（见 ADR-0014）。
// 各省脏解析 + 全国表入库，下游投影成网站 JSON。DB 是规范化真相，运行时不连库。
// 只依赖 core，进出都用 core 类型，与省份代码解耦。
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/sunfmin/zhiyuanwiki/internal/core"
)

// DB 是 staging 库句柄。
type DB struct{ sql *sql.DB }

const schema = `
CREATE TABLE IF NOT EXISTS school (        -- 全国院校属性（按归一化校名）
  name_norm TEXT PRIMARY KEY, name TEXT, province TEXT, city TEXT,
  ownership TEXT, kind TEXT, is985 INTEGER, is211 INTEGER, syl INTEGER, rank INTEGER
);
CREATE TABLE IF NOT EXISTS major_catalog (  -- 全国 专业→学科门类（按校名+专业）
  name_norm TEXT, major TEXT, menlei TEXT
);
CREATE TABLE IF NOT EXISTS major_score (    -- 各省 专业录取分数 行
  prov TEXT, year INTEGER, track TEXT, batch TEXT,
  school_code TEXT, school_name TEXT, group_code TEXT,
  major TEXT, sel_ke TEXT, min_score INTEGER, min_rank INTEGER, max_score INTEGER
);
CREATE TABLE IF NOT EXISTS plan (           -- 各省 招生计划 行
  prov TEXT, year INTEGER, track TEXT, batch TEXT,
  school_code TEXT, school_name TEXT, group_code TEXT, group_name TEXT,
  major TEXT, full_name TEXT, remark TEXT, sel_ke TEXT,
  plan INTEGER, schooling TEXT, tuition TEXT
);
CREATE TABLE IF NOT EXISTS yifenyiduan (    -- 各省 一分一段 行
  prov TEXT, year INTEGER, track TEXT, score INTEGER, count INTEGER, cum INTEGER,
  control_line INTEGER  -- 本科批控制线（特控线），同一 年×科类 各行相同
);
CREATE INDEX IF NOT EXISTS idx_score_prov ON major_score(prov);
CREATE INDEX IF NOT EXISTS idx_plan_prov  ON plan(prov);
CREATE INDEX IF NOT EXISTS idx_yfd_prov   ON yifenyiduan(prov);
`

// Open 打开（或新建）staging 库并建表（幂等）。
func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("建表: %w", err)
	}
	return &DB{sql: db}, nil
}

func (d *DB) Close() error { return d.sql.Close() }

// tx 在一个事务里执行 fn（批量写必走，否则逐行 commit 极慢）。
func (d *DB) tx(fn func(*sql.Tx) error) error {
	t, err := d.sql.Begin()
	if err != nil {
		return err
	}
	if err := fn(t); err != nil {
		t.Rollback()
		return err
	}
	return t.Commit()
}

// SchoolInfo 是全国院校属性表的一条。
type SchoolInfo struct {
	Name, Province, City, Ownership, Kind string
	Is985, Is211, Syl                     bool
	Rank                                  int
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ReplaceSchools 全量重写全国院校属性表（按归一化校名去重，后到不覆盖）。
func (d *DB) ReplaceSchools(infos []SchoolInfo) error {
	return d.tx(func(t *sql.Tx) error {
		if _, err := t.Exec(`DELETE FROM school`); err != nil {
			return err
		}
		st, err := t.Prepare(`INSERT OR IGNORE INTO school
			(name_norm,name,province,city,ownership,kind,is985,is211,syl,rank)
			VALUES (?,?,?,?,?,?,?,?,?,?)`)
		if err != nil {
			return err
		}
		defer st.Close()
		for _, s := range infos {
			if _, err := st.Exec(core.NormName(s.Name), s.Name, s.Province, s.City,
				s.Ownership, s.Kind, b2i(s.Is985), b2i(s.Is211), b2i(s.Syl), s.Rank); err != nil {
				return err
			}
		}
		return nil
	})
}

// CatalogRow 是 专业→门类 的一条（按校名+专业）。
type CatalogRow struct{ SchoolName, Major, Menlei string }

// ReplaceCatalog 全量重写全国专业门类表。
func (d *DB) ReplaceCatalog(rows []CatalogRow) error {
	return d.tx(func(t *sql.Tx) error {
		if _, err := t.Exec(`DELETE FROM major_catalog`); err != nil {
			return err
		}
		st, err := t.Prepare(`INSERT INTO major_catalog (name_norm,major,menlei) VALUES (?,?,?)`)
		if err != nil {
			return err
		}
		defer st.Close()
		for _, r := range rows {
			if _, err := st.Exec(core.NormName(r.SchoolName), r.Major, r.Menlei); err != nil {
				return err
			}
		}
		return nil
	})
}

// ReplaceScores 按省幂等重写专业录取分数行。
func (d *DB) ReplaceScores(prov string, rows []core.MajorScoreRow) error {
	return d.tx(func(t *sql.Tx) error {
		if _, err := t.Exec(`DELETE FROM major_score WHERE prov=?`, prov); err != nil {
			return err
		}
		st, err := t.Prepare(`INSERT INTO major_score
			(prov,year,track,batch,school_code,school_name,group_code,major,sel_ke,min_score,min_rank,max_score)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`)
		if err != nil {
			return err
		}
		defer st.Close()
		for _, r := range rows {
			if _, err := st.Exec(prov, r.Year, r.Track, "", r.SchoolCode, r.SchoolName,
				r.GroupCode, r.MajorName, r.SelKe, r.MinScore, r.MinRank, r.MaxScore); err != nil {
				return err
			}
		}
		return nil
	})
}

// ReplacePlan 按省幂等重写招生计划行。
func (d *DB) ReplacePlan(prov string, rows []core.PlanRow) error {
	return d.tx(func(t *sql.Tx) error {
		if _, err := t.Exec(`DELETE FROM plan WHERE prov=?`, prov); err != nil {
			return err
		}
		st, err := t.Prepare(`INSERT INTO plan
			(prov,year,track,batch,school_code,school_name,group_code,group_name,major,full_name,remark,sel_ke,plan,schooling,tuition)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
		if err != nil {
			return err
		}
		defer st.Close()
		for _, r := range rows {
			if _, err := st.Exec(prov, r.Year, r.Track, r.Batch, r.SchoolCode, r.SchoolName,
				r.GroupCode, r.GroupName, r.MajorName, r.FullName, r.Remark, r.SelKe,
				r.Plan, r.Schooling, r.Tuition); err != nil {
				return err
			}
		}
		return nil
	})
}

// ReplaceYiFenYiDuan 按省幂等重写一分一段行（多年多科类）。
func (d *DB) ReplaceYiFenYiDuan(prov string, yfds []*core.YiFenYiDuan) error {
	return d.tx(func(t *sql.Tx) error {
		if _, err := t.Exec(`DELETE FROM yifenyiduan WHERE prov=?`, prov); err != nil {
			return err
		}
		st, err := t.Prepare(`INSERT INTO yifenyiduan (prov,year,track,score,count,cum,control_line) VALUES (?,?,?,?,?,?,?)`)
		if err != nil {
			return err
		}
		defer st.Close()
		for _, y := range yfds {
			for _, e := range y.Entries {
				if _, err := st.Exec(prov, y.Year, y.Track, e.Score, e.Count, e.Cumulative, y.ControlLine); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
