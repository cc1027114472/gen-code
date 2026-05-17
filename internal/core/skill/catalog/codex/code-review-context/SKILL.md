---
name: code-review-context
description: Model visible context
---

Codex 会维护一个上下文（消息历史），并在推理请求中发送给模型。

1. 不要重写历史记录 - 上下文必须以增量方式逐步构建。
2. 避免频繁更改上下文，以免导致缓存未命中。
3. 不要有无界项目 - 注入到模型上下文中的所有内容都必须具有有界大小和硬上限。
4. 不要有超过 10K tokens 的项目。
5. 将可能超过 >1k tokens 的新单个项目标记为 P0。这些需要额外的人工审查。
6. 所有注入的片段都必须在 `core/context` 中定义为 struct，并实现 ContextualUserFragment trait
