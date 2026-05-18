---
name: connect-chrome
version: 1.0.0
description: 启动由 gstack 控制且自动加载 Side Panel extension 的可视 Chrome 窗口。
allowed-tools:
  - Bash
  - Read
  - AskUserQuestion
---

# `connect-chrome`

启动一个由 gstack 控制的可视 Chrome 窗口，并自动加载 Side Panel extension。用户可以在真实可见的浏览器里实时看到 Claude 的操作和活动流。

这个复制版继续保留 gstack-heavy preamble、`browse` server 检查、extension 路径提示、端口 `34567`、sidebar chat 说明，以及 `~/.claude/skills/gstack/bin/*`、`~/.gstack/*`、`.claude/skills/gstack/browse/dist/browse` 相关依赖说明。这里完成的是 copied-skill 治理与中文化，不代表真实 Chrome、extension 和 headed browser 链路已经做了执行级验收。

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
  echo '{"skill":"connect-chrome","ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","repo":"'$(basename "$(git rev-parse --show-toplevel 2>/dev/null)" 2>/dev/null || echo "unknown")'"}'  >> ~/.gstack/analytics/skill-usage.jsonl 2>/dev/null || true
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

- 如果 `PROACTIVE=false`，只在用户明确调用时运行 gstack skills，不要自动触发。
- 如果 `SKILL_PREFIX=true`，跨 skill 建议或调用时使用 `/gstack-` 前缀，磁盘路径保持原样。
- 如果看到 `UPGRADE_AVAILABLE` 或 `JUST_UPGRADED`，继续沿用原 gstack 升级与提示流程。
- telemetry、Completeness Principle、routing rules、contributor mode 等一次性初始化逻辑，仍按 preamble 中的规则处理。
- 这些说明继续依赖 `~/.claude/skills/gstack/bin/*`、`~/.gstack/*` 和 `CLAUDE.md`，这里不改写成新的独立机制。

## `connect-chrome` 工作流

把 Claude 连接到一个可视的 Chrome 窗口，并自动加载 gstack Side Panel extension。

## Step 0：在任何 browse 命令之前先做检查

先找到 browse 二进制：

```bash
_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)
B=""
[ -n "$_ROOT" ] && [ -x "$_ROOT/.claude/skills/gstack/browse/dist/browse" ] && B="$_ROOT/.claude/skills/gstack/browse/dist/browse"
[ -z "$B" ] && B=~/.claude/skills/gstack/browse/dist/browse
if [ -x "$B" ]; then
  echo "READY: $B"
else
  echo "NEEDS_SETUP"
fi
```

如果输出 `NEEDS_SETUP`：

1. 先问用户：`gstack browse 需要一次性构建，大约 10 秒。现在要继续吗？`
2. 得到确认后再继续后续 browse 安装或 setup 流程。

然后清理可能残留的 browse server 和锁文件，避免新的 headed 会话被旧状态误伤。

## Step 1：连接 Chrome

运行 gstack 的 headed connect 流程。完成后把完整输出回显给用户，并确认输出里出现 `Mode: headed`。

如果输出报错，或者没有看到 `Mode: headed`，先运行：

```bash
$B status
```

把结果展示给用户，再决定是否重试。

## Step 2：验证状态

```bash
$B status
```

确认输出里仍然有 `Mode: headed`。然后从状态文件中读取端口：

```bash
cat "$(git rev-parse --show-toplevel 2>/dev/null)/.gstack/browse.json" 2>/dev/null | grep -o '"port":[0-9]*' | grep -o '[0-9]*'
```

默认端口应为 `34567`。如果不同，把实际端口告诉用户。

再找出 extension 路径，方便用户必要时手动加载：

```bash
_EXT_PATH=""
_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)
[ -n "$_ROOT" ] && [ -f "$_ROOT/.claude/skills/gstack/extension/manifest.json" ] && _EXT_PATH="$_ROOT/.claude/skills/gstack/extension"
[ -z "$_EXT_PATH" ] && [ -f "$HOME/.claude/skills/gstack/extension/manifest.json" ] && _EXT_PATH="$HOME/.claude/skills/gstack/extension"
echo "EXTENSION_PATH: ${_EXT_PATH:-NOT FOUND}"
```

## Step 3：引导用户打开 Side Panel

使用 `AskUserQuestion`，引导用户确认以下事项：

- 可见的是 Playwright 控制的 Chrome，而不是用户日常使用的常规 Chrome
- 工具栏里能看到或固定 `gstack browse` extension
- 点击 pinned icon 后，右侧会打开 Side Panel，并显示实时活动流
- extension 默认使用端口 `34567`

如果用户能看到 Side Panel，就继续。

如果用户能看到 Chrome 但找不到 extension，继续按原流程引导用户：

1. 打开 `chrome://extensions`
2. 查找并确认 `gstack browse` 已启用
3. 如未固定，回到普通页面后从 puzzle piece 菜单里固定
4. 如果根本没列出，就使用 `Load unpacked` 手动加载 Step 2 中找到的 `EXTENSION_PATH`

如果用户报告异常：

1. 运行 `$B status` 并展示结果
2. 如果 server 不健康，重新做清理和连接
3. 如果 server 健康但窗口不可见，尝试 `$B focus`
4. 如果仍失败，继续询问用户具体看到了什么

## Step 4：演示一次活动流

用户确认 Side Panel 正常后，做一个快速演示：

```bash
$B goto https://news.ycombinator.com
```

等待 2 秒后运行：

```bash
$B snapshot -i
```

告诉用户：Side Panel 里应该能看到 `goto` 和 `snapshot` 出现在活动流中。

## Step 5：说明 sidebar chat

告诉用户：

- Side Panel 还有一个 chat tab
- 可以在里面输入自然语言，例如“先抓一个 snapshot，再描述当前页面”
- sidebar agent 会在浏览器里执行这些动作，并把命令实时显示到活动流里
- 每个任务最多运行 5 分钟，且与当前 Claude Code 会话隔离，不会直接干扰这里的窗口

## Step 6：接下来能做什么

告诉用户：

- 运行 `/qa`、`/design-review`、`/benchmark` 等技能时，可以在可视 Chrome 里看到每一步
- 不需要再额外做 cookie import，这个 Playwright 浏览器会共享自己的会话
- 可以直接用 `$B goto`、`$B click`、`$B fill`、`$B snapshot -i` 控制浏览器
- 可以用 `$B focus` 把窗口拉到前台，用 `$B disconnect` 退回 headless 模式

如果用户没有给出下一步任务，再问他们想测试什么或想浏览什么。
