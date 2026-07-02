// 渲染浙江「专业页面」→ out/zhuanye-zj.png（整页）+ out/zhuanye-zj-hero.png（首屏）。
// 专业页 = 跨院校把「开设该专业的院校按录取水平横排比较」（ADR：专业为横向比较视图）。
// URL 段改用专业名本身（名字即 slug，与院校归一化校名一致）。
// 端到端验证：真实 src/data/zj/majors/<名>.json → 真实页面逻辑（跨校排序表 + 锚回院校页链接）。
import { afterAll, beforeAll, expect, test } from "vitest";
import { startPreview, renderToImage, type Preview } from "./render-glue";

let server: Preview;
beforeAll(async () => {
  server = await startPreview(4392);
}, 40_000);
afterAll(() => server?.stop());

test(
  "浙江专业页面（大气科学 · 跨院校录取位次横向比较）",
  async () => {
    const { page, browser, out } = await renderToImage({
      baseURL: server.baseURL,
      name: "zhuanye-zj",
      path: "/zj/zhuanye/大气科学/",
      viewport: { width: 1120, height: 1600 },
      fullPage: true,
      interact: async (p) => {
        await p.getByRole("heading", { level: 1, name: "大气科学" }).waitFor({ timeout: 8_000 });
      },
    });

    const main = page.locator("main");
    const mainText = await main.innerText();

    // 1) 真实数据流到 UI：专业名 + 开设院校数（来自 major.schools.length）。换一致源（ADR-0022）后 14 所。
    expect(mainText).toContain("大气科学");
    expect(mainText).toMatch(/14\s*所院校/);

    // 2) 首屏「顶点」= 最难考院校（schools[0]）：位次读数（朱红签名量）+ 校名。
    expect(mainText).toContain("最难考的院校");
    expect(mainText).toContain("南京大学"); // 位次 8,445 的最难考院校
    expect(mainText).toMatch(/录取位次\s*8,445–160,802/);

    // 3) 签名排行：14 所院校排成 14 条 .zy-row，各按 #z-<mk> 锚回其院校页历年区。
    const rows = main.locator(".zy-row");
    expect(await rows.count()).toBe(14);
    // 院校链接段用归一化校名（名字即 slug），且带 #z- 锚点。
    const firstHref = await rows.first().getAttribute("href");
    expect(firstHref).toMatch(/^\/zj\/yuanxiao\/.+\/#z-[0-9a-f]{8}$/);
    // 每条都有对数刻度竞争度条；最难考(首条)满格、最易(末条)近乎空。
    expect(await main.locator(".zy-bar-fill").count()).toBe(14);
    const firstBar = await main.locator(".zy-bar-fill").first().evaluate((e) => e.style.width);
    const lastBar = await main.locator(".zy-bar-fill").last().evaluate((e) => e.style.width);
    expect(parseFloat(firstBar)).toBeGreaterThan(parseFloat(lastBar));
    expect(parseFloat(firstBar)).toBeGreaterThan(95); // 最难考≈满格
    // 单科类省（浙江=综合）不该重复渲染科类列/角标。
    expect(await main.locator(".zy-chip").count()).toBe(0);

    // 首屏细节单独截一张（看 hero + 签名排行）。
    await page.screenshot({ path: out.replace(".png", "-hero.png"), fullPage: false });

    // 窄屏：竞争度条应隐去，读数自身够比较。另存一张移动端首屏。
    await page.setViewportSize({ width: 390, height: 844 });
    expect(await main.locator(".zy-bar:visible").count()).toBe(0);
    await page.screenshot({ path: out.replace(".png", "-mobile.png"), fullPage: false });

    console.log(`浙江专业页：22 所院校 · 首个链接 ${firstHref} → ${out}`);

    await browser.close();
  },
  60_000,
);
