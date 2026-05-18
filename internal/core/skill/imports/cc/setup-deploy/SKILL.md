---
name: setup-deploy
preamble-tier: 2
version: 1.0.0
description: 为 /land-and-deploy 配置部署信息，把平台、健康检查和 deploy 状态写入 CLAUDE.md。
allowed-tools:
  - Bash
  - Read
  - Write
  - Edit
  - Glob
  - Grep
  - AskUserQuestion
---

# `setup-deploy`

为 `/land-and-deploy` 配置部署信息，包括 deploy 平台、生产 URL、健康检查地址、状态命令和 deploy workflow，并把结果写入 `CLAUDE.md`，让后续发布流程自动读取。

这个复制版继续保留 gstack-heavy preamble、平台检测、`CLAUDE.md` 读写、`Write` / `Edit` 工作流，以及 `~/.claude/skills/gstack/bin/*`、`~/.gstack/*`、`CLAUDE.md` 等依赖说明。这里完成的是 copied-skill 治理与中文化，不代表真实 deploy 平台、健康检查、CLI 或发布链已经做了执行级验收。

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
  echo '{"skill":"setup-deploy","ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","repo":"'$(basename "$(git rev-parse --show-toplevel 2>/dev/null)" 2>/dev/null || echo "unknown")'"}'  >> ~/.gstack/analytics/skill-usage.jsonl 2>/dev/null || true
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

- `PROACTIVE`、`SKILL_PREFIX`、`UPGRADE_AVAILABLE`、telemetry、routing rules 和 contributor mode 等逻辑继续沿用原 gstack preamble。
- 这部分说明依然依赖 `~/.claude/skills/gstack/bin/*`、`~/.gstack/*` 与 `CLAUDE.md`，不在本阶段重写成新的 standalone 机制。
- 本阶段治理目标是 clean UTF-8 的 copied truth，不是替换 gstack 的 deploy 生态或发布策略。

## `setup-deploy` 工作流

检测 deploy 平台、生产 URL、健康检查和状态命令，再把结果持久化到 `CLAUDE.md`。

`/land-and-deploy` 之后会直接读取这里写入的配置，从而跳过重复检测。

## Step 1：检查现有配置

先看 `CLAUDE.md` 里是否已经存在 deploy 配置：

```bash
grep -A 20 "## Deploy Configuration" CLAUDE.md 2>/dev/null || echo "NO_CONFIG"
```

如果已经存在配置，先展示给用户，然后用 `AskUserQuestion` 询问：

- 从头重新配置并覆盖
- 只编辑某些字段
- 保留现状，不做变更

## Step 2：检测 deploy 平台

继续沿用原 skill 的平台检测顺序和语义：

- `fly.toml`，视为 Fly.io
- `render.yaml`，视为 Render
- `vercel.json` 或 `.vercel`
- `netlify.toml`
- 仅有 GitHub Actions workflow
- 如果都没发现，转入 custom / manual 模式

平台检测时，保留原流程中的 CLI 检查、生产 URL 推断、deploy status command 推断和 health check 推断逻辑。

## Step 3：让用户确认关键信息

如果能自动检测到平台，就把检测结果展示给用户并要求确认，尤其是：

- 生产 URL
- health check URL 或命令
- deploy 状态命令
- merge 方式
- 项目类型

如果是 custom / manual 模式，使用 `AskUserQuestion` 收集：

1. deploy 是如何触发的
2. 生产 URL 是什么
3. gstack 应如何检查 deploy 是否成功
4. 合并前或合并后有没有额外 hooks

## Step 4：写入配置

读取 `CLAUDE.md`，找到并替换 `## Deploy Configuration` 段；如果不存在，就追加到末尾。

保留原 skill 的目标配置形态：

```markdown
## Deploy Configuration (configured by /setup-deploy)
- Platform: {platform}
- Production URL: {url}
- Deploy workflow: {workflow file or "auto-deploy on push"}
- Deploy status command: {command or "HTTP health check"}
- Merge method: {squash/merge/rebase}
- Project type: {web app / API / CLI / library}
- Post-deploy health check: {health check URL or command}

### Custom deploy hooks
- Pre-merge: {command or "none"}
- Deploy trigger: {command or "automatic on push to main"}
- Deploy status: {command or "poll production URL"}
- Health check: {URL or command}
```

`CLAUDE.md` 继续是这里的唯一真值来源，不新增平行配置文件。

## Step 5：验证

写入完成后，继续沿用原流程做最小验证：

1. 如果配置了 health check URL，就尝试请求它
2. 如果配置了 deploy status command，就尝试执行一次并读取前几行输出

如果验证失败，要明确告诉用户，但不要因此阻断配置落地。

## Step 6：总结

最后向用户总结：

- 当前平台
- 生产 URL
- health check
- status command
- merge method

并明确说明：配置已经保存到 `CLAUDE.md`，后续 `/land-and-deploy` 会自动使用。

## 重要规则

- 绝不回显完整的密钥、token 或密码
- 写入前必须把检测结果给用户确认
- `CLAUDE.md` 继续是 deploy 配置唯一真值
- 再次运行 `/setup-deploy` 时，应能稳定覆盖旧配置
- 平台 CLI 是可选项，缺少 CLI 时继续退回 URL 级健康检查
