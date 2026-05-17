---
name: codex-bug
description: Diagnose GitHub bug reports in openai/codex. Use when given a GitHub issue URL from openai/codex and asked to decide next steps such as verifying against the repo, requesting more info, or explaining why it is not a bug; follow any additional user-provided instructions.
---

# Codex 缺陷诊断

## 概述

诊断一个 Codex GitHub 缺陷报告，并决定下一步操作：根据来源验证、请求更多信息，或解释为什么它不是一个缺陷。

## 工作流

1. 确认输入

- 需要一个指向 `github.com/openai/codex/issues/…` 的 GitHub issue URL。
- 如果 URL 缺失或不在正确的仓库中，向用户索要正确的链接。

2. 网络访问

- 始终立即通过网络访问该 issue，即使你认为访问被阻止或不可用也是如此。
- 优先使用 GitHub API 而不是 HTML 页面，因为 HTML 页面噪声较多：
  - Issue: `https://api.github.com/repos/openai/codex/issues/<number>`
  - Comments: `https://api.github.com/repos/openai/codex/issues/<number>/comments`
- 如果环境需要显式批准，通过工具按需申请，并继续执行，无需额外提示用户。
- 只有在请求批准后网络尝试仍然失败时，才说明你离线能够做什么（例如起草一个回复模板），并询问如何继续。

3. 阅读 issue

- 使用 GitHub API 的响应（issue + comments）作为事实来源，而不是抓取 HTML issue 页面。
- 提取：标题、正文、复现步骤、期望结果与实际结果、环境、日志，以及任何附件。
- 记录该报告是否已经包含日志或会话详情。
- 如果报告包含 thread ID，在摘要中提到它，并在你有权限访问时用它来查找日志和会话详情。

4. 在调查前先总结缺陷

- 在深入检查代码、文档或日志之前，用你自己的话为该报告写一个简短摘要。
- 包括已报告的行为、期望行为、复现步骤、环境，以及已经附带了哪些证据或缺少哪些证据。

5. 决定处理方式

- 当报告具体且很可能可复现时，**根据来源验证**。检查相关的 Codex 文件（如果无法访问，则说明应检查哪些文件）。
- 当报告含糊、缺少复现步骤，或缺少日志/环境信息时，**请求更多信息**。
- 当报告与当前行为或文档化约束相矛盾时，**解释它不是缺陷**（引用 issue 中的证据以及你检查过的任何本地来源）。

6. 回复

- 简明提供你的发现和后续步骤报告。
