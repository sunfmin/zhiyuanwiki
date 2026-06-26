// 渲染浙江「位次定位」结果画面 → out/locator-zj.png。
// 浙江=单科类「综合」（无物理/历史切换）+ 7选3 选考；用真实 2025 综合一分一段换算位次，
// 真实 locator-zonghe 数据填三档。验证多省份泛化在浙江侧端到端可用。见 ADR-0009。
import { afterAll, beforeAll, expect, test } from "vitest";
import { startPreview, renderToImage, type Preview } from "./render-glue";

let server: Preview;
beforeAll(async () => {
  server = await startPreview(4390);
}, 40_000);
afterAll(() => server?.stop());

test(
  "浙江定位结果画面（综合单科类 · 7选3 · 真实数据）",
  async () => {
    const { page, browser, out } = await renderToImage({
      baseURL: server.baseURL,
      name: "locator-zj",
      path: "/zj/",
      viewport: { width: 1180, height: 1120 },
      fullPage: false,
      interact: async (p) => {
        // 综合类·分数模式（默认）；填一个一段线以上的分数，真实换算位次填满三档。
        await p.getByPlaceholder("输入分数", { exact: true }).fill("600");
        await p.getByText("你的全省位次").waitFor({ timeout: 8_000 });
        await p.waitForFunction(
          () => document.querySelectorAll('a[href^="/zj/yuanxiao/"]').length > 0,
          { timeout: 8_000 },
        );
      },
    });

    const mainText = await page.locator("main").innerText();
    expect(mainText).toContain("你的全省位次");
    for (const b of ["冲", "稳", "保"]) expect(mainText).toContain(b);
    // 浙江特有：7选3 选考（含「技术」），且无物理/历史科类切换按钮。
    expect(mainText).toContain("选考科目（7选3）");
    expect(mainText).toContain("技术");
    const trackToggle = await page.getByRole("button", { name: "历史类", exact: true }).count();
    expect(trackToggle).toBe(0);
    const cards = await page.locator('a[href^="/zj/yuanxiao/"]').count();
    expect(cards).toBeGreaterThan(0);
    console.log(`浙江渲染 ${cards} 个可填报项 → ${out}`);

    await browser.close();
  },
  60_000,
);
