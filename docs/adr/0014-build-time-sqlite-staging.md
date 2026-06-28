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

## 数据源（截至 2026-06）

新源在本机 `~/Downloads/高考志愿/`，**三棵高度重叠的树共 ~52 GB**：
`志愿高报资料/「028」…大数据/`（最全，含政策 docx/艺考）、`2026高考志愿填报大数据(1)/`
（重复副本）、**`各省份/`（最干净：只留录取数据 + `college_data/`，1.3 G，导入就用它）**。
`各省份/<省>/…/` 路径嵌套层数不一，故 import 用**子树 glob**（按文件名子串）定位，不写死路径。

全国表（每省 `college_data/` 下一份相同副本，按**校名**挂接）：
- `全国高等院校信息汇总.xlsx`（3067 校：985/211/双一流/办学性质/所在省/所在城市/学校类型/综合排名）→ `school`
- `全国高等院校开设专业汇总.xlsx`（11 万行：学校+专业→学科门类）→ `major_catalog`

## 接入新省份的配方

江苏走通了这条路（`internal/js` + `import`/`yuanxiao`/`fenduan` 的 js 分支）。加一省：

1. 写 `internal/<省>` 解析器——多数能照抄 `internal/js`，只改**科类取值**（物理类/历史类→归一）
   与**批次过滤**（留本科、丢专科/艺体）。这是真正因省而异的部分（ADR-0013）。
2. `import.go`：登记 `provDirName[slug]` + 加 `importXX` 分支入库。
3. `provinces.go`（Go）+ `src/lib/provinces.ts`（前端镜像）各加一条；3+1+2 省 `fillModel:"group"`。
4. `yuanxiao.go` / `fenduan.go` 加 `case "<slug>"` 走 DB 投影（`buildXXBundle` 照抄 `buildJSBundle`）。
5. 跑 `import → fenduan → yuanxiao → zhuanye → dingwei`，再 build/test。

**坑**：Go 把 `*_js.go`、`*_amd64.go` 等当 GOOS/GOARCH 构建约束——`js` 是 GOOS=wasm，
文件名 `yuanxiao_js.go` 在 darwin 会被**静默排除**。江苏的 cmd 文件命名为 `yuanxiao_jiangsu.go`。
（包目录名 `internal/js/` 不受影响，只有文件名后缀触发。）

## 省份就绪度（按源表格式，截至 2026-06）

- **可直接导（22-25 多年·含位次·内联院校属性）**：浙江✓ 江苏✓ 湖南 重庆 贵州 西藏 新疆 海南 宁夏
- **可导（仅 2025 或分年文件·同 schema·含位次）**：黑龙江✓ 云南 四川 安徽 江西 广西 河南 湖北 北京 天津 辽宁 甘肃 福建 内蒙 吉林
- **需单独啃（表头异常/无合表/无位次）**：山西 陕西 上海 青海 河北

填报模型按高考制度分：3+3（综合）= 浙江/上海/山东/海南/北京/天津 走 `major`；
3+1+2（物理/历史）= 其余多数走 `group`；老文理（新疆/西藏）需文/理科口径。

## 给未来的人

DB 是手段不是目的。运行时契约永远是「仓库里的静态 JSON」。任何想让站点直连 DB 的改动，
都要先回到这条 ADR：staging 在构建期，渲染在静态层。
