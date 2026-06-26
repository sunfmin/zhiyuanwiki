# 托管：Cloudflare Pages

静态产物托管在 **Cloudflare Pages**，连接 `sunfmin/zhiyuanwiki` 仓库自动构建部署，服务于域名根路径（无 GitHub Pages 项目页的 base-path 问题），零成本、零 ICP 备案。

**已知取舍**：受众是中国大陆考生，而 Cloudflare 属海外 CDN，国内访问可能偏慢或不稳。明确**不**在上线时走"国内云 OSS+CDN + ICP 备案"路线——备案需域名实名 + 约 2–3 周等待期，会拖慢验证。先用 Cloudflare 快速上线验证内容与功能，跨过 MVP 后再评估是否迁国内+备案或加自定义域名/国内加速。
