---
name: babysit-pr
description: Babysit a GitHub pull request after creation by continuously polling review comments, CI checks/workflow runs, and mergeability state until the PR is merged/closed or user help is required. Diagnose failures, retry likely flaky failures up to 3 times, auto-fix/push branch-related issues when appropriate, and keep watching open PRs so fresh review feedback is surfaced promptly. Use when the user asks Codex to monitor a PR, watch CI, handle review comments, or keep an eye on failures and feedback on an open PR.
---

# PR 保姆

## 目标
持续照看一个 PR，直到发生以下任一终止结果：

- PR 被合并或关闭。
- 某种情况需要用户协助（例如 CI 基础设施问题、在重试预算耗尽后仍反复出现的 flaky 失败、权限问题，或无法安全消除的歧义）。
- 可选的交接里程碑：PR 当前是绿色 + 可合并 + 评审干净。将此视为进度状态，而不是停止 watcher 的条件，这样在 PR 仍保持打开时，迟到的评审评论仍能被及时呈现。

不要仅仅因为某一次快照返回了 `idle` 且检查仍在等待中就停止。

## 输入
接受以下任一形式：

- 不带 PR 参数：从当前分支推断 PR（`--pr auto`）
- PR 编号
- PR ??

## 核心工作流

1. 当用户要求“monitor”/“watch”/“babysit”某个 PR 时，从 watcher 的持续模式（`--watch`）开始，除非你是有意在做一次性诊断快照。
2. 运行 watcher 脚本以获取 PR/评审/CI 状态快照（或消费 `--watch` 流式输出的每个快照）。
3. 检查 JSON 响应中的 `actions` 列表。
4. 如果存在 `diagnose_ci_failure`，检查失败的 run 日志并对失败进行分类。
5. 如果失败很可能由当前分支引起，在本地修补代码、提交并推送。不要去修补随机的 flaky 测试、CI 基础设施、依赖故障、runner 问题或其他与该分支无关的失败。
6. 如果存在 `process_review_comment`，检查被呈现的评审项，并决定是否处理它们。
7. 如果某条评审项是可执行且正确的，在本地修补代码、提交、推送，然后在修复已进入 GitHub 后，将相关评审线程/评论标记为已解决。
8. 不要对人类作者写下的评审评论/线程发回复，除非用户明确确认了精确回复内容。如果某条人工评审项不可执行、已被处理或并不成立，应将该项和建议回复呈现给用户，而不是直接在 GitHub 上回复。
9. 如果失败很可能是 flaky/无关的，并且存在 `retry_failed_checks`，使用 `--retry-failed-now` 重新运行失败的 job。
10. 如果同时存在可执行的评审反馈和 `retry_failed_checks`，优先处理评审反馈；新的提交会重新触发 CI，因此除非你有意推迟评审修改，否则不要在旧 SHA 上重新运行 flaky 检查。
11. 在每次循环中，在处理 CI 失败或可合并状态之前，先查看是否有新出现的评审反馈，然后连同 CI 一起验证可合并性 / 合并冲突状态（例如通过 `gh pr view`）。
12. 在任何推送或重跑操作之后，立刻返回第 1 步，并在更新后的 SHA/状态上继续轮询。
13. 如果你在暂停去 patch/commit/push 之前一直在使用 `--watch`，那么在推送之后要在同一轮中自行重新启动 `--watch`（不要等用户重新调用此 skill）。
14. 重复轮询，直到出现 `stop_pr_closed` 或遇到需要用户帮助的阻塞。绿色 + 评审干净 + 可合并的 PR 是进度里程碑，而不是在 PR 仍然打开时停止 watcher 的理由。
15. 维持终端/会话的所有权：当 babysitting 处于激活状态时，在同一轮中持续消费 watcher 输出；不要留下一个脱离的 `--watch` 进程在后台运行，然后把这一轮结束得像监控已完成一样。

## 命令

### 一次性快照

```bash
python3 .codex/skills/babysit-pr/scripts/gh_pr_watch.py --pr auto --once
```

### 持续监控（JSONL）

```bash
python3 .codex/skills/babysit-pr/scripts/gh_pr_watch.py --pr auto --watch
```

### 触发 flaky 重试循环（仅当 watcher 指示时）

```bash
python3 .codex/skills/babysit-pr/scripts/gh_pr_watch.py --pr auto --retry-failed-now
```

### 显式指定 PR 目标

```bash
python3 .codex/skills/babysit-pr/scripts/gh_pr_watch.py --pr <number-or-url> --once
```

## CI 失败分类
在决定是否重跑之前，使用 `gh` 命令检查失败的 run。

- `gh run view <run-id> --json jobs,name,workflowName,conclusion,status,url,headSha`
- `gh api repos/<owner>/<repo>/actions/runs/<run-id>/jobs -X GET -f per_page=100`
- `gh api repos/<owner>/<repo>/actions/jobs/<job-id>/logs > /tmp/codex-gh-job-<job-id>-logs.zip`
- 当整个 workflow run 完成后，将 `gh run view <run-id> --log-failed` 作为兜底方案

`gh run view --log-failed` 的作用域是 workflow run，而且在整个 run 完成之前，可能不会暴露 failed-job 日志。为了更快地诊断，先轮询该 run 的 jobs，并且一旦某个具体 job 已失败，就直接从 Actions job logs endpoint 拉取该 job 的日志。只要 GitHub 暴露了相关信息，watcher 就会在 `failed_jobs` 列表中为每个失败 job 提供其 `job_id` 和 `logs_endpoint`。

当 failed-job 日志指向已变更代码（变更区域中的编译/测试/lint/typecheck/snapshots/static analysis）时，优先将失败视为与分支相关。

当日志显示瞬时的基础设施/外部问题（超时、runner 供应失败、registry/network 故障、GitHub Actions 基础设施错误）时，优先将失败视为 flaky/无关。

不要尝试通过修改测试、构建脚本、CI 配置、依赖版本锁定或靠近基础设施的代码来修复 flaky/无关失败，除非日志清楚地将失败与 PR 分支关联起来。对于 flaky/无关失败，仅当 watcher 建议 `retry_failed_checks` 时才重跑；否则就等待或停止并请求用户帮助。

如果分类存在歧义，在选择重跑之前先执行一次手动诊断尝试。

阅读 `.codex/skills/babysit-pr/references/heuristics.md` 获取简明检查清单。

## 评审评论处理
watcher 会呈现来自以下来源的评审项：

- PR issue 评论
- 行内评审评论
- 评审提交（COMMENT / APPROVED / CHANGES_REQUESTED）

它有意呈现 Codex reviewer bot 的反馈（例如来自 `chatgpt-codex-connector[bot]` 的评论/评审），以及人类评审者的反馈。大多数无关的 bot 噪音仍应忽略。
出于安全考虑，watcher 只会自动呈现受信任的人类评审作者（例如仓库 OWNER/MEMBER/COLLABORATOR，以及已认证的操作者）和被批准的评审 bot，例如 Codex。
在一个全新的 watcher 状态文件上，现有的待处理评审反馈可能会被立即呈现（而不仅仅是监控开始后到来的评论）。这是有意为之，以免漏掉已经打开的评审评论。

当你同意某条评论且它是可执行的：

1. 在本地修补代码。
2. 使用 `codex: address PR review feedback (#<n>)` 提交。
3. 推送到 PR 的头部分支。
4. 推送成功后，将相关的 GitHub 评审线程/评论标记为已解决。
5. 立即在新的 SHA 上恢复监控（不要在报告完推送后停止）。
6. 如果监控之前运行在 `--watch` 模式下，那么在推送后要立刻在同一轮中重启 `--watch`；不要等用户再次发问。

不要自动对人类作者写下的 GitHub 评审评论/线程发布回复。如果你不同意某条人工评论、认为它不可执行/已处理，或需要回答某个问题，应将该项报告给用户并附上建议回复，在向 GitHub 发任何内容之前等待明确确认。如果用户批准回复，请以前缀 `[codex]` 开头，以清楚表明该回复是自动化的，而不是来自人类用户。
如果 watcher 之后再次呈现了你自己已经批准的回复，因为已认证操作者被视为受信任的评审作者，那么应将该自写项视为已处理，不要再次回复。
如果某条代码评审评论/线程在 GitHub 中已经被标记为已解决，则应将其视为不可执行并安全忽略，除非出现了新的未解决跟进反馈。

## Git 安全规则

- 只在 PR 的头部分支上工作。
- 避免破坏性的 git 命令。
- 除非为了恢复上下文确有必要，否则不要切换分支。
- 编辑前，检查是否存在无关的未提交更改。如果存在，停止并询问用户。
- 每次成功修复后，提交并执行 `git push`，然后重新运行 watcher。
- 如果你为了修复而中断了一个正在运行的 `--watch` 会话，那么在推送后必须立刻在同一轮中重启 `--watch`。
- 不要为同一个 PR/状态文件运行多个并发的 `--watch` 进程；保持一个 watcher 会话处于活动状态，并持续复用它，直到它停止或你有意重启它。
- 推送不是终止结果；除非满足严格停止条件，否则继续监控循环。

提交信息默认值：

- `codex: fix CI failure on PR #<n>`
- `codex: address PR review feedback (#<n>)`

## 监控循环模式
在实时 Codex 会话中使用以下循环：

1. 运行 `--once`。
2. 读取 `actions`。
3. 先检查 PR 现在是否已被合并或以其他方式关闭；如果是，报告该终止状态并立即停止轮询。
4. 检查 CI 摘要、新的评审项以及可合并性/冲突状态。
5. 诊断 CI 失败，并将其分类为分支相关或 flaky/无关。如果整体 run 仍在等待中，但 `failed_jobs` 已经包含某个失败 job，就获取该 job 的日志并立即诊断，而不是等整个 workflow run 结束。只有在失败与分支相关时才 patch。
6. 对于来自其他作者的每个已呈现评审项，如果它是可执行的，就 patch/commit/push 并解决它。如果它不可执行、已被处理，或需要书面答复，则将其与建议回复一起呈现给用户，而不是自动发帖。如果后续快照再次呈现了你自己已经批准的回复，将其视为信息性内容并继续，不要再次响应。
7. 当可执行的评审评论和 flaky 重跑同时存在时，先处理评审评论；如果评审修复需要提交，就推送它，并跳过在旧 SHA 上重新运行失败检查。
8. 只有当存在 `retry_failed_checks` 且你并不打算用评审/CI 修复提交替换当前 SHA 时，才重试失败检查。不要为了让 CI 变绿而因为无关的 flaky 或基础设施失败去修改代码。
9. 如果你推送了提交、解决了评审线程或触发了重跑，简要报告该动作并继续轮询（不要停止）。如果某条人工评审评论需要一个书面的 GitHub 回复，则停止并在发帖前请求确认。
10. 在评审修复推送之后，主动在同一轮中重启持续监控（`--watch`），除非已经达到严格停止条件。
11. 如果一切都已通过、可合并、没有被必需的评审批准阻塞，并且没有未处理的评审项，报告 PR 当前已准备好合并，但保持 watcher 继续运行，以便在 PR 仍然打开时快速呈现新评审评论。
12. 如果因需要用户帮助的问题而被阻塞（基础设施故障、flaky 重试耗尽、不明确的评审要求、权限问题），报告该阻塞并停止。
13. 否则按照下面的轮询节奏休眠并重复。

当用户明确要求监控/watch/babysit 某个 PR 时，优先使用 `--watch`，这样轮询会通过一条命令自主持续进行。只有在调试、本地测试，或用户明确要求一次性检查时，才使用重复的 `--once` 快照。
不要停下来询问用户是否继续轮询；在满足严格停止条件或用户明确中断之前，自主继续。
不要在评审修复推送后仅仅因为创建了新的 SHA 就把控制权交还给用户；重启 watcher 并重新进入轮询循环是同一个 babysitting 任务的一部分。
如果 `--watch` 进程仍在运行，且尚未达到严格停止条件，那么 babysitting 任务仍在进行中；继续流式读取/消费 watcher 输出，而不是结束这一轮。

## 轮询节奏
保持积极的评审轮询，并且即使在 CI 变绿后也继续监控：

- 当 CI 不是绿色时（pending/running/queued 或失败中）：每 1 分钟轮询一次。
- 当 CI 变绿之后：只要 PR 仍保持打开，就按基础节奏继续轮询，这样新发布的评审评论能够被及时呈现，而不是等待一个很长的绿色状态退避时间。
- 每当任何内容发生变化（新的提交/SHA、检查状态变化、新评审评论、可合并性变化、评审决定变化）时，立即重置节奏。
- 如果 CI 再次不绿（新提交、重跑或回归）：保持基础轮询节奏。
- 如果任一轮询显示 PR 已合并或以其他方式关闭：立即停止轮询并报告该终止状态。

## 停止条件（严格）
只有在以下任一情况成立时才停止：

- PR 已合并或关闭（在某次轮询/快照确认后立即停止）。
- 需要用户介入，且 Codex 无法独自安全继续。

在以下情况下继续轮询：

- `actions` 只包含 `idle`，但检查仍在等待中。
- CI 仍在运行/排队。
- 评审状态平静，但 CI 不是终止状态。
- CI 是绿色，但可合并性未知/等待中。
- CI 是绿色且可合并，但 PR 仍然打开，而你在等待可能出现的新评审评论或合并冲突变化。
- PR 是绿色，但被评审批准阻塞（`REVIEW_REQUIRED` / 类似状态）；按基础节奏继续轮询，并呈现任何新的评审评论，而无需询问是否继续监控。

## 输出预期
在监控过程中提供简洁的进度更新，并给出包含以下内容的最终总结：

- 在长时间没有变化的监控期间，不要在每次轮询时都输出完整更新；只总结状态变化，并偶尔给出心跳式更新。
- 将推送确认、中间 CI 快照、ready-to-merge 快照和评审动作更新仅视为进度更新；除非满足严格停止条件，否则不要输出最终总结，也不要结束 babysitting 会话。
- 用户要求“monitor”并不意味着做几次示例轮询就算完成；保持循环，直到达到严格停止条件或用户明确中断。
- 评审修复提交 + 推送不是完成事件；在同一轮中立刻恢复实时监控（`--watch`），并继续报告进度更新。
- 当当前 SHA 的 CI 第一次转为全绿时，发送一次性庆祝式进度更新（不要在每次绿色轮询时重复）。首选风格：`🚀 CI 已全部变绿！33/33 通过。仍在继续观察评审批准情况。`
- 当 watcher 终端仍在运行时，不要发送最终总结，除非 watcher 已发出/确认了严格停止条件；否则继续给出进度更新。

- 最终 PR SHA
- CI 状态摘要
- 可合并性 / 冲突状态
- 已推送的修复
- 已使用的 flaky 重试轮次
- 剩余未解决的失败或评审评论

## 参考资料

- 启发式与决策树：`.codex/skills/babysit-pr/references/heuristics.md`
- watcher 使用的 GitHub CLI/API 细节：`.codex/skills/babysit-pr/references/github-api-notes.md`
