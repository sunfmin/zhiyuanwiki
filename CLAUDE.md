# zhiyuanwiki

## 改完流程：先测试，再开浏览器交人工

每次改完代码，**先用测试确认通过**，然后**打开浏览器交给我手动测试**——绿测试不等于完工。

1. 测试：逻辑改动跑 `npm test`（单元）；任何 UI 改动跑 `npm run build && npm run test:render`（渲染，慢，serve `dist/`）。必须全绿。
2. 通过后起 dev server 并打开改动的页面：

   ```
   npm run dev              # 打印 URL，如 http://localhost:4321（端口被占会自增）
   open <打印出的 URL>/zj/   # 改了哪个页面就开哪个；macOS 用 open
   ```

   把要看的 URL/页面告诉我，并保持 server 运行。

## 数据管线（构建期）

构建期 SQLite staging 库（官方 xlsx → 结构化数据的中转库，见 ADR-0014）**规范位置**：

```
/Users/sunfmin/Library/Mobile Documents/com~apple~CloudDocs/zhiyuanwiki/zhiyuan.db
```

工具 `cmd/zhiyuan-data`（`go build -o zhiyuan-data ./cmd/zhiyuan-data`）各子命令默认 `-db out/zhiyuan.db`，
指向上面这份库时传 `-db "<上述路径>"`。原始 xlsx 与该库都是本机产物、不入仓库。

xlsx 源：多数省在 `~/Downloads/高考志愿/各省份/<省名>`；浙江/黑龙江的分数·计划走万师兄树
`~/Developments/zhiyuan/官方数据`（`zj`/`hlj` 专用 import 内部硬编码）；西藏是独立包
`~/Downloads/31、西藏-2026志愿填报资料`（`xz` 用 `-src ~/Downloads`）。

**重新导入全部省份 excel → db**（按省幂等整省替换，首省刷新全国表，其余 `-skip-national`）：
见 `scripts/reimport-all.sh`（先 seed 本地副本、逐省导入、成功后拷回规范位置，容错并汇总）。
导入后如需刷新站点 JSON：对每省顺跑 `fenduan→yuanxiao→zhuanye→dingwei`、最后 `landing`（同样带 `-db`）。

## Agent skills

### Issue tracker

Issues and PRDs are tracked as **GitHub issues** (via the `gh` CLI); external PRs are **not** a triage surface. See `docs/agents/issue-tracker.md`.

### Triage labels

Default 1:1 vocabulary — `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context layout — one `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.
