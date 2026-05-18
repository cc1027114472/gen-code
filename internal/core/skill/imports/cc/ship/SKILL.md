---
name: ship
description: 发布工作流，负责同步基线、运行测试、整理版本与变更日志、提交、推送并创建 PR。
allowed-tools:
  - Bash
  - Read
  - Write
  - Edit
  - Grep
  - Glob
  - Agent
  - AskUserQuestion
  - WebSearch
---

# `ship` 发布工作流

`ship` 用来把当前分支整理成“可以发起 PR 的完整发布候选”。它会检测并同步 base branch，运行测试，检查 diff，更新 `VERSION` 和 `CHANGELOG`，提交变更、推送远端并创建 PR。

这个项目内 copied skill 继续保留 gstack-heavy preamble、发布链说明、`CLAUDE.md` 与外部 Git / PR / review / deploy 依赖说明。当前完成的是 copied-skill 的静态治理、中文化和 runtime-visible promotion，不代表真实 GitHub、远端仓库、PR 或发布链已经做了执行级验收。

## 预执行 Preamble

```bash
_UPD=$(~/.claude/skills/gstack/bin/gstack-update-check 2>/dev/null || .claude/skills/gstack/bin/gstack-update-check 2>/dev/null || true)
[ -n "$_UPD" ] && echo "$_UPD" || true
mkdir -p ~/.gstack/sessions
touch ~/.gstack/sessions/"$PPID"
_SESSIONS=$(find ~/.gstack/sessions -mmin -120 -type f 2>/dev/null | wc -l | tr -d ' ')
find ~/.gstack/sessions -mmin +120 -type f -exec rm {} + 2>/dev/null || true
_CONTRIB=$(~/.claude/skills/gstack/bin/gstack-config get gstack_contributor 2>/dev/null || true)
_PROACTIVE=$(~/.claude/skills/gstack/bin/gstack-config get proactive 2>/dev/null || echo "true")
_PROACTIVE_PROMPTED=$([ -f ~/.gstack/.proactive-prompted ] && echo "yes" || echo "no")
_BRANCH=$(git branch --show-current 2>/dev/null || echo "unknown")
echo "BRANCH: $_BRANCH"
_SKILL_PREFIX=$(~/.claude/skills/gstack/bin/gstack-config get skill_prefix 2>/dev/null || echo "false")
echo "PROACTIVE: $_PROACTIVE"
echo "PROACTIVE_PROMPTED: $_PROACTIVE_PROMPTED"
echo "SKILL_PREFIX: $_SKILL_PREFIX"
source <(~/.claude/skills/gstack/bin/gstack-repo-mode 2>/dev/null) || true
REPO_MODE=${REPO_MODE:-unknown}
echo "REPO_MODE: $REPO_MODE"
_LAKE_SEEN=$([ -f ~/.gstack/.completeness-intro-seen ] && echo "yes" || echo "no")
echo "LAKE_INTRO: $_LAKE_SEEN"
_TEL=$(~/.claude/skills/gstack/bin/gstack-config get telemetry 2>/dev/null || true)
_TEL_PROMPTED=$([ -f ~/.gstack/.telemetry-prompted ] && echo "yes" || echo "no")
_TEL_START=$(date +%s)
_SESSION_ID="$$-$(date +%s)"
echo "TELEMETRY: ${_TEL:-off}"
echo "TEL_PROMPTED: $_TEL_PROMPTED"
mkdir -p ~/.gstack/analytics
if [ "${_TEL:-off}" != "off" ]; then
  echo '{"skill":"ship","ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","repo":"'$(basename "$(git rev-parse --show-toplevel 2>/dev/null)" 2>/dev/null || echo "unknown")'"}'  >> ~/.gstack/analytics/skill-usage.jsonl 2>/dev/null || true
fi
for _PF in $(find ~/.gstack/analytics -maxdepth 1 -name '.pending-*' 2>/dev/null); do
  if [ -f "$_PF" ]; then
    if [ "$_TEL" != "off" ] && [ -x "~/.claude/skills/gstack/bin/gstack-telemetry-log" ]; then
      ~/.claude/skills/gstack/bin/gstack-telemetry-log --event-type skill_run --skill _pending_finalize --outcome unknown --session-id "$_SESSION_ID" 2>/dev/null || true
    fi
    rm -f "$_PF" 2>/dev/null || true
  fi
  break
done
eval "$(~/.claude/skills/gstack/bin/gstack-slug 2>/dev/null)" 2>/dev/null || true
_LEARN_FILE="${GSTACK_HOME:-$HOME/.gstack}/projects/${SLUG:-unknown}/learnings.jsonl"
if [ -f "$_LEARN_FILE" ]; then
  _LEARN_COUNT=$(wc -l < "$_LEARN_FILE" 2>/dev/null | tr -d ' ')
  echo "LEARNINGS: $_LEARN_COUNT entries loaded"
else
  echo "LEARNINGS: 0"
fi
_HAS_ROUTING="no"
if [ -f CLAUDE.md ] && grep -q "## Skill routing" CLAUDE.md 2>/dev/null; then
  _HAS_ROUTING="yes"
fi
_ROUTING_DECLINED=$(~/.claude/skills/gstack/bin/gstack-config get routing_declined 2>/dev/null || echo "false")
echo "HAS_ROUTING: $_HAS_ROUTING"
echo "ROUTING_DECLINED: $_ROUTING_DECLINED"
```

## Preamble 行为说明

- `PROACTIVE`、`SKILL_PREFIX`、升级提示、telemetry、routing rules、contributor mode 等初始化逻辑继续沿用原始 gstack preamble。
- 这里仍然依赖 `~/.claude/skills/gstack/bin/*`、`~/.gstack/*` 和 `CLAUDE.md`，本阶段不把它们重写成新的 standalone 运行机制。
- 本阶段治理目标是 clean UTF-8 的 copied truth，不是替换原始 gstack 发布生态。

## 语气与风格

- 先说结果，再说证据，最后说动作。
- 发布链上的问题要直接指出，不要把真实阻塞讲得模糊。
- 始终把用户体验、回归风险、版本一致性和可发布性放在第一位。

## AskUserQuestion 约束

- 先重申项目、当前分支和当前 ship 任务。
- 用足够简单的话解释风险或取舍，不堆 Git 内部术语。
- 给出明确推荐，并同时标注 human / CC 成本。
- 只有在会改变发布行为或需要真实判断的分叉点才 AskUserQuestion，机械步骤优先自动推进。

## `ship` 工作流

`ship` 的目标是把当前分支推进到“可以发起 PR 的完整候选”状态，而不是直接绕过 review 和 deploy 链路。

### Step 1：确认当前分支与工作树

- 读取当前分支、base branch 和工作树状态。
- 如果工作树是脏的，要先向用户说明会影响发布整理，并决定继续、分离或暂停。
- 如果用户明确只想创建 PR，也不能跳过后面的测试与 review 基线检查。

### Step 2：同步 base branch

- 检测并同步 base branch，确保当前分支不是在过旧基线上发起 PR。
- 如果同步 base branch 后出现冲突，停止并按完成状态协议上报 `BLOCKED` 或 `NEEDS_CONTEXT`。

### Step 3：运行测试与 review 前置检查

- 运行与当前项目相匹配的测试命令。
- 如果项目已经有 `/review` 或等价审查结果，要优先读取并复用；没有时就按原工作流继续执行 diff review。
- 如有明显测试或 review 阻塞，不能假装“可发布”。

### Step 4：检查版本与变更日志

- 如果仓库使用 `VERSION`、`CHANGELOG` 或其他发布记录，按原工作流更新它们。
- 版本与变更日志要和这次将要发起的 PR 内容一致，不能出现“代码改了，版本或说明没跟上”的情况。

### Step 5：整理提交

- 只把当前 ship 目标相关的变更纳入提交。
- 保持提交信息与发布内容一致，不要把无关改动混进发布候选。

### Step 6：推送并创建 PR

- 推送当前分支到远端。
- 创建 PR，并把标题、描述和 review 上下文整理清楚。
- 如果工作流中要求补充 issue、风险说明、测试证据或 release notes，要一起带上。

### Step 7：总结

- 总结当前分支、base branch、测试结果、review 状态、版本变化、`CHANGELOG` 变化和 PR 结果。
- 明确说明后续应由 `/land-and-deploy` 接手 merge、等待 CI / deploy 和生产验证。

## 完成状态协议

完成 skill 时只能使用以下状态之一：

- **DONE**：所有步骤完成且每条结论都有证据。
- **DONE_WITH_CONCERNS**：完成了，但仍有用户必须知道的风险或缺口。
- **BLOCKED**：无法继续，并说明被什么阻塞、已经尝试了什么。
- **NEEDS_CONTEXT**：缺少继续所需的信息，并明确指出需要什么。

### 升级处理

- 如果连续三次尝试仍无法解决同一个问题，停止并上报。
- 如果涉及安全敏感修改而你不确定，停止并上报。
- 如果范围已经超出你能验证的边界，停止并上报。

上报格式保持：

```text
STATUS: BLOCKED | NEEDS_CONTEXT
REASON: [1-2 sentences]
ATTEMPTED: [what you tried]
RECOMMENDATION: [what the user should do next]
```

## 重要规则

- 不要跳过测试、review、版本和 CHANGELOG 基线，只为了更快“发 PR”。
- 不要把 `ship` 改写成只做 commit 或只生成 PR 文案的弱化版本。
- 不要回显完整凭据、token 或密钥。
- capability verified 只说明 copied-skill 静态治理通过，不代表真实 GitHub、push、PR 创建或发布链执行已经验收。
