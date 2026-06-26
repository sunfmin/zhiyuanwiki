// 可复用渲染胶水：把项目真实页面（astro preview 服务 dist/，含真实 src/data + public/data）
// 用真实浏览器渲染成 PNG。岛屿在真实浏览器里水合、真实 Tailwind 生效、真实数据被 fetch。
// 唯一“胶水”在这里：起 preview server + 开浏览器 + goto + 截图。各渲染测试只换 path/interact。
//
// 前置：需先 `npm run build`（preview 服务的是 dist/）。
import { chromium, type Browser, type Page } from "playwright";
import { spawn, type ChildProcess } from "node:child_process";
import { mkdirSync } from "node:fs";
import { dirname, resolve } from "node:path";

export interface Preview {
  baseURL: string;
  stop: () => void;
}

/** 起一个 astro preview（服务已构建的 dist/），轮询到可访问为止。 */
export async function startPreview(port = 4388): Promise<Preview> {
  const bin = resolve("node_modules/.bin/astro");
  const proc: ChildProcess = spawn(bin, ["preview", "--port", String(port)], {
    cwd: resolve("."),
    stdio: "ignore",
  });
  const baseURL = `http://localhost:${port}`;
  const deadline = Date.now() + 30_000;
  while (Date.now() < deadline) {
    try {
      const r = await fetch(baseURL + "/");
      if (r.ok) return { baseURL, stop: () => proc.kill() };
    } catch {
      /* 还没起来 */
    }
    await new Promise((s) => setTimeout(s, 300));
  }
  proc.kill();
  throw new Error(`astro preview 未在 30s 内就绪（端口 ${port}）。先跑 npm run build？`);
}

export interface RenderOpts {
  baseURL: string;
  name: string; // 输出文件名（不含扩展名）
  path: string; // 页面路径，如 /dingwei
  viewport?: { width: number; height: number };
  /** 截整页（默认 true）；false 则只截视口（适合看“首屏”细节）。 */
  fullPage?: boolean;
  /** 在截图前操作页面（填表单、等结果出现等）。 */
  interact?: (page: Page) => Promise<void>;
}

/** 渲染一个页面到 out/<name>.png，返回 page/browser（供内容断言）与文件路径。 */
export async function renderToImage(
  opts: RenderOpts,
): Promise<{ page: Page; browser: Browser; out: string }> {
  const browser = await chromium.launch();
  const page = await browser.newPage({
    viewport: opts.viewport ?? { width: 1120, height: 1600 },
    deviceScaleFactor: 2,
  });
  await page.goto(opts.baseURL + opts.path, { waitUntil: "networkidle" });
  if (opts.interact) await opts.interact(page);
  const out = resolve("out", opts.name + ".png");
  mkdirSync(dirname(out), { recursive: true });
  await page.screenshot({ path: out, fullPage: opts.fullPage ?? true });
  return { page, browser, out };
}
