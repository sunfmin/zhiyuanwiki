# 退役 internal/hlj 的 core 薄门面（aliases.go）

ADR-0009 多省份泛化时，把省份无关原语下沉到 `internal/core`，并在 `internal/hlj/aliases.go` 用类型别名 / 函数别名把这些符号按原 `hlj.*` 名再导出，**为的是迁移期「保持黑龙江调用方与测试不变」**——当时那是一项有意的取舍（见 ADR-0009「代价与边界」：「黑龙江 hlj 用类型别名再导出 core 符号以保持调用方与测试不变——多一层薄门面，但逻辑只此一份」）。

迁移期已结束。本 ADR **推翻**那条「保留薄门面」的决定：删除 `aliases.go`，调用方直引 `core`。

## 为什么退役

对 `aliases.go` 做删除测试：删掉后复杂度不在别处重现，只是把 `hlj.X` 换成 `core.X`（约 10 处）——这是 pass-through，不是深模块。它还带来反作用：

- `cmd` 同时 import `hlj` 别名与 `core`，制造不对称；
- 真正的省份逻辑（`LoadSchoolMeta` / `BuildGroups2026` / `ParsePlanXLSX`）本就不走门面，门面只覆盖了「碰巧也被 hlj 用到的 core 原语」，掩盖了 hlj 解析器其实在调 core 助手这一事实。

seam 应落在真正变化的地方——**省份专属的 xlsx 解析**——而不是一层别名。

## 决定

1. 删除 `internal/hlj/aliases.go`。
2. hlj 包内部（`plan.go` / `major_scores.go` / `school_meta.go`）对 `HasCell` / `FindCol` / `Cell` / `ParseLeadingInt` / `NormName` / `BaseName` 及 `MajorScoreRow` / `MajorLeaf` / `YearScore` / `YearTrack` / `NormalizeMajorName` / `MajorKey` / `AggregateLeaves` / `EquivRank` / `IsCoop` 等改为直引 `core.X`。
3. `cmd/zhiyuan-data/yuanxiao.go` 的 `hlj.LoadMenlei` / `hlj.ParseYiFenYiDuanXLSX` 改为 `core.LoadMenlei` / `core.ParseYiFenYiDuanXLSX`。`hlj.ParseMajorScoresXLSX` / `hlj.LoadSchoolMeta` / `hlj.Group2026` / `hlj.ParsePlanXLSX` / `hlj.BuildGroups2026` 是真正的省份符号，保留。
4. 纯 core 函数的测试（`city_tier_test.go`、`attrs_test.go`）**移到 `internal/core`**——测试与被测代码同包，不再借道 hlj。`major_scores_test.go` / `plan_test.go` 仍含 hlj 专属解析测试，留在 hlj 并以 `core.X` 引用。

## 边界

- 这是纯结构调整：无行为变化，产物 JSON 不变；`go test ./...` 为安全网。
- 黑龙江 hlj 包从此只装**省份专属**逻辑（解析物理/历史·本科批 xlsx、组视图、院校属性/门类挂接、选科判定）。
