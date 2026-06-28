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
// 即使带结果 + 展开「更多筛选」抽屉（含 30+ 学校类别、34 省份长列）+ 常显关键词框里塞极端长串，
// 也不能把整页撑宽。兜底来自 global.css 的 html { overflow-x: clip } + chip / 输入框的可换行与 max-w 加固。
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
        // 院校 / 专业两个关键词框都常显，各塞极端长串；再展开「更多筛选」把最长的省份/类别列也纳入溢出考验。
        const longMajor = "计算机科学与技术 软件工程 人工智能 数据科学 电子信息工程 自动化";
        const longSchool = "哈尔滨工业大学 黑龙江中医药大学 哈尔滨师范大学 东北农业大学";
        await p.getByPlaceholder("空格分隔=任一匹配，如 计算机 软件", { exact: true }).fill(longMajor);
        await p.getByPlaceholder("空格分隔=任一匹配，如 浙江大学 师范", { exact: true }).fill(longSchool);
        await p.getByRole("button", { name: "更多筛选" }).click();
        await p.waitForTimeout(150);
      },
    });

    const { scrollW, clientW } = await page.evaluate(() => ({
      scrollW: document.documentElement.scrollWidth,
      clientW: document.documentElement.clientWidth,
    }));
    expect(scrollW).toBeLessThanOrEqual(clientW);

    // OR 语义在常显关键词框的占位符里说明（空格分隔 = 任一匹配）；以专业框为例校验。
    const kw = page.getByPlaceholder("空格分隔=任一匹配，如 计算机 软件", { exact: true });
    expect(await kw.getAttribute("placeholder")).toContain("任一匹配");
    expect(await kw.inputValue()).toContain("软件工程");

    await browser.close();
  },
  60_000,
);

test(
  "手机窄屏：常显快捷筛选 + 展开更多筛选抽屉",
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
        // 专业大类常显——直接选「工学」；再展开抽屉看到其余维度。
        await p.getByRole("button", { name: "工学", exact: true }).first().click();
        await p.getByRole("button", { name: "更多筛选" }).click();
      },
    });
    console.log(`手机筛选面板 → ${out}`);
    await browser.close();
  },
  60_000,
);
