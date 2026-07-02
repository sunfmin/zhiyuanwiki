# 浙江改用与他省一致的数据源 + 通用 major 模型，退役浙江专属栈（照搬山东）

浙江自 ADR-0009 接入起走的是**专属路径**：源在万师兄的 `09、浙江-2026高考志愿填报资料/` 树、
解析在 `internal/zj/*`、投影用 `major-zj` 模型（`buildDBBundleMajor` + `zj.BuildPlan2026`）、
院校属性按院校代码挂「一表联动」表（`school_attr`）。ADR-0021 把院校主键收口到归一化校名后，
其后果段要求「按码属性改按名」——issue #40 本为此立。但排查 #40 时发现一个**更根本**的事实：
浙江其实**不需要任何专属栈**。

## 促成事实（推翻「浙江太特殊」）

1. **一致源已存在且完整**：`各省份/浙江/浙江/` 下有 `22-25年全国高校在浙江的专业录取分数.xlsx`、
   `…招生计划.xlsx`、`一分一段/浙江{2022-2025}年的一分一段表.xlsx`——与重庆/贵州等通用 `major`
   省同一命名规范。与旧 `09、浙江` 源近乎同一份（专业录取分数 6,408,262 vs 6,409,437 字节，同数据集
   略异版本），换源预期只有小 diff。
2. **单科类「综合」+ `major` 模型 + 一致源 早已跑通——就是山东**（`sd`: `tracks:["综合"], model:"major"`）。
   `group3p12` 的 `keep` 收「综合」，选科要求/批次只是列里的字符串（山东 3+3 选考已证明可解析）。
   浙江（综合、专业平行志愿、22-25 一致源）与山东**结构同型**。

结论：浙江可**照搬山东**，不再需要专属源、专属解析、专属投影、按码属性。

## 决定

1. **换源**：浙江分数/计划/一分一段改从 `各省份/浙江` 读，在 `provParsers` 登记浙江条目
   （`Scores/Plan: group3p12.*`，`YFD` 用 `group3p12.ParseYiFenYiDuan` 或保留 `zj.ParseYiFenYiDuan`——
   以浙江一分一段表实际格式为准），退役专用 `importZJ` 与 `09、浙江` 源。
2. **换模型**：浙江 `model` 从 `major-zj` 改为 `major`（同山东），报考视图走通用
   `buildMajorBundle` + `buildPlanMajorsTracked`，删除 `buildDBBundleMajor` 与 `major-zj` case。
3. **属性一律按名**（本条即吞掉 #40）：通用 `major` 路径本就 `idx.Lookup(s.Name)` 挂全国 `school` 表，
   校区经 `byBase` 继承母体 985/211；city_tier 由静态 `core.CityTier` 派生。**退役按码属性栈**：
   `school_attr` 表 + `store.SchoolAttr`/`SchoolAttrs`/`ReplaceSchoolAttrs`、`internal/zj/attrs.go`。
4. **退役 `internal/zj` 的解析/投影**：`BuildPlan2026`/`ToCorePlan`/`FromCorePlan`/`PlanMajor`/
   `ParseMajorScoresXLSX`/`ParsePlan2026XLSX` 等随之删除。**保留 `zj.BaseMajorName`**——浙江大类招生的
   方向折叠是**源无关的域规则**，`zhuanye.go` 仍用它跨校归并大类。
5. **不改公开 URL / 数据模型**：浙江仍是综合单科类、专业平行志愿、一段/二段/提前批；仅换「数据从哪来、
   用哪套代码算」。

## 后果

- 删掉一整条省专属栈（源树 + `internal/zj` 大半 + `major-zj` 投影 + 按码属性），浙江与其余 major 省
   同一套代码路径——SSOT 大幅收敛，延续 ADR-0017「退役省专属 builder」的方向。
- **换源有数据 diff**（同源略异版本）：验收不追逐字节等同，而是抽查院校×专业条数、位次、分段口径与
   现版一致量级，人工复核改名/未匹配。
- 触及既有 ADR 的适用面：**ADR-0009**（浙江专属接入路径）此后仅存「综合/专业平行志愿」的模型语义、
   其「专属源+专属解析」部分作废；**ADR-0011**（官方 PDF 一分一段）——一致源已含浙江 2022-2025 一分一段
   xlsx，若其即官方数据的规范化落地则 0011 的产物不变、仅取用路径变；**ADR-0021** 的按名属性后果由本
   ADR 对浙江落地。
- `CONTEXT.md` 术语表「城市层级=静态映射」「院校层次按名挂接」自此对全 31 省名副其实（浙江不再是反例），
   **无需改动**。
- 可逆：源路径与 `provParsers` 条目、模型选择都是小改，git 可退。
