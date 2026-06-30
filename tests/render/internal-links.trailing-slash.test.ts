// 尾斜杠不变量（ADR-0018，修订见 ADR-0019）：R2 自定义域原生服务目录索引，canonical 带尾斜杠——
// `…/index.html` 与不带尾斜杠的目录路径都被 R2 308 收敛到 `…/`。故任何「root-relative、非锚点、
// 无文件后缀、且不以 / 结尾」的站内链接虽不会死链，却会平白多吃一跳 308。本测试扫描构建产物
// dist/**/*.html，断言这类链接为零——把「链接直接命中 R2 canonical、零重定向」钉死，挡住未来回归
// （如新组件又写出无尾斜杠的 href）。
//
// 这是纯文件扫描，不起浏览器；放在 tests/render/ 仅因它依赖 `npm run build` 的产物（该套件已假设
// dist 存在，见 render-glue.ts）。R2 的 clean-URL 308 只在 Cloudflare 边缘存在，本地测不到。
import { expect, test } from "vitest";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, resolve } from "node:path";

const DIST = resolve("dist");

function htmlFiles(dir: string): string[] {
  const out: string[] = [];
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) out.push(...htmlFiles(p));
    else if (name.endsWith(".html")) out.push(p);
  }
  return out;
}

const HREF = /href="([^"]*)"/g;
// 末段带文件后缀（如 .json/.css/.svg/.png/.js/.xml/.txt/.ico/.webp）→ 资源，按精确 key 提供。
const HAS_EXT = /\.[a-z0-9]{1,8}$/i;

function wouldDeadLink(raw: string): boolean {
  const path = raw.split("#")[0].split("?")[0]; // 只看路径，去掉 hash/query
  if (path === "") return false; // 纯锚点 #...
  if (!path.startsWith("/")) return false; // 外链/相对/mailto/tel
  if (path.startsWith("//")) return false; // 协议相对外链
  if (path.endsWith("/")) return false; // 目录式，重写命中
  if (HAS_EXT.test(path)) return false; // 资源文件
  return true; // 非 / 结尾 + 无后缀 → 会 404
}

test(
  "dist 内无会在 R2 Rules-only 下 404 的站内链接（尾斜杠不变量 · ADR-0018）",
  () => {
    const files = htmlFiles(DIST);
    expect(files.length, "dist 下没有 .html——是否漏跑 npm run build？").toBeGreaterThan(0);

    const violations = new Set<string>();
    for (const f of files) {
      const html = readFileSync(f, "utf8");
      for (const m of html.matchAll(HREF)) {
        if (wouldDeadLink(m[1])) violations.add(`${f.slice(DIST.length)} → ${m[1]}`);
      }
    }

    const list = [...violations];
    expect(
      list,
      `发现会 404 的站内链接（应补尾斜杠）：\n${list.slice(0, 30).join("\n")}` +
        (list.length > 30 ? `\n…共 ${list.length} 处` : ""),
    ).toEqual([]);
  },
  120_000,
);
