// 渲染「位次定位」在手机窄屏（iPhone 13 量级 390×844）下的布局 → out/locator-mobile*.png。
// 用真实数据，验证窄屏可堆叠使用、冲/稳/保在手机上的浏览方式、过滤面板可用。
import { afterAll, beforeAll, expect, test } from "vitest";
import { startPreview, renderToImage, type Preview } from "./render-glue";

let server: Preview;
beforeAll(async () => {
  server = await startPreview(4389);
}, 40_000);
afterAll(() => server?.stop());

const PHONE = { width: 390, height: 844 };

test(
  "手机窄屏：定位结果首屏",
  async () => {
    const { page, browser, out } = await renderToImage({
      baseURL: server.baseURL,
      name: "locator-mobile",
      path: "/hlj/",
      viewport: PHONE,
      fullPage: false,
      interact: async (p) => {
        await p.getByPlaceholder("输入分数", { exact: true }).fill("520");
        await p.waitForFunction(
          () => document.querySelectorAll('a[href^="/hlj/yuanxiao/"]').length > 0,
          { timeout: 8_000 },
        );
      },
    });

    const mainText = await page.locator("main").innerText();
    expect(mainText).toContain("你的全省位次");
    for (const b of ["冲", "稳", "保"]) expect(mainText).toContain(b);
    const cards = await page.locator('a[href^="/hlj/yuanxiao/"]').count();
    expect(cards).toBeGreaterThan(0);
    console.log(`手机首屏渲染 ${cards} 个可填报项 → ${out}`);
    await browser.close();
  },
  60_000,
);

// 回归：iPhone Pro Max 量级窄屏不应出现整页横向滚动条（html.scrollWidth ≤ clientWidth）。
// 即使带结果 + 展开筛选 + 极端长关键词 chip 也不能把整页撑宽。
// 兜底来自 global.css 的 html { overflow-x: clip } + Locator chip 的可换行加固。
test(
  "手机窄屏：整页无横向溢出",
  async () => {
    const { page, browser } = await renderToImage({
      baseURL: server.baseURL,
      name: "locator-mobile-no-hscroll",
      path: "/hlj/",
      viewport: { width: 440, height: 900 }, // iPhone 17 Pro Max 量级
      fullPage: false,
      interact: async (p) => {
        await p.getByPlaceholder("输入分数", { exact: true }).fill("520");
        await p.waitForFunction(
          () => document.querySelectorAll('a[href^="/hlj/yuanxiao/"]').length > 0,
          { timeout: 8_000 },
        );
        await p.getByRole("button", { name: "筛选" }).click();
        await p
          .getByPlaceholder(/空格分隔/)
          .fill("计算机科学与技术 软件工程 人工智能 数据科学 电子信息工程 自动化");
        await p.waitForTimeout(150);
      },
    });

    const { scrollW, clientW } = await page.evaluate(() => ({
      scrollW: document.documentElement.scrollWidth,
      clientW: document.documentElement.clientWidth,
    }));
    expect(scrollW).toBeLessThanOrEqual(clientW);

    // OR 关系在 chip 上要显式体现（空格分隔 = 任一匹配）。
    const chip = await page.locator('button:has-text("关键词")').first().innerText();
    expect(chip).toContain("或");

    await browser.close();
  },
  60_000,
);

test(
  "手机窄屏：展开筛选面板",
  async () => {
    const { browser, out } = await renderToImage({
      baseURL: server.baseURL,
      name: "locator-mobile-filter",
      path: "/hlj/",
      viewport: { width: 390, height: 1200 },
      fullPage: false,
      interact: async (p) => {
        await p.getByPlaceholder("输入分数", { exact: true }).fill("520");
        await p.waitForFunction(
          () => document.querySelectorAll('a[href^="/hlj/yuanxiao/"]').length > 0,
          { timeout: 8_000 },
        );
        await p.getByRole("button", { name: "筛选" }).click();
        await p.getByRole("button", { name: "工学", exact: true }).first().click();
      },
    });
    console.log(`手机筛选面板 → ${out}`);
    await browser.close();
  },
  60_000,
);
