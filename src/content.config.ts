import { defineCollection, z } from "astro:content";
import { glob } from "astro/loaders";

// 方法论文章集合（Markdown）。
const guide = defineCollection({
  loader: glob({ pattern: "**/*.md", base: "./src/content/guide" }),
  schema: z.object({
    title: z.string(),
    description: z.string().optional(),
    order: z.number().default(99),
  }),
});

export const collections = { guide };
