// 渲染省份列表落地页（根路径 /）→ out/landing.png（桌面宽表）+ out/landing-mobile.png（卡片）。
// 用真实构建产物：已上线省从真实 schools.json 汇总收录列，未上线省置灰「敬请期待」。
import { afterAll, beforeAll, expect, test } from "vitest";
import { startPreview, renderToImage, type Preview } from "./render-glue";

let server: Preview;
beforeAll(async () => {
  server = await startPreview();
}, 40_000);
afterAll(() => server?.stop());

test(
  "落地页桌面：6 省已上线可点 + 敬请期待省置灰，收录列有真实数字",
  async () => {
    const { page, browser, out } = await renderToImage({
      baseURL: server.baseURL,
      name: "landing",
      path: "/",
      viewport: { width: 1180, height: 1400 },
      fullPage: true,
    });

    const mainText = await page.locator("main").innerText();
    expect(mainText).toContain("选择你的省份");
    expect(mainText).toContain("敬请期待"); // 未上线省状态
    expect(mainText).toContain("河南"); // 未上线大省在册
    expect(mainText).toContain("高考人数"); // 省情列
    expect(mainText).toContain("本省院校"); // 省情列
    expect(mainText).toContain("185"); // 河南本省院校（未上线行也填省情）

    // 6 个已上线省各有一个进入 /[slug]/ 的链接（桌面表 + 手机卡片各一份 → 去重后 6 个 slug）。
    const liveSlugs = await page.evaluate(() => {
      const hrefs = [...document.querySelectorAll('a[href^="/"]')]
        .map((a) => (a as HTMLAnchorElement).getAttribute("href") || "")
        .filter((h) => /^\/(hlj|zj|js|hn|sc|ah)\/$/.test(h));
      return [...new Set(hrefs)].sort();
    });
    expect(liveSlugs).toEqual(["/ah/", "/hlj/", "/hn/", "/js/", "/sc/", "/zj/"]);

    console.log(`落地页：已上线 ${liveSlugs.length} 省 → ${out}`);
    await browser.close();
  },
  60_000,
);

test(
  "落地页手机：卡片形态（窄视口）",
  async () => {
    const { page, browser, out } = await renderToImage({
      baseURL: server.baseURL,
      name: "landing-mobile",
      path: "/",
      viewport: { width: 390, height: 1400 },
      fullPage: true,
    });
    const mainText = await page.locator("main").innerText();
    expect(mainText).toContain("选择你的省份");
    // 已上线卡片显示收录摘要文案
    expect(mainText).toContain("收录");
    console.log(`落地页手机卡片 → ${out}`);
    await browser.close();
  },
  60_000,
);
