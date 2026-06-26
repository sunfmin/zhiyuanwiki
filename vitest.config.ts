import { configDefaults, defineConfig } from "vitest/config";

// 默认 `npm test` 只跑快速单元测试；渲染测试（起浏览器/preview，慢）单独跑 `npm run test:render`。
export default defineConfig({
  test: {
    exclude: [...configDefaults.exclude, "tests/render/**"],
  },
});
