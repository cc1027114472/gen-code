---
name: code-review
description: Run a final code review on a pull request
---

使用子代理，借助此仓库中除这个 orchestrator 之外的所有 `code-review-*` skills 来审查代码。每个 skill 对应一个子代理。把完整的 skill 路径传给子代理。使用 xhigh 推理。

你必须返回每一个子代理发现的每一个问题。你可以返回不限数量的 findings。
使用原始 Markdown 报告 findings。
对 findings 编号，便于引用。
每个 finding 都必须包含具体的文件路径和行号。

如果执行评审的 GitHub 用户是 pull request 的所有者，则添加一个 `code-reviewed` label。
除非被明确要求，否则不要留下 GitHub 评论。
