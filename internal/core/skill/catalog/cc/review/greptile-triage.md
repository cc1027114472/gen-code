# Greptile Comment Triage

这个文件定义 `/review` 与 `/ship` 共用的 Greptile 评论拉取、分类、回复和历史记录规则。

## 分类

- `VALID & ACTIONABLE`，是真问题，且当前代码里仍存在。
- `VALID BUT ALREADY FIXED`，是真问题，但已经在当前分支后续提交中修掉。
- `FALSE POSITIVE`，评论误解了代码或把噪音当成问题。
- `SUPPRESSED`，已经被历史 suppressions 规则过滤。

## 处理原则

- line-level 评论要读对应文件上下文。
- top-level 评论要结合完整 diff 一起判断。
- 不能因为是机器人评论就跳过证据核对。

## 回复模板

### Fixed

```text
**Fixed** in `<commit-sha>`.

```diff
- old
+ new
```

**Why:** 一句话解释问题和修法。
```

### Already fixed

```text
**Already fixed** in `<commit-sha>`.

**What was done:** 用 1 到 2 句话说明后续提交已经如何修掉它。
```

### False positive

```text
**Not a bug.** 先直接说明为什么它不是 bug。

**Evidence:**
- 给出具体代码证据
- 必要时指出真正的防护点或调用链

**Suggested re-rank:** 如果原评论严重级别不对，明确建议降级。
```

## 历史记录

- 回复或分类结果要写入项目级与全局 `greptile-history.md`。
- 记录格式：`<date> | <repo> | <type> | <file-pattern> | <category>`。

## 升级策略

- 如果同一线程上已经有过 GStack 回复，而 Greptile 又重复标记，再用更强证据和更直接的口吻回复。
- 如果无法可靠识别线程上下文，默认使用第一层级模板，不做过度升级。
