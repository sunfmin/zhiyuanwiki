// 渲染省份列表落地页（根路径 /）→ out/landing.png（桌面宽表）+ out/landing-mobile.png（卡片）。
// 用真实构建产物：已上线省从真实 schools.json 汇总收录列，未上线省置灰「敬请期待」。
import { afterAll, beforeAll, expect, test } from "vitest";
import { startPreview, renderToImage, type Preview } from "./render-glue";
import { CURRENT_GAOKAO_YEAR } from "../../src/lib/provinces";

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
    expect(mainText).toContain("本科线"); // 省情列（物理本科批控制线）
    expect(mainText).toContain("185"); // 河南本省院校（未上线行也填省情）
    expect(mainText).toContain("463"); // 江苏 2025 物理本科线

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
    // 卡片也带一分一段年（与桌面同源）
    expect(mainText).toContain("一分一段");
    console.log(`落地页手机卡片 → ${out}`);
    await browser.close();
  },
  60_000,
);

test(
  "落地页：一分一段年（当年标绿）+ 本科招生计划占比 + 首屏规模条",
  async () => {
    const { page, browser, out } = await renderToImage({
      baseURL: server.baseURL,
      name: "landing-features",
      path: "/",
      viewport: { width: 1180, height: 1400 },
      fullPage: true,
    });

    const mainText = await page.locator("main").innerText();
    // 新列「一分一段」（本站收录轴）与既有「本科招生计划」表头同在
    expect(mainText).toContain("一分一段");
    expect(mainText).toContain("本科招生计划");
    // 本科招生计划占比：至少一处「N%」（= 计划 ÷ 高考人数）
    expect(mainText).toMatch(/\d+%/);
    // 首屏规模条：三项真实合计标签都在
    expect(mainText).toContain("已上线省份");
    expect(mainText).toContain("覆盖统考考生");
    expect(mainText).toContain("基准数据年");

    // 一分一段表格单元用 emerald 类标当年——首格内容应恰为 CURRENT_GAOKAO_YEAR。
    // （td.text-emerald-600 只命中一分一段列；规模条用 dd、脚注用 strong，均不会误匹配。）
    const greenCells = await page.locator("td.text-emerald-600");
    expect(await greenCells.count()).toBeGreaterThan(0);
    expect((await greenCells.first().innerText()).trim()).toBe(String(CURRENT_GAOKAO_YEAR));

    console.log(`落地页新列/规模条：${await greenCells.count()} 省已覆盖当年 → ${out}`);
    await browser.close();
  },
  60_000,
);
