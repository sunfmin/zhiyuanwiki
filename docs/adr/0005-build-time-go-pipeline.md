# 数据管线：build 前用 Go 把 xlsx 预处理为 JSON，原始 xlsx 不入 web 仓库

原始数据是一批 xlsx，解析逻辑不简单（组号逐年变需按院校+专业名等效挂接、位次挂接、选科清洗）。决定写一个 **Go** 命令行工具做 build 前预处理，把 xlsx → 干净的、按 `省份/科类/年份/院校/专业` 结构化的 JSON，提交进 zhiyuanwiki 仓库；Astro 在 build 时只读这些 JSON 生成静态页面。读 xlsx 用 Go 的 excelize 库；现有 `zhiyuan` 项目里的 python 脚本（`hlj_lib.py`/`expand_groups_2026.py` 等）仅作为解析逻辑的**参考规格**，不直接复用。

**原始 xlsx 不进 web 仓库**（版权 + 体积），它们留在 `~/Developments/zhiyuan/官方数据/`。

代价：引入了跨语言的 build 依赖（Go 在前、Astro/node 在后），CI 需装 Go 工具链。换来的是把最易出错的解析逻辑（组号映射/位次挂接/等效位次换算）放进一个强类型、可表驱动测试的 Go 包，而不是用 JS（sheetjs）在前端 build 里重写。

注意：现有 `zhiyuan` 项目里的派生 CSV（候选池/志愿表）是**按单个考生的排除清单过滤过的**，本站须从原始 xlsx **重新生成完整、未过滤**的全量数据集（个人排除项改为访客侧可选筛选），不得直接复用那些被污染的派生文件。
