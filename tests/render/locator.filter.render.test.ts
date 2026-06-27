// 渲染「位次定位」优化后的筛选区 → out/locator-filter-zj.png。
// 优化点：最常用的筛选（专业大类 / 院校层次 / 城市层级 / 专业关键词）常显在顶部，无需展开即可用；
// 体积大或低频的维度（办学性质 / 学校类别 / 省份 / 计划下限 / 隐藏中外）收进「更多筛选」抽屉。
// 用真实浙江数据（7选3 综合类），端到端验证这套信息架构。见 Locator.tsx 筛选区注释。
import { afterAll, beforeAll, expect, test } from "vitest";
import { startPreview, renderToImage, type Preview } from "./render-glue";

let server: Preview;
beforeAll(async () => {
  server = await startPreview(4393);
}, 40_000);
afterAll(() => server?.stop());

// 常显（快捷）维度 vs 抽屉内（更多筛选）维度——本测试的核心契约。
const QUICK = ["专业大类", "院校层次", "城市层级", "专业关键词"];
const DRAWER = ["办学性质", "学校类别", "省份（院校所在地）", "隐藏中外合作"];

test(
  "筛选优化：常用筛选常显顶部、其余收进「更多筛选」抽屉（浙江真实数据）",
  async () => {
    const { page, browser, out } = await renderToImage({
      baseURL: server.baseURL,
      name: "locator-filter-zj",
      path: "/zj/",
      viewport: { width: 1180, height: 1480 },
      fullPage: false,
      interact: async (p) => {
        await p.getByPlaceholder("输入分数", { exact: true }).fill("600");
        await p.getByText("你的全省位次").waitFor({ timeout: 8_000 });
        await p.waitForFunction(
          () => document.querySelectorAll('a[href^="/zj/yuanxiao/"]').length > 0,
          { timeout: 8_000 },
        );

        // 收起态：常显维度全部可见，应用最常用的两项（无需点开「更多筛选」）。
        const before = await p.locator('a[href^="/zj/yuanxiao/"]').count();
        await p.getByRole("button", { name: "工学", exact: true }).first().click(); // 专业大类
        await p.getByRole("button", { name: "985", exact: true }).first().click(); // 院校层次
        await p.waitForFunction(
          (n) => {
            const c = document.querySelectorAll('a[href^="/zj/yuanxiao/"]').length;
            return c > 0 && c < n;
          },
          before,
          { timeout: 8_000 },
        );

        // 展开「更多筛选」，露出抽屉内维度，供截图呈现完整两段式布局。
        await p.getByRole("button", { name: "更多筛选" }).click();
        await p.getByText("省份（院校所在地）").waitFor({ timeout: 4_000 });
      },
    });

    // 契约 1：四项常用筛选常显（且抽屉展开后仍在）。
    const mainText = await page.locator("main").innerText();
    for (const q of QUICK) expect(mainText).toContain(q);

    // 契约 2：抽屉维度此刻（已展开）可见。
    for (const d of DRAWER) expect(mainText).toContain(d);

    // 契约 3：常显维度即时生效——选了工学 + 985 后仍有结果，且「清除全部」常驻。
    expect(mainText).toContain("清除全部");
    const cards = await page.locator('a[href^="/zj/yuanxiao/"]').count();
    expect(cards).toBeGreaterThan(0);
    console.log(`优化筛选区渲染 ${cards} 个可填报项 → ${out}`);

    await browser.close();
  },
  60_000,
);

test(
  "筛选优化：未展开时抽屉维度应隐藏（信息架构回归）",
  async () => {
    const { page, browser } = await renderToImage({
      baseURL: server.baseURL,
      name: "locator-filter-zj-collapsed",
      path: "/zj/",
      viewport: { width: 1180, height: 900 },
      fullPage: false,
      interact: async (p) => {
        await p.getByPlaceholder("输入分数", { exact: true }).fill("600");
        await p.getByText("你的全省位次").waitFor({ timeout: 8_000 });
        await p.waitForFunction(
          () => document.querySelectorAll('a[href^="/zj/yuanxiao/"]').length > 0,
          { timeout: 8_000 },
        );
      },
    });

    // 默认收起：常显维度可见，但抽屉维度的标签不应出现在 DOM。
    const mainText = await page.locator("main").innerText();
    for (const q of QUICK) expect(mainText).toContain(q);
    expect(mainText).not.toContain("省份（院校所在地）");
    expect(mainText).not.toContain("学校类别");
    expect(mainText).not.toContain("隐藏中外合作");

    await browser.close();
  },
  60_000,
);
