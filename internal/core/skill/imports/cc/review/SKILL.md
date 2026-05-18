---
name: review
description: 预落地代码评审，围绕 diff、风险类别、自动修复与对抗式审查给出结构化结论。
allowed-tools:
  - Bash
  - Read
  - Edit
  - Write
  - Grep
  - Glob
  - Agent
  - AskUserQuestion
  - WebSearch
---

# `review` 预落地评审

预落地 PR 评审。读取当前分支相对 base branch 的 diff，围绕 SQL 与数据安全、LLM 信任边界、条件副作用、测试覆盖、设计 review、Greptile 评论和对抗式审查给出结构化结论。

这个项目内 copied skill 保留 gstack-heavy preamble、Fix-First Review、Greptile triage、Adversarial review、TODOS / 文档陈旧性检查等工作流语义，但当前完成的是 copied-skill 的静态治理、中文化与 runtime-visible promotion，不代表真实审查链路或外部模型执行已经做了执行级验收。

## Preamble（先运行）

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
  echo '{"skill":"review","ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","repo":"'$(basename "$(git rev-parse --show-toplevel 2>/dev/null)" 2>/dev/null || echo "unknown")'"}'  >> ~/.gstack/analytics/skill-usage.jsonl 2>/dev/null || true
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

- 如果 `PROACTIVE=false`，不要主动推荐或自动触发其他 gstack skill，除非用户明确要求。
- 如果 `SKILL_PREFIX=true`，涉及其他 gstack skill 时使用 `/gstack-` 前缀，磁盘路径规则保持不变。
- 如果看到 `UPGRADE_AVAILABLE`、`JUST_UPGRADED`、telemetry、routing 或 contributor 初始化分支，继续使用原 preamble 流程。
- 这些说明仍依赖 `~/.claude/skills/gstack/bin/*`、`~/.gstack/*` 和 `CLAUDE.md`，这里不重写成新的 project-local 运行机制。

## 语气与风格

- 先说结论，再说证据，再说修法。
- 任何问题都要尽量落到具体文件、具体行、具体风险，不要空泛评价。
- 审查是为了真实用户体验、数据安全、发布稳定性和维护成本，不是为了表演“看过了”。

## AskUserQuestion 约束

- 先重申项目、当前分支和当前 review 任务。
- 用足够简单的话描述风险，避免把解释写成框架内部实现导览。
- 给出明确推荐，同时标注 completeness 和 human / CC 成本。
- 对真正有分歧的修法才提问，机械性问题优先自动修。

## 核心工作流

### Step 1: 读取完整 diff

- 先读完整 `git diff origin/main`，避免评论已经在 diff 里被修掉的问题。
- 必要时结合提交历史、base branch 和文档上下文理解这次改动的真实意图。

### Step 2: 两遍评审

- Pass 1 聚焦 critical：SQL / 数据安全、竞争条件、LLM 信任边界、Shell 注入、枚举完整性。
- Pass 2 聚焦 informational：条件副作用、魔法值、死代码、测试缺口、设计 review、文档陈旧性、性能与分发等问题。
- 分类与具体检查项使用：
  - [预落地评审检查表](checklist.md)
  - [设计评审检查表](design-checklist.md)
  - [Greptile 评论分流说明](greptile-triage.md)
  - [TODOS.md 格式参考](TODOS-format.md)

### Step 2.5: Greptile 评论分流

- 读取 Greptile line-level 和 top-level 评论。
- 把评论分成 `VALID & ACTIONABLE`、`VALID BUT ALREADY FIXED`、`FALSE POSITIVE`、`SUPPRESSED`。
- 回复模板、分级和历史文件写入规则，使用 [greptile-triage](greptile-triage.md)。

### Step 3: 设计 review

- 如果 diff 触及前端文件，再按 [design-checklist](design-checklist.md) 做轻量设计 review。
- 没有前端文件时，整段设计 review 可以静默跳过。

### Step 4: 测试覆盖与风险补图

- 识别现有测试计划来源，优先复用已有测试计划。
- 如果存在明显覆盖缺口，要把缺口和建议动作说清楚，但不要把“还没补完测试”伪装成“已经安全”。

### Step 5：先修后评审

- 每个发现都要有动作，不只是 critical 问题。
- AUTO-FIX 的问题直接修，并用一行说明 `[AUTO-FIXED] [file:line] Problem → what you did`。
- 需要人做取舍的问题批量 AskUserQuestion。
- 所有声称“已安全”“别处已处理”“已有测试覆盖”的地方，都必须给出代码或测试证据。

### Step 5.5: TODOS 与文档陈旧性

- 如果仓库里有 `TODOS.md`，要交叉核对当前 diff 是否修掉了旧 TODO，或引入了应该新增的 TODO。
- 再核对 README、ARCHITECTURE、CLAUDE.md 等根文档是否因本次代码改动而变陈旧。

### Step 5.7：对抗式评审

- 根据 diff 大小自动决定是否追加 Codex 或 Claude adversarial review。
- 小 diff 可以跳过，对中大 diff 要明确写出用了哪几路模型、哪些发现是交叉确认的高置信度问题。

### Step 5.8：持久化评审结果

- review 完成后，把最终结论写入 gstack review log，供 `/ship` 等后续工作流读取。

### Step 6：记录 learnings

- 如果这次 review 发现了真正非显然、能在未来节省时间的模式、陷阱或架构洞察，再写入 learnings。

## 输出要求

- 标题格式使用：`Pre-Landing Review: N issues (X critical, Y informational)`。
- 如果没有问题，直接输出：`Pre-Landing Review: No issues found.`。
- 只讲真实问题，不做无信息量的整体表扬。

## 重要边界

- AUTO-FIX 优先处理机械问题，ASK 只留给安全、竞态、用户可见行为和其他高分歧决策。
- 不要因为 Greptile 评论存在就默认它是对的，必须自己验。
- 不要把 capability verified 误写成“真实 review 引擎链路已验收”。
- 当前 copied skill 的 promotion 只说明项目内文档资产、引用链、中文化与静态能力结构满足治理要求。
