# 多省导入用「构建期 SQLite staging」，运行时仍是静态 JSON

从 2 省（黑龙江/浙江）扩到 ~31 省时，新数据源（`各省份/`）有两个旧管线没有的特征：

1. **全国共享表**：`全国高等院校信息汇总`（3067 校：城市/层次/性质/类型/排名）、
   `全国高等院校开设专业汇总`（专业→学科门类）、`省控线汇总`——都是**全国一份**，按
   校名挂接到每省的分数行。这是 join。31 省各自重复读 6.5 MB 全国表、各自拼名，既慢又易飘。
2. **异构收敛**：各省 xlsx 格式杂（多年/单年/老文理/分年命名），但模型统一（院校、
   专业叶子、`年×科类×批次` 录取线、招生计划、一分一段）。

## 决定

引入一层**仅构建期**的 SQLite staging（`internal/store`），管线变为：

```
xlsx（不入仓库, ADR-0005）
  └─ zhiyuan-data import   省专属脏解析 + 全国表 → 规范化入库（按省幂等）
       └─ SQLite（out/zhiyuan.db, 不入仓库, 派生产物）
            └─ zhiyuan-data yuanxiao/fenduan   纯投影：DB → []core.MajorScoreRow / 属性 / 计划
                 └─ src|public/data/<slug>/*.json（仍是提交物）
                      └─ astro build → Cloudflare（运行时完全不变）
```

DB 是**规范化真相**，JSON 是**派生投影**——契合本仓 single-source-of-truth 取向。
站点是静态站，运行时只读已提交 JSON，**不连数据库**；DB 和原始 xlsx 一样是本机构建产物，
不进 CI、不入仓库（CI 仍只跑 `astro build`）。

## 缝的位置（与 ADR-0013 一致）

- **入库（因省而异）**：每省自己的逐行解析循环留在 `internal/<省>`，复用 `core.OpenSheet`
  定位表头。不做共用大配置（见 ADR-0013）。
- **投影（共用）**：`core.AggregateLeaves`、`core.BuildGroups2026`（3+1+2 院校专业组聚合，
  江苏/黑龙江同形，本次从 hlj 下沉到 core）等聚合逻辑共用，只是数据来源从 xlsx 换成 DB 行。
- `store` 只依赖 `core`，进出都用 `core` 类型（`MajorScoreRow`/`PlanRow`/`YiFenYiDuan`）；
  省份代码与 store 解耦。

## 取舍

- **代价**：多一层 schema + DAO；多一个纯 Go 依赖 `modernc.org/sqlite`（无 cgo）。
- **为什么不纯内存**：现有 `AttrIndex` 已是「内存 DB」，2 省够用。但 31 省要的是
  跨省 join、按省幂等重导、摸排时的交互式 `SELECT`——持久化在这个阶段收益最大。
- **院校代码是分省的**（招生代码），故 `major_score` 按 `(prov, code)` 建键，靠**校名**
  join 全国 `school` 取属性（与黑龙江既有拼名思路一致，但收成一处）。

## 给未来的人

DB 是手段不是目的。运行时契约永远是「仓库里的静态 JSON」。任何想让站点直连 DB 的改动，
都要先回到这条 ADR：staging 在构建期，渲染在静态层。
