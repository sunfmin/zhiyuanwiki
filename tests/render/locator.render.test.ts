// 渲染「位次定位」结果画面（冲/稳/保 三列填满）→ out/locator-results.png。
// 用真实数据：输入一个分数，真实 2026 一分一段换算位次，真实 locator 数据填三档。
import { afterAll, beforeAll, expect, test } from "vitest";
import { startPreview, renderToImage, type Preview } from "./render-glue";

let server: Preview;
beforeAll(async () => {
  server = await startPreview();
}, 40_000);
afterAll(() => server?.stop());

test(
  "位次定位结果画面（真实数据，冲/稳/保 三档）",
  async () => {
    const { page, browser, out } = await renderToImage({
      baseURL: server.baseURL,
      name: "locator-results",
      path: "/dingwei",
      viewport: { width: 1180, height: 1120 },
      fullPage: false, // 只截首屏：控件 + 三列顶部，便于看清设计细节
      interact: async (p) => {
        // 物理类·分数模式（默认）；填一个中段分数，真实换算出位次并填满三档。
        // 页面有两个数字输入（定位 + 一分一段换算），按占位符精确定位到「定位」那个。
        await p.getByPlaceholder("输入分数", { exact: true }).fill("520");
        await p.getByText("你的全省位次").waitFor({ timeout: 8_000 });
        await p.waitForFunction(
          () => document.querySelectorAll('a[href^="/yuanxiao/"]').length > 0,
          { timeout: 8_000 },
        );
      },
    });

    // 断言：真实数据流经真实逻辑渲染出了应显示的内容（查 DOM，不查像素）。
    const mainText = await page.locator("main").innerText();
    expect(mainText).toContain("你的全省位次");
    for (const b of ["冲", "稳", "保"]) expect(mainText).toContain(b);
    const cards = await page.locator('a[href^="/yuanxiao/"]').count();
    expect(cards).toBeGreaterThan(0);
    console.log(`渲染 ${cards} 个可填报项 → ${out}`);

    await browser.close();
  },
  60_000,
);

test(
  "施加过滤（专业大类=工学 + 省份=北京）后三档计数收窄、chip 常显",
  async () => {
    const { page, browser, out } = await renderToImage({
      baseURL: server.baseURL,
      name: "locator-filtered",
      path: "/dingwei",
      viewport: { width: 1180, height: 1280 },
      fullPage: false,
      interact: async (p) => {
        await p.getByPlaceholder("输入分数", { exact: true }).fill("520");
        await p.waitForFunction(
          () => document.querySelectorAll('a[href^="/yuanxiao/"]').length > 0,
          { timeout: 8_000 },
        );
        const before = await p.locator('a[href^="/yuanxiao/"]').count();

        // 展开筛选面板，选 专业大类=工学 + 省份=北京（北京 chip 由 meta 派生）。
        await p.getByRole("button", { name: "筛选" }).click();
        await p.getByRole("button", { name: "工学", exact: true }).first().click();
        await p.getByRole("button", { name: "北京", exact: true }).first().click();

        // 过滤后渲染数 < 过滤前（且仍有结果）。
        await p.waitForFunction(
          (n) => {
            const c = document.querySelectorAll('a[href^="/yuanxiao/"]').length;
            return c > 0 && c < n;
          },
          before,
          { timeout: 8_000 },
        );
      },
    });

    // 生效过滤以可移除 chip 常显；三档计数已是过滤后的数。
    const mainText = await page.locator("main").innerText();
    expect(mainText).toContain("工学");
    expect(mainText).toContain("北京");
    expect(mainText).toContain("清除全部");
    const cards = await page.locator('a[href^="/yuanxiao/"]').count();
    expect(cards).toBeGreaterThan(0);
    console.log(`过滤后渲染 ${cards} 个可填报项 → ${out}`);

    await browser.close();
  },
  60_000,
);
