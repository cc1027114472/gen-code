---
name: codex-issue-digest
description: Run a GitHub issue digest for openai/codex by feature-area labels, all areas, and configurable time windows. Use when asked to summarize recent Codex bug reports or enhancement requests, especially for owner-specific labels such as tui, exec, app, or similar areas.
---

# Codex Issue 摘要

## 目标

默认针对前 24 小时内、所请求功能区域标签对应的 `openai/codex` issues，产出一份以标题优先、洞察导向的摘要。当用户要求其他时长时要遵从，例如“过去一周”或“48 小时”。默认只返回摘要；仅在被请求时包含细节。

只包含当前同时具有 `bug` 或 `enhancement`，以及至少一个所请求 owner 标签的 issue。如果用户要求所有区域或所有标签，则收集所有标签范围内的 `bug`/`enhancement` issue。

## 输入

- 功能区域标签，例如 `tui exec`
- 使用 `all areas` / `all labels` 扫描当前所有功能标签
- 可选的仓库覆盖，默认是 `openai/codex`
- 可选的时间窗口，默认前 24 小时；示例：`48h`、`7d`、`1w`、`past week`

## 工作流

1. 从当前的 Codex 仓库检出中运行收集器：

```bash
python3 .codex/skills/codex-issue-digest/scripts/collect_issue_digest.py --labels tui exec --window-hours 24
```

当用户要求非默认时长时，使用 `--window "past week"` 或 `--window-hours 168`。当用户说所有区域或所有标签时，使用 `--all-labels`。

2. 使用 JSON 作为事实来源。它包含新 issue、新 issue 评论、新 reaction/upvote、当前标签、当前 reaction 计数、可供模型使用的 `summary_inputs`，以及详细的 `digest_rows`。
3. 根据用户请求选择输出模式：
   - 默认模式：以 `## Summary` 开始报告，不输出 `## Details`。
   - 细节前置模式：如果用户要求细节、表格、完整 digest、“include details”或类似内容，以 `## Summary` 开始，然后包含 `## Details`。
   - 追问细节模式：如果用户在仅摘要 digest 之后要求更多细节，在现有收集器 JSON 仍可用时从中生成 `## Details`；否则重新运行收集器。
4. 在 `## Summary` 中，撰写以标题优先的执行摘要：
   - `## Summary` 下第一行非空行必须是一条单行标题或判断，而不是项目符号。即使读者只读到这里，它也应当有用。
   - 在平静的日子里，优先精确使用：`No major issues reported by users.` 当没有高关注行、没有新出现的重复主题，且没有需要 owner 采取行动的事项时，使用这句。
   - 当用户正在反馈值得注意的问题时，让标题点明数量或主题，例如 `Two issues are being surfaced by users:`。
   - 在活跃标题正下方，只列出驱动关注度的问题或主题，并按重要性排序。每一行在有 `attention_marker` 时以该行的 `attention_marker` 开头，然后是简洁、owner 可读的描述和行内 issue 引用。
   - 将 `🔥🔥` 视为值得上标题，将 `🔥` 视为高于常态。不要自行添加火焰 emoji；只复制该行的 `attention_marker`。
   - 标题之后的任何额外摘要细节都控制在 1-3 条简洁行内，并且只在它增加了与决策相关的注意事项、重复主题或 owner 行动时才写。
   - 不要在 `## Summary` 中包含常规计数、宽泛统计或低信号表格摘要，除非它们会改变标题。把元数据和可选计数放到 `## Details` 或页脚中。
   - 在默认模式下，以简洁提示结束报告，例如 `Want details? I can expand this into the issue table.` 让它与摘要标题分开，这样标题可以保持干净。
   - 自行根据 `summary_inputs` 聚类并命名主题；收集器有意不对 issue 类别做硬编码。
   - 仅当这些 issue 确实共享同一个产品问题时才使用聚类。如果几个 issue 只是共享一个宽泛的平台或标签，请分别描述它们。
   - 不要仅仅因为某个重复主题中的单个 issue 低于细节表格的截断线，就省略该主题。多个相似报告应被点出为重复出现的客户关切。
   - 对于单 issue 行，直接概括该关切，而不是称之为一个聚类。
   - 使用每个相关行 `ref_markdown` 中的行内编号 issue 链接。
   - 平静摘要示例：

```markdown
## Summary
No major issues reported by users.

Source: collector v5, git `abc123def456`, window `2026-04-27T00:00:00Z` to `2026-04-28T00:00:00Z`.
Want details? I can expand this into the issue table.
```

   - 活跃摘要示例：

```markdown
## Summary
Two issues are being surfaced by users:
🔥🔥 Terminal launch hangs on startup [1](https://github.com/openai/codex/issues/123)
🔥 Resume switches model providers unexpectedly [2](https://github.com/openai/codex/issues/456)

Source: collector v5, git `abc123def456`, window `2026-04-27T00:00:00Z` to `2026-04-28T00:00:00Z`.
Want details? I can expand this into the issue table.
```
5. 在 `## Details` 中，当用户请求细节时，仅在有用的情况下包含紧凑表格：
   - 优先使用 `digest_rows` 中的行；使用每一行的 `ref_markdown` 包含 `Refs` 列。
   - 保持表格简短；当摘要已经覆盖低信号行时，将其省略。
   - 使用紧凑列，例如 marker、area、type、description、interactions 和 refs。
   - `Description` 单元格应是简短、owner 可读的短语。使用行的 `description`、标题、正文摘录和最新评论，但当原始 GitHub issue 标题包含偶然细节时，不要机械地照抄。
   - 当没有有意义的信号时，写一句清晰的平静/无关切说明。
6. 严格按原样使用 JSON 中的 `attention_marker`。普通行为空，高关注行为 `🔥`，非常高关注行为 `🔥🔥`。实际阈值在 `attention_thresholds` 中。
7. 当某一行或项目符号指向 issue 时，使用行内编号引用，例如 `Compaction bugs [1](https://github.com/openai/codex/issues/123), [2](https://github.com/openai/codex/issues/456)`。不要额外添加脚注章节。
8. 将 `interactions` 标注为 `Interactions`；它统计在请求时间窗口内创建新 issue、添加新评论或作出 reaction 的唯一 GitHub 真人用户。同一用户在同一 issue 上的多次发帖/reaction 只计一次。
9. 用一条紧凑的来源行提到收集器的 `script_version`、仓库检出的 `git_head` 和时间窗口。在默认模式中，把它放在细节提示之前，这样最后一行仍然是在询问用户是否想看细节。在细节前置模式中，它可以作为页脚。

## Reaction 处理

收集器使用 GitHub reactions 端点，其中包含 `created_at`，以统计在 digest 时间窗口内、针对已补全 issue 创建的 reactions。它同时报告窗口内 reaction 计数和当前 reaction 总数。将当前 reaction 总数视为持续参与度，将 `new_reactions` / `new_upvotes` 视为窗口内活动。

默认情况下，收集器使用 `since=<window start>` 抓取 issue 评论，并限制每个 issue 的评论页数。这可以避免非常长的历史线程主导一次 digest 运行，并让报告聚焦于最近的发言。仅当完整评论历史比运行时间更重要时才使用 `--fetch-all-comments`。

GitHub issue 搜索仍然以 issue 的 `updated_at` 为种子，因此如果 reaction 不会推动 `updated_at`，纯 reaction-only 的 issue 可能会被漏掉。要覆盖每一个纯 reaction 情况，需要持久化快照存储或对带标签的 issue 做更广泛扫描。

## 关注度标记

收集器会根据请求的时间窗口缩放关注度标记。基线是在 24 小时内，`🔥` 对应 5 个唯一真人用户，`🔥🔥` 对应 10 个唯一真人用户；更长或更短的窗口会按线性比例缩放这些阈值并向上取整。例如，一周报告使用 35 和 70 次互动。唯一真人用户是指在该窗口内创建新 issue、撰写新评论或作出 reaction（包括 upvote）的用户。同一用户在同一 issue 上的多次操作只计一次。机器人发帖和机器人 reaction 会被排除。在行文中，应将其解释为高用户互动，而不是点名 emoji。

## 新鲜度

自动化应从包含此 skill 的仓库检出中运行。对于共享的日常使用，优先采用以下模式之一：

- 在自动化开始之前，先在一个已刷新的检出中运行自动化，例如执行 `git pull --ff-only`。
- 如果自动化无法安全地修改检出，让它从收集器输出中报告当前 `git_head`，以便读者知道是哪一个 skill/script 版本生成了该 digest。

## 示例 Owner 提示词

```text
Use $codex-issue-digest to run the Codex issue digest for labels tui and exec over the previous 24 hours.
```

```text
Use $codex-issue-digest to run the Codex issue digest for all areas over the past week.
```

## 验证

针对最近的 issues 对收集器执行 dry run：

```bash
python3 .codex/skills/codex-issue-digest/scripts/collect_issue_digest.py --labels tui exec --window-hours 24
```

```bash
python3 .codex/skills/codex-issue-digest/scripts/collect_issue_digest.py --all-labels --window "past week" --limit-issues 10
```

运行聚焦的脚本测试：

```bash
pytest .codex/skills/codex-issue-digest/scripts/test_collect_issue_digest.py
```
