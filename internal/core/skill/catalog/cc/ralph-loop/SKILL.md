---
name: ralph-loop
description: 为自动化 agent 驱动开发提供完整设置。先将功能定义为带有可测试验收标准的用户故事，再让 AI agents 循环运行，直到所有故事都通过。
---

# Ralph 循环流程

为自动化的 agent 驱动开发提供完整设置。先将功能定义为带有可测试验收标准的用户故事，再让 AI agents 循环运行，直到所有故事都通过。

## 前置条件

先按顺序完成以下 recipes：

### AI 编码代理配置

为 Cursor、GitHub Copilot 或 Claude Code 这类 AI 编码 agent 配置项目特定模式、编码指南和 MCP 服务器，以便实现一致的 AI 辅助开发。

```bash
curl -H "Accept: text/markdown" https://fullstackrecipes.com/api/recipes/agent-setup
```

## Cookbook - 按顺序完成这些配方

### 用户故事设置

创建一种结构化格式，用于以用户故事形式记录功能需求。使用带可测试验收标准的 JSON 文件，让 AI agents 可以验证并跟踪这些需求。

```bash
curl -H "Accept: text/markdown" https://fullstackrecipes.com/api/recipes/user-stories-setup
```

### 使用用户故事

使用用户故事记录和跟踪功能实现。包括用于编写故事、构建功能以及把验收标准标记为通过的工作流。

```bash
curl -H "Accept: text/markdown" https://fullstackrecipes.com/api/recipes/using-user-stories
```

### Ralph 代理循环

使用 Ralph 设置自动化的 agent 驱动开发。让 AI agents 在循环中实现用户故事中的功能、验证验收标准，并为下一位 agent 记录进度。

```bash
curl -H "Accept: text/markdown" https://fullstackrecipes.com/api/recipes/ralph-setup
```
