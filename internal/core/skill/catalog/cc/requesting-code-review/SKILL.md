---
name: requesting-code-review
description: Use when completing tasks, implementing major features, or before merging to verify work meets requirements
---

# 请求代码审查

派发 `superpowers:code-reviewer` 子代理，在问题扩散之前把它们拦下来。审查者拿到的是为评估精确构造的上下文 - 绝不是你这个会话的历史。这让审查者专注于工作产物，而不是你的思考过程，同时也保留了你自己的上下文以继续工作。

**核心原则：** 尽早审查，经常审查。

## 何时请求审查

**强制：**
- 在子代理驱动开发中，每完成一个任务之后
- 完成一个重大功能之后
- 合并到 `main` 之前

**可选但有价值：**
- 卡住的时候（获得新的视角）
- 重构之前（做基线检查）
- 修复复杂 bug 之后

## 如何请求

**1. 获取 git SHA：**
```bash
BASE_SHA=$(git rev-parse HEAD~1)  # 或 origin/main
HEAD_SHA=$(git rev-parse HEAD)
```

**2. 派发 `code-reviewer` 子代理：**

使用 `Task` 工具，类型设为 `superpowers:code-reviewer`，并填写 `code-reviewer.md` 里的模板

**占位符：**
- `{WHAT_WAS_IMPLEMENTED}` - 你刚刚构建了什么
- `{PLAN_OR_REQUIREMENTS}` - 它本应该做什么
- `{BASE_SHA}` - 起始提交
- `{HEAD_SHA}` - 结束提交
- `{DESCRIPTION}` - 简短总结

**3. 根据反馈行动：**
- 立即修复 `Critical` 问题
- 在继续之前修复 `Important` 问题
- 记录 `Minor` 问题，稍后处理
- 如果审查者错了，就反驳（并给出理由）

## 示例

```
[刚完成任务 2：添加验证函数]

你：在继续之前，我先请求一次代码审查。

BASE_SHA=$(git log --oneline | grep "Task 1" | head -1 | awk '{print $1}')
HEAD_SHA=$(git rev-parse HEAD)

[派发 superpowers:code-reviewer 子代理]
  WHAT_WAS_IMPLEMENTED: 会话索引的验证与修复函数
  PLAN_OR_REQUIREMENTS: docs/superpowers/plans/deployment-plan.md 中的任务 2
  BASE_SHA: a7981ec
  HEAD_SHA: 3df7661
  DESCRIPTION: 新增了 verifyIndex() 和 repairIndex()，覆盖 4 类问题

[子代理返回]：
  Strengths: 架构清晰，测试真实
  Issues:
    Important: 缺少进度指示器
    Minor: 用于报告间隔的魔法数字（100）
  Assessment: 可以继续

你： [修复进度指示器]
[继续任务 3]
```

## 与工作流的集成

**子代理驱动开发：**
- 每个任务之后都审查
- 在问题复合之前就把它们抓出来
- 修完再进入下一个任务

**执行计划：**
- 每个批次（3 个任务）之后审查
- 获取反馈，应用修复，继续

**临时式开发：**
- 合并之前审查
- 卡住时审查

## 危险信号

**绝不要：**
- 因为“这很简单”就跳过审查
- 忽略 `Critical` 问题
- 带着未修复的 `Important` 问题继续推进
- 和有效的技术反馈争辩

**如果审查者错了：**
- 用技术理由反驳
- 展示能证明它可工作的代码/测试
- 请求澄清

模板见：`requesting-code-review/code-reviewer.md`
