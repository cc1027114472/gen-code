---
name: executing-plans
description: Use when you have a written implementation plan to execute in a separate session with review checkpoints
---

# 执行计划

## 概览

加载计划，进行批判性审查，执行所有任务，并在完成时汇报。

**开始时要声明：** “我正在使用 `executing-plans` skill 来实现这个计划。”

**注意：** 告诉你的人类伙伴，Superpowers 在能够访问子代理时效果会好得多。如果在支持子代理的平台上运行（例如 Claude Code 或 Codex），它的工作质量会显著更高。如果子代理可用，请使用 `superpowers:subagent-driven-development`，而不是这个 skill。

## 这个流程

### 步骤 1：加载并审查计划
1. 读取计划文件
2. 进行批判性审查 - 识别对这个计划的任何问题或担忧
3. 如果有担忧：在开始之前向你的人类伙伴提出
4. 如果没有担忧：创建 `TodoWrite` 并继续

### 步骤 2：执行任务

对每个任务：
1. 标记为 `in_progress`
2. 严格遵循每个步骤（计划已经被拆成细粒度步骤）
3. 按要求运行验证
4. 标记为 `completed`

### 步骤 3：完成开发

在所有任务都完成并验证之后：
- 声明： “我正在使用 `finishing-a-development-branch` skill 来完成这项工作。”
- **必需的子 skill：** 使用 `superpowers:finishing-a-development-branch`
- 遵循那个 skill 来验证测试、呈现选项并执行选择

## 何时停止并寻求帮助

**在以下情况下，立即停止执行：**
- 遇到阻塞（缺失依赖、测试失败、指令不清）
- 计划存在阻止开始的关键缺口
- 你不理解某条指令
- 验证反复失败

**要请求澄清，而不是猜。**

## 何时回到更早的步骤

**在以下情况下，回到审查（步骤 1）：**
- 伙伴根据你的反馈更新了计划
- 基本方法需要重新思考

**不要硬顶着阻塞往前冲** - 停下来提问。

## 请记住
- 先批判性审查计划
- 严格按照计划步骤执行
- 不要跳过验证
- 当计划要求引用 skill 时就要引用
- 被阻塞时要停下来，不要猜
- 没有用户明确同意时，绝不要在 `main/master` 分支上开始实现

## 集成

**必需的工作流 skills：**
- **`superpowers:using-git-worktrees`** - 必需：开始前设置隔离工作区
- **`superpowers:writing-plans`** - 创建由这个 skill 执行的计划
- **`superpowers:finishing-a-development-branch`** - 在所有任务完成后完成开发
