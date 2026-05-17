---
name: writing-plans
description: 在你已经有一个规格说明或多步骤任务需求时使用，必须先于动手改代码
---

# 编写计划

## 概述

写出完整的实现计划，假设执行它的工程师对我们的代码库完全没有上下文，而且品味还不太可靠。把他们需要知道的一切都写清楚：每个任务要改哪些文件、需要参考哪些代码、测试、文档，以及该如何测试。把整个计划拆成一口一口能吃下去的小任务。DRY。YAGNI。TDD。频繁提交。

假设他们是熟练的开发者，但几乎不了解我们的工具链或问题领域。假设他们也不太懂好的测试设计。

**开始时要声明：** “我正在使用 writing-plans skill 来创建实现计划。”

**上下文：** 这应当在一个专用 worktree 中进行（由 brainstorming skill 创建）。

**计划保存位置：** `docs/superpowers/plans/YYYY-MM-DD-<feature-name>.md`
- （如果用户对计划存放位置有偏好，以用户偏好为准）

## 范围检查

如果这个规格覆盖了多个相互独立的子系统，那么它本应该在 brainstorming 阶段就被拆成多个子项目规格。如果还没拆，建议把它拆成多个独立计划——每个子系统一个。每个计划都应该能单独产出可运行、可测试的软件。

## 文件组织结构

在定义任务之前，先梳理将要创建或修改哪些文件，以及每个文件各自负责什么。拆解决策会在这里被锁定。

- 设计边界清晰、接口明确的单元。每个文件应当只有一个清晰职责。
- 你对能一次放进上下文里的代码推理得最好，而当文件更聚焦时，你的改动也更可靠。优先选择小而聚焦的文件，而不是做太多事的大文件。
- 会一起变化的文件应该放在一起。按职责拆分，而不是按技术层拆分。
- 在现有代码库里，要遵循既有模式。如果代码库本来就使用大文件，不要单方面重构——但如果你正在改的文件已经变得难以驾驭，把拆分写进计划是合理的。

这个结构会决定任务拆解方式。每个任务都应该产出独立看也成立的、自包含的变更。

## 一口大小的任务粒度

**每一步只做一个动作（2-5 分钟）：**
- “编写失败测试”——一步
- “运行它，确认它会失败”——一步
- “实现最小代码让测试通过”——一步
- “运行测试并确认通过”——一步
- “提交”——一步

## 计划文档头部

**每一个计划都必须以这个头部开始：**

```markdown
# [Feature Name] Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** [One sentence describing what this builds]

**Architecture:** [2-3 sentences about approach]

**Tech Stack:** [Key technologies/libraries]

---
```

## 任务结构

````markdown
### Task N: [Component Name]

**Files:**
- Create: `exact/path/to/file.py`
- Modify: `exact/path/to/existing.py:123-145`
- Test: `tests/exact/path/to/test.py`

- [ ] **Step 1: Write the failing test**

```python
def test_specific_behavior():
    result = function(input)
    assert result == expected
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/path/test.py::test_name -v`
Expected: FAIL with "function not defined"

- [ ] **Step 3: Write minimal implementation**

```python
def function(input):
    return expected
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/path/test.py::test_name -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/path/test.py src/path/file.py
git commit -m "feat: add specific feature"
```
````

## 不要留占位符

每一步都必须包含工程师真正需要的实际内容。以下这些都属于**计划失败**——绝不要这样写：
- “TBD”、“TODO”、“后续实现”、“补全细节”
- “加入适当的错误处理” / “加入校验” / “处理边界情况”
- “为上面的内容写测试” （却不提供实际测试代码）
- “类似任务 N” （代码要重复写——工程师可能会乱序阅读任务）
- 只描述做什么却不展示怎么做的步骤（代码类步骤必须带代码块）
- 引用了任何任务中都没有定义的类型、函数或方法

## 记住
- 始终给出精确文件路径
- 每一步都给出完整代码——只要这一步会改代码，就把代码展示出来
- 给出精确命令和预期输出
- DRY、YAGNI、TDD、频繁提交

## 自我评审

完整写完计划之后，用新的眼光再看一遍规格，并把计划与规格逐项对照。这是你自己执行的检查清单——不是派发给子代理做的。

**1. 规格覆盖：** 略读规格中的每一节 / 每一项要求。你能指出实现它的任务是哪一个吗？列出任何缺口。

**2. 占位符扫描：** 在你的计划里搜索危险信号——也就是上面“不要留占位符”一节中的那些模式。发现就修。

**3. 类型一致性：** 你在后续任务中使用的类型、方法签名、属性名，是否和前面任务中定义的一致？例如任务 3 里叫 `clearLayers()`，任务 7 里却写成 `clearFullLayers()`，这就是 bug。

如果你发现问题，直接原地修复。无需再做一次评审——修完继续即可。如果你发现某个规格要求没有对应任务，就把任务补上。

## 执行交接

保存计划后，提供执行方式选择：

**“计划已完成，并保存到 `docs/superpowers/plans/<filename>.md`。有两种执行方式：**

**1. 子代理驱动（推荐）** - 我为每个任务派发一个全新的子代理，在任务之间做评审，迭代快

**2. 内联执行** - 在当前会话中使用 executing-plans 执行这些任务，按批次执行并带检查点

**你希望用哪一种？**”

**如果选择子代理驱动：**
- **必需子技能：** 使用 `superpowers:subagent-driven-development`
- 每个任务使用全新子代理 + 两阶段评审

**如果选择内联执行：**
- **必需子技能：** 使用 `superpowers:executing-plans`
- 按批执行，并设置检查点用于评审
