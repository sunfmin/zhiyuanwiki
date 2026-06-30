// 定位页「所有搜索都体现在 URL」——端到端契约（真实浏览器、浙江真实数据）。
// 1) 操作定位页（分数 + 选科 + 大类 + 层次 + 院校关键词）后，地址栏 query 应完整反映这份搜索。
// 2) 把这串 URL 当作「别人复制来的链接」在全新页面打开，应原样重建同一搜索并出同一结果。
import { afterAll, beforeAll, expect, test } from "vitest";
import { chromium, type Browser } from "playwright";
import { startPreview, type Preview } from "./render-glue";

let server: Preview;
let browser: Browser;
beforeAll(async () => {
  server = await startPreview(4394);
  browser = await chromium.launch();
}, 40_000);
afterAll(async () => {
  await browser?.close();
  server?.stop();
});

const SCHOOL_KW_PH = "空格分隔=任一匹配，如 浙江大学 师范";

test(
  "搜索写入 URL：分数 / 选科 / 大类 / 层次 / 院校关键词都进 query",
  async () => {
    const page = await browser.newPage({ viewport: { width: 1180, height: 1480 } });
    await page.goto(server.baseURL + "/zj/", { waitUntil: "networkidle" });

    await page.getByPlaceholder("输入分数", { exact: true }).fill("650");
    await page.getByText("你的全省位次").waitFor({ timeout: 8_000 });
    await page.getByRole("button", { name: "工学", exact: true }).first().click(); // 专业大类
    await page.getByRole("button", { name: "985", exact: true }).first().click(); // 院校层次
    await page.getByPlaceholder(SCHOOL_KW_PH).fill("浙江");

    // 等 URL 把这份搜索回写完整（replaceState 在 effect 里，等到最后一项关键词出现为止）。
    await page.waitForFunction(() => location.search.includes("schoolKeyword"), { timeout: 8_000 });

    const u = new URL(page.url());
    expect(u.searchParams.get("score")).toBe("650");
    expect(u.searchParams.get("subjects")).toBe("物理,化学,生物"); // 浙江 7选3 默认三科
    expect(u.searchParams.get("categories")).toBe("工");
    expect(u.searchParams.get("levels")).toBe("985");
    expect(u.searchParams.get("schoolKeyword")).toBe("浙江");
    // 单科类省份不写 track；未设维度不出现。
    expect(u.searchParams.has("track")).toBe(false);
    expect(u.searchParams.has("ownership")).toBe(false);

    console.log(`定位搜索 → URL: ${u.search}`);
    await page.close();
  },
  60_000,
);

test(
  "复制链接即结果：打开带搜索的 URL，原样重建状态并出结果",
  async () => {
    const share =
      server.baseURL +
      "/zj/?score=650&subjects=" +
      encodeURIComponent("物理,化学,生物") +
      "&categories=" +
      encodeURIComponent("工") +
      "&levels=985&schoolKeyword=" +
      encodeURIComponent("浙江");

    const page = await browser.newPage({ viewport: { width: 1180, height: 1480 } });
    await page.goto(share, { waitUntil: "networkidle" });
    await page.getByText("你的全省位次").waitFor({ timeout: 8_000 });
    await page.waitForFunction(
      () => document.querySelectorAll('a[href^="/zj/yuanxiao/"]').length > 0,
      { timeout: 8_000 },
    );

    // 输入框、关键词框回填。
    expect(await page.getByPlaceholder("输入分数", { exact: true }).inputValue()).toBe("650");
    expect(await page.getByPlaceholder(SCHOOL_KW_PH).inputValue()).toBe("浙江");

    // 大类 / 层次 chip 呈选中态（选中样式 bg-slate-800）。
    const gongCls = await page.getByRole("button", { name: "工学", exact: true }).first().getAttribute("class");
    const cls985 = await page.getByRole("button", { name: "985", exact: true }).first().getAttribute("class");
    expect(gongCls).toContain("bg-slate-800");
    expect(cls985).toContain("bg-slate-800");

    // 顶部位次显示已定位、筛选生效（清除全部常驻）、主档仍有结果。
    const mainText = await page.locator("main").innerText();
    expect(mainText).toContain("你的全省位次");
    expect(mainText).toContain("清除全部");
    const cards = await page.evaluate(
      () => [...document.querySelectorAll('a[href^="/zj/yuanxiao/"]')].filter((a) => !a.closest("details")).length,
    );
    expect(cards).toBeGreaterThan(0);

    await page.close();
  },
  60_000,
);
