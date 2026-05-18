---
name: land-and-deploy
description: 合并 PR、等待 CI 与 deploy 完成，并通过 canary 与健康检查验证生产状态。
allowed-tools:
  - Bash
  - Read
  - Write
  - Glob
  - AskUserQuestion
---

# `land-and-deploy` 合并与发布工作流

`land-and-deploy` 用来接管 `/ship` 之后的步骤：合并 PR、等待 CI、等待 deploy，并通过 canary 与健康检查验证生产环境状态。

这个项目内 copied skill 继续保留 gstack-heavy preamble、deploy / canary / `CLAUDE.md` 依赖说明与发布链语义。当前完成的是 copied-skill 的静态治理、中文化和 runtime-visible promotion，不代表真实 merge、CI、deploy 或 canary 链路已经做了执行级验收。

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
  echo '{"skill":"land-and-deploy","ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","repo":"'$(basename "$(git rev-parse --show-toplevel 2>/dev/null)" 2>/dev/null || echo "unknown")'"}'  >> ~/.gstack/analytics/skill-usage.jsonl 2>/dev/null || true
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
- 本阶段治理目标是 clean UTF-8 的 copied truth，不是替换原始 merge / deploy / canary 生态。

## 语气与风格

- 生产验证必须讲结果，不讲空话。
- 出现阻塞时要明确是卡在 merge、CI、deploy 还是 canary。
- 始终把真实线上风险放在第一位。

## AskUserQuestion 约束

- 先重申项目、当前分支和当前发布任务。
- 用简单语言解释当前阻塞或取舍，不把回答写成平台内部术语堆叠。
- 给出明确推荐，并标注 human / CC 成本。
- 只在 merge、deploy、生产验证的关键分叉点上提问。

## `land-and-deploy` 工作流

`land-and-deploy` 接手 `/ship` 创建好的 PR，继续完成 merge、等待 CI、等待 deploy，并验证生产健康。

### Step 1：读取 deploy 配置与当前 PR 状态

- 优先从 `CLAUDE.md` 读取 deploy 配置、生产 URL、health check、status command 和 merge method。
- 检查当前 PR 是否存在、是否可合并、是否还缺 review 或前置状态。

### Step 2：执行合并

- 按配置的 merge method 合并 PR。
- 如果 merge 失败、分支保护阻塞或缺失前置条件，停止并按完成状态协议上报。

### Step 3：等待 CI 与 deploy

- 等待 CI 完成。
- 等待 deploy 触发并完成。
- 如果项目定义了 status command，就优先读取它；否则退回到健康检查 URL 或生产环境可用性检查。

### Step 4：验证生产健康

- 按 `CLAUDE.md` 中记录的 health check 进行验证。
- 如工作流要求 canary，就继续执行 canary / smoke 检查。
- 只要生产环境仍不健康，就不能把结果标成 `DONE`。

### Step 5：总结

- 总结合并结果、CI 状态、deploy 状态、health check 结果、canary 结果和仍存 concern。
- 明确告诉用户：这次完成的是发布链动作与健康验证结果，不是单纯“已经点了 merge 按钮”。

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

- 不要把 `land-and-deploy` 改写成只做 merge 的简化版本。
- 如果 deploy 或 canary 没通过，就不能把结果粉饰成“发布成功”。
- 不要回显完整凭据、token 或密钥。
- capability verified 只说明 copied-skill 静态治理通过，不代表真实 merge、CI、deploy 或 canary 链路已经验收。
