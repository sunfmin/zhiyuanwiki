# 退役省专属 builder：黑龙江/浙江全量并入 staging（含 major 模型按码属性）

ADR-0014 给「各省份」省（江苏/湖南/四川/安徽）建了构建期 SQLite staging，但黑龙江、浙江仍走各自的
legacy builder（`buildHLJBundle`/`buildZJBundle`）直接读源树。结果是两条投影路径并存、`yuanxiao`/
`fenduan` 里按 slug 硬编码分流，且黑龙江/浙江的属性靠省专属内存索引（`hlj.LoadSchoolMeta` 按校名、
`zj.LoadAttrs` 按校代码），与全国表两套真相。

## 决定

把黑龙江、浙江也全量并入 staging，退役 legacy builder，分流改按**填报模型**而非 slug。

1. **统一投影，按模型分流**。`yuanxiao` 按 `province.model` 选投影：`group→buildDBBundle`
   （院校专业组）、`major→buildDBBundleMajor`（专业平行志愿，浙江）。`fenduan` 一律从 DB 投影。
   无 per-slug 分支。`buildHLJBundle`/`buildZJBundle`/`yuanxiao_zj.go` 删除。

2. **源异构 → 专用 import，不强塞通用 glob**。黑龙江/浙江源跨两棵树、布局特殊，故走 `importHLJ`/
   `importZJ`（通用 `importProvince` 仍服务 各省份 省）：
   - 黑龙江：分数(2023-25)/计划(2026) 取**万师兄树**（与旧 builder 同源，保证院校/叶子/组零回归）；
     一分一段取**各省份树**（2024/2025 物理+历史，2025 合表带本科线）+ 保留万师兄 2026 物理。
   - 浙江：分数/计划/一分一段/属性 取万师兄树（与旧 builder 同源 → 投影 diff 一致）。
   - 全国院校属性/专业门类表仍每次从各省份树刷新（全国一份，见 ADR-0014）。

3. **浙江 major 模型入库**。浙江是综合·专业平行志愿（无院校专业组），与组模型阻抗不一：
   - 计划行 `PlanRow2026`（有「招生类型」、无组码）↔ 统一 `core.PlanRow` 用 `zj.ToCorePlan/
     FromCorePlan` 两向无损转换：科类落「综合」、组码空、**招生类型落 `core.PlanRow.Batch`**
     （此前闲置的 plan.batch 列）——投影时还原喂 `core.IsCoop`，否则中外合作判定回归。
   - `buildDBBundleMajor` 走 `zj.BuildPlan2026` 产 `plan2026`（院校×专业）而非 `groups2026`。

4. **省专属按码属性另立投影（`school_attr`）**。浙江 `city_tier` 是「一表联动」的**显式标签**
   （非由城市名推算）、且全套属性**按院校代码**挂接；全国 `school` 表按校名、无 `city_tier`，且
   城市/类型命名与浙江源大相径庭（实测城市差 1432/1654、类型差 1632/1654）——直接迁到全国表会
   整体回归。故新增 `school_attr(prov, school_code, …, city_tier)` 表，`importZJ` 按码写入，
   `buildDBBundleMajor` 按码投影。组模型省仍走全国 `school` 表（按校名）。

## 取舍与回归核对

- **黑龙江**：院校/叶子/组结构与旧 builder **逐字节同**（0 diff）；一分一段严格增量（+历史/2024/2025）；
  `equivRank` 因接入真实往年总人数而细化（旧版缺总人数未缩放）；院校属性统一到全国表（覆盖 +45，
  军事类院校/个别校区因不在全国表丢 meta，属全国表数据缺口、宜上游统一补）。
- **浙江**：`plan2026` 成员/排序、plan/选科/学制/学费/中外合作/往年位次/等效位次、以及 meta
  （含 `city_tier`）/层次 **全部逐字节保全**；唯一差异是 **15 项 menlei**（0.06%）——门类源由
  「一表联动」改挂全国专业目录（单一真相）后，个别长名大类/小语种（交通运输类/生物医学工程类/
  格鲁吉亚语等）分类器关键词兜底变弱。属单一真相迁移的一致代价（黑龙江同源变 119 项），
  分类器对带括号大类名的处理改进另案跟进。

## 结果

一条投影路径（按模型二选一）、一族 import（通用 + 2 个专用）、`fenduan` 全从 DB。6 省同构于
staging：DB 是规范化真相，JSON 是派生投影（ADR-0014）。运行时仍是静态站、不连库。
