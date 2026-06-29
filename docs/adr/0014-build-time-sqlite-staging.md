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

江苏/湖南/四川/安徽走通了这条路。**编排已收敛**：`import.go` 的 `provParsers` 注册表是「构建期
staging 管线」省份的单一登记处——登记即走 `import → DB → 投影`，`fenduan`/`yuanxiao` 据此分流
（不再有 `case "<slug>"`）；DB 投影 `buildDBBundle` 是省份无关的，**不再每省照抄**。加一省：

1. 解析器：**格式与四川/安徽一致的干净 group 省（广西/湖北/云南/河南…）直接复用 `internal/group3p12`**
   ——这份解析是 sc/ah 逐字节相同的超集（组代码列 `专业组代码`\|`所属专业组` 双兜底、`专业名称`\|`专业`
   双列名、`StripParenTail`、`物理类/历史类`\|裸`物理/历史`→归一、学制/学费 `ColContains` 容单位后缀），
   同构省共享一份、不再每省照抄（与 ADR-0013 不冲突：那条缝是给**异构**省发散用，同构省共享正合
   single-source-of-truth）。`group3p12` 已吸收大量列名/批次/科类皮肤差异（计划列 `计划数`、组列 `专业组`\|
   `组代码`、track 列 `文理`、选科列 `选课要求`\|`选科`、batch `一段/二段/常规批`、科类 `综合`\|`物理科目组合`、
   组名兜底建组），故江西/吉林/甘肃/山东等只是登记即用。真正各写各包的只剩**计划表缺关键 join 列**（天津
   计划无院校代码，`internal/tj` 由「专业组代码」剥后缀还原）与**老文理口径**（新疆 理科/文科，见就绪度表）。
   新省先试 `group3p12`，跑挂了再判断是扩 `group3p12` 还是另写包。
2. `import.go`：`provDirName[slug]` + `provParsers[slug]`（三个解析函数 + 可选 `PlanMust`/`ScoreMust`）。
   **`PlanMust`/`ScoreMust` 几乎必填**：2025 数据目录名常含「招生计划」子串、且有更大的 2024 同名副本或
   「专业录取分数/分数线」混在 root 子树里，默认按体积选最大会错配——精确指向 2025 文件。分数文件名各省
   不一（`专业录取分数`\|`专业分数线`），用 `ScoreMust` 覆盖。
3. `provinces.go`（Go）+ `src/lib/provinces.ts`（前端镜像）各加一条，slug 唯一（河南=`henan`、河北=`hebei`、
   海南=`hain` 避与既有冲突）。**选 `model`**：有真实院校专业组→`group`；专业平行志愿（组列全空）→`major`
   （前端 `fillModel`）。综合(3+3) `subjectMode:"pick3of6"`，3+1+2 `"primary+reselect"`。
4. **投影按 model 分流**（无需改投影代码）：group→`buildDBBundle`；major→`buildMajorBundle`（通用，全国
   school 表挂属性、track-aware，双科类省如重庆/辽宁开箱即用）；浙江 major 走专用 `buildDBBundleMajor`
   （`model:"major-zj"`，一表联动 by-code 属性）。
5. 跑 `import → fenduan → yuanxiao → zhuanye → dingwei`（或 `scripts/import-province.sh <slug>...`），
   再 build/test/render，最后开浏览器人工核对。

**坑一**：Go 把 `*_js.go`、`*_amd64.go` 等当 GOOS/GOARCH 构建约束——`js` 是 GOOS=wasm，
文件名 `yuanxiao_js.go` 在 darwin 会被**静默排除**（包目录名 `internal/js/` 不受影响）。
**坑二**：院校专业组在 3+1+2 下是按 (院校,**科类**,组代码) 一等的——见 ADR-0015。

## 省份就绪度（截至 2026-06；已接入 28 省，剩 3 省待数据）

**已接入 28 省**：
- **group · 物理/历史**（3+1+2，院校专业组）：黑龙江 浙江(综合) 江苏 湖南 四川 安徽 广西 湖北 云南 河南
  陕西 内蒙古 广东 福建 宁夏 江西 吉林 甘肃。解析全走 `internal/group3p12`（江西 `计划数`/`专业组`、
  甘肃 `文理`/`组代码`、吉林仅 `专业组名称`、内蒙 裸 `专业组` 等列名差异已由别名 + 组名兜底吸收）。
- **group · 综合**（3+3 + 院校专业组）：北京 上海 海南 天津。`group3p12.keep` 放行「综合」；上海组列
  `专业组代码`、天津计划无院校代码（`internal/tj` 由专业组代码剥后缀还原）。前端 6选3（`pick3of6`）。
- **major · 物理/历史**（专业平行志愿，组列全空）：重庆 贵州 辽宁 河北。走通用 `buildMajorBundle`
  （全国 school 表挂属性、track-aware 双科类）。
- **major · 综合**：山东（`buildMajorBundle`）、浙江（专用 `buildDBBundleMajor`，一表联动 by-code）。
- **major · 老文理**（理科/文科，专业平行志愿）：新疆。专属 `internal/xj`（`group3p12.*With(keep={理科,文科})`，
  yfd 无批次列时免 batch 过滤）；组列全空走 `buildMajorBundle`；前端 `wenli` 定位模式（无选科、仅理/文切换）。
  理科/文科**不进** `group3p12` 默认 keep——否则会把重庆/贵州等 major 省 22-24 年的老文理历史行一并吸入。

**关键纠错**：组码不同形（录取分数 `（501）`/`（01）` vs 计划 `第501组`/`01`/裸数字）对 fill **无害**——
`core.BuildGroups2026`/`buildMajorBundle` 的往年位次按 `(院校代码, 专业名)` 挂接、组仅取自计划侧建组。
故江西/陕西/北京等一度被误判「组码需归一」的省，实际登记即用。「能否接入」的真判据是：**录取分数有逐专业
最低位次** + **计划表 join 列（院校代码/专业名）齐备**——组代码可有可无（无组→major 模型）。

**待接入 3 省**：
- **青海 / 山西 / 西藏**（缺数据，见 #26）：本数据源无「含逐专业最低位次的录取分数合表」（青海/山西仅
  院校级投档线/PDF；西藏 2025 位次列全空）。等省考院补齐后按 model 接入，代码侧无需预留。

## 给未来的人

DB 是手段不是目的。运行时契约永远是「仓库里的静态 JSON」。任何想让站点直连 DB 的改动，
都要先回到这条 ADR：staging 在构建期，渲染在静态层。
