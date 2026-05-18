# TODOS.md Format Reference

`TODOS.md` 的目标是让几个月后接手的人也能快速看懂上下文，而不是只留下标题。

## 文件结构

```markdown
# TODOS

## <Skill/Component>
<items sorted P0 first, then P1, P2, P3, P4>

## Completed
<finished items with completion annotation>
```

## TODO 条目格式

每个条目都是一个三级标题，至少包含：

- `What`
- `Why`
- `Context`
- `Effort`
- `Priority`

可选：

- `Depends on`
- `Blocked by`

示例：

```markdown
### 修复 review 对枚举值漏判

**What:** 让 review 覆盖新增枚举值的所有消费者。
**Why:** 避免新增状态上线后有消费者悄悄落到默认分支。
**Context:** 当前 diff 给后端加了新状态，但前端和报表层还没同步。
**Effort:** M
**Priority:** P1
**Depends on:** None
```

## 优先级定义

- `P0`，阻断下次发布
- `P1`，本周期应完成
- `P2`，重要，但可排在 P0 / P1 之后
- `P3`，有价值，但不急
- `P4`，长期想法

## Completed 条目

完成后把原条目移动到 `## Completed`，并补：

```markdown
**Completed:** vX.Y.Z (YYYY-MM-DD)
```
