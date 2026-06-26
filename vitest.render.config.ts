import { defineConfig } from "vitest/config";

// 渲染测试专用：起 astro preview + Playwright，较慢。前置需 `npm run build`。
export default defineConfig({
  test: {
    include: ["tests/render/**/*.test.ts"],
    testTimeout: 60_000,
    hookTimeout: 40_000,
    fileParallelism: false,
  },
});
