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

## Agent skills

### Issue tracker

Issues and PRDs are tracked as **GitHub issues** (via the `gh` CLI); external PRs are **not** a triage surface. See `docs/agents/issue-tracker.md`.

### Triage labels

Default 1:1 vocabulary — `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context layout — one `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.
