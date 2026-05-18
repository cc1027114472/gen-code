---
name: create-implementation-plan
description: '为新功能、既有代码重构、包升级、设计、架构或基础设施创建新的实施计划文件。'
---

# 创建实施计划

## 主要指令

你的目标是为 `${input:PlanPurpose}` 创建一个新的实施计划文件。你的输出必须可供机器读取、具备确定性，并且其结构应适合由其他 AI 系统或人工自主执行。

## 执行上下文

该提示词面向 AI 与 AI 之间的通信以及自动化处理。所有指令都必须按字面含义理解，并在不依赖人工解释或澄清的情况下系统化执行。

## 核心要求

- 生成可由 AI 代理或人工完整执行的实施计划
- 使用零歧义的确定性语言
- 将所有内容组织成适合自动解析和执行的结构
- 确保内容完全自包含，理解时不依赖外部资料

## 计划结构要求

计划必须由离散的、原子化的阶段构成，并包含可执行任务。除非显式声明跨阶段依赖，否则每个阶段都必须可由 AI 代理或人工独立处理。

## 阶段架构

- 每个阶段都必须具有可衡量的完成标准
- 除非显式声明依赖，否则同一阶段内的任务必须可以并行执行
- 所有任务描述都必须包含具体文件路径、函数名和精确实现细节
- 任何任务都不应依赖人工解释或临场决策

## 面向 AI 优化的实施标准

- 使用明确、无歧义、无需解释的语言
- 将所有内容组织为机器可解析的格式（表格、列表、结构化数据）
- 在适用时包含具体文件路径、行号和精确代码引用
- 显式定义所有变量、常量和配置值
- 在每个任务描述中提供完整上下文
- 为所有标识符使用标准化前缀（REQ-、TASK- 等）
- 包含可自动验证的校验标准

## 输出文件规范

- 将实施计划文件保存到 `/plan/` 目录
- 使用命名约定：`[purpose]-[component]-[version].md`
- `purpose` 前缀限定为：`upgrade|refactor|feature|data|infrastructure|process|architecture|design`
- 示例：`upgrade-system-command-4.md`、`feature-auth-module-1.md`
- 文件必须是有效的 Markdown，并具有正确的 front matter 结构

## 强制模板结构

所有实施计划都必须严格遵循以下模板。每个章节都是必填项，并且必须填入具体、可执行的内容。AI 代理在执行前必须验证模板是否合规。

## 模板校验规则

- 所有 front matter 字段都必须存在且格式正确
- 所有章节标题都必须完全匹配（区分大小写）
- 所有标识符前缀都必须遵循指定格式
- 表格必须包含所有必需列
- 最终输出中不得残留占位文本

## 状态

实施计划的状态必须在 front matter 中清晰定义，并反映计划的当前状态。状态只能是以下之一（括号中为 `status_color`）：`Completed`（亮绿色徽章）、`In progress`（黄色徽章）、`Planned`（蓝色徽章）、`Deprecated`（红色徽章）或 `On Hold`（橙色徽章）。该状态还应在引言部分以徽章形式展示。

```md
---
goal: [Concise Title Describing the Package Implementation Plan's Goal]
version: [Optional: e.g., 1.0, Date]
date_created: [YYYY-MM-DD]
last_updated: [Optional: YYYY-MM-DD]
owner: [Optional: Team/Individual responsible for this spec]
status: 'Completed'|'In progress'|'Planned'|'Deprecated'|'On Hold'
tags: [Optional: List of relevant tags or categories, e.g., `feature`, `upgrade`, `chore`, `architecture`, `migration`, `bug` etc]
---

# Introduction

![Status: <status>](https://img.shields.io/badge/status-<status>-<status_color>)

[A short concise introduction to the plan and the goal it is intended to achieve.]

## 1. Requirements & Constraints

[Explicitly list all requirements & constraints that affect the plan and constrain how it is implemented. Use bullet points or tables for clarity.]

- **REQ-001**: Requirement 1
- **SEC-001**: Security Requirement 1
- **[3 LETTERS]-001**: Other Requirement 1
- **CON-001**: Constraint 1
- **GUD-001**: Guideline 1
- **PAT-001**: Pattern to follow 1

## 2. Implementation Steps

### Implementation Phase 1

- GOAL-001: [Describe the goal of this phase, e.g., "Implement feature X", "Refactor module Y", etc.]

| Task | Description | Completed | Date |
|------|-------------|-----------|------|
| TASK-001 | Description of task 1 | ✅ | 2025-04-25 |
| TASK-002 | Description of task 2 | |  |
| TASK-003 | Description of task 3 | |  |

### Implementation Phase 2

- GOAL-002: [Describe the goal of this phase, e.g., "Implement feature X", "Refactor module Y", etc.]

| Task | Description | Completed | Date |
|------|-------------|-----------|------|
| TASK-004 | Description of task 4 | |  |
| TASK-005 | Description of task 5 | |  |
| TASK-006 | Description of task 6 | |  |

## 3. Alternatives

[A bullet point list of any alternative approaches that were considered and why they were not chosen. This helps to provide context and rationale for the chosen approach.]

- **ALT-001**: Alternative approach 1
- **ALT-002**: Alternative approach 2

## 4. Dependencies

[List any dependencies that need to be addressed, such as libraries, frameworks, or other components that the plan relies on.]

- **DEP-001**: Dependency 1
- **DEP-002**: Dependency 2

## 5. Files

[List the files that will be affected by the feature or refactoring task.]

- **FILE-001**: Description of file 1
- **FILE-002**: Description of file 2

## 6. Testing

[List the tests that need to be implemented to verify the feature or refactoring task.]

- **TEST-001**: Description of test 1
- **TEST-002**: Description of test 2

## 7. Risks & Assumptions

[List any risks or assumptions related to the implementation of the plan.]

- **RISK-001**: Risk 1
- **ASSUMPTION-001**: Assumption 1

## 8. Related Specifications / Further Reading

[Link to related spec 1]
[Link to relevant external documentation]
```
