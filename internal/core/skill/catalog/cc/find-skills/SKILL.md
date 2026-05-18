---
name: find-skills
description: 当用户询问“我该怎么做 X”“帮我找一个做 X 的 skill”“有没有适合做某件事的 skill”，或表达想扩展 agent 能力的需求时，帮助用户发现并安装 agent skill。应在用户寻找可能已经以可安装 skill 形式存在的功能时使用。
---

# 查找技能

这个 skill 用于帮助你从开放的 agent skills 生态中发现并安装技能。

## 何时使用这个技能

当用户出现以下情况时使用这个 skill：

- 询问“我该怎么做 X”，而 X 可能是已有 skill 覆盖的常见任务
- 说“帮我找一个做 X 的 skill”或“有没有适合做 X 的 skill”
- 询问“你能做 X 吗”，而 X 属于某个专门能力
- 表达出想扩展 agent 能力的兴趣
- 想查找工具、模板或工作流
- 提到他们希望在某个特定领域（设计、测试、部署等）获得帮助

## 技能命令行工具是什么？

“Skills CLI”（`npx skills`）是开放式 agent skills 生态的包管理器。技能是模块化的软件包，用来以专门知识、工作流和工具扩展 agent 的能力。

**关键命令：**

- `npx skills find [query]` - 交互式搜索 skill，或按关键词搜索
- `npx skills add <package>` - 从 GitHub 或其他来源安装 skill
- `npx skills check` - 检查 skill 更新
- `npx skills update` - 更新所有已安装的 skill

**浏览技能目录：** 访问 https://skills.sh/

## 如何帮助用户查找技能

### 步骤 1：理解他们的需求

当用户请求某方面帮助时，识别以下内容：

1. 所属领域（例如 React 开发、测试、设计、部署）
2. 具体任务（例如写测试、做动画、审查拉取请求）
3. 这是否属于一个足够常见、很可能已有 skill 的任务

### 步骤 2：搜索技能

使用相关查询运行查找命令：

```bash
npx skills find [query]
```

例如：

- 用户问“怎么让我的 React 应用更快？” -> `npx skills find react performance`
- 用户问“你能帮我做 PR review 吗？” -> `npx skills find pr review`
- 用户说“我需要创建 changelog” -> `npx skills find changelog`

该命令会返回类似结果：

```
Install with npx skills add <owner/repo@skill>

vercel-labs/agent-skills@vercel-react-best-practices
└ https://skills.sh/vercel-labs/agent-skills/vercel-react-best-practices
```

### 步骤 3：向用户展示选项

当你找到相关 skill 时，向用户展示以下内容：

1. skill 名称以及它的用途
2. 他们可以运行的安装命令
3. 一个用于进一步了解的 skills.sh 链接

示例回复：

```
我找到了一个可能有帮助的 skill！“vercel-react-best-practices”
这个 skill 提供来自 Vercel Engineering 的 React 和 Next.js 性能优化指南。

安装方式：
npx skills add vercel-labs/agent-skills@vercel-react-best-practices

了解更多：https://skills.sh/vercel-labs/agent-skills/vercel-react-best-practices
```

### 步骤 4：提出安装

如果用户想继续，你可以替他们安装该 skill：

```bash
npx skills add <owner/repo@skill> -g -y
```

`-g` 标志表示全局安装（用户级），`-y` 表示跳过确认提示。

## 常见技能分类

搜索时，可以考虑以下常见分类：

| 分类 | 示例查询 |
| ---- | -------- |
| Web 开发 | React 性能、Next.js、TypeScript、CSS、Tailwind |
| 测试 | 测试、Jest、Playwright、端到端测试 |
| DevOps | 部署、Docker、Kubernetes、持续集成 |
| 文档 | 文档、README、变更日志、接口文档 |
| 代码质量 | 审查、Lint、重构、最佳实践 |
| 设计 | 界面设计、用户体验、设计系统、无障碍 |
| 效率工具 | 工作流、自动化、Git |

## 高效搜索的小技巧

1. **使用具体关键词**：比如“React 测试”通常比单独搜索“测试”更好
2. **尝试替代表达**：如果“部署”没结果，可以换成“发布”或“持续集成”
3. **检查热门来源**：很多 skill 来自 `vercel-labs/agent-skills` 或 `ComposioHQ/awesome-claude-skills`

## 没有找到技能时怎么办

如果没有相关 skill：

1. 明确说明没有找到现成 skill
2. 表示你仍然可以用通用能力直接帮助完成任务
3. 建议用户可以用 `npx skills init` 创建自己的 skill

示例：

```
我搜索了与 “xyz” 相关的 skill，但没有找到匹配项。
我仍然可以直接帮你完成这项任务！如果你愿意，我可以继续处理。

如果这是你经常要做的事情，也可以创建你自己的 skill：
npx skills init my-xyz-skill
```
