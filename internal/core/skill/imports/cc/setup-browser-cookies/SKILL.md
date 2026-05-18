---
name: setup-browser-cookies
preamble-tier: 1
version: 1.0.0
description: 把真实 Chromium 浏览器中的 cookies 导入 headless browse 会话，供已登录页面测试使用。
allowed-tools:
  - Bash
  - Read
  - AskUserQuestion
---

# `setup-browser-cookies`

把你真实 Chromium 浏览器中的登录态 cookies 导入 gstack 的 headless browse 会话。

这个 skill 适用于需要测试已登录页面、复用真实浏览器会话，或在开始 QA 之前先完成认证准备的场景。

本复制版继续保留 gstack-heavy preamble、`browse` 二进制检查、cookie picker、CDP 模式分支，以及 `~/.claude/skills/gstack/bin/*` 和 `~/.gstack/*` 相关说明。这里完成的是 copied-skill 治理与中文化，不代表真实浏览器导入链路已经做了执行级验收。

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
  echo '{"skill":"setup-browser-cookies","ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","repo":"'$(basename "$(git rev-parse --show-toplevel 2>/dev/null)" 2>/dev/null || echo "unknown")'"}'  >> ~/.gstack/analytics/skill-usage.jsonl 2>/dev/null || true
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

- 如果 `PROACTIVE=false`，不要主动建议或自动触发其他 gstack skills，除非用户明确要求。
- 如果 `SKILL_PREFIX=true`，涉及其他 gstack skill 时使用 `/gstack-` 前缀，磁盘路径规则不变。
- 如果输出里出现 `UPGRADE_AVAILABLE <old> <new>`，按 `gstack-upgrade/SKILL.md` 中的内联升级流程处理。
- 如果 `LAKE_INTRO=no`、`TEL_PROMPTED=no`、`PROACTIVE_PROMPTED=no` 或路由规则尚未配置，继续沿用 preamble 中定义的用户提示和一次性初始化流程。
- 上述逻辑继续依赖 `~/.claude/skills/gstack/bin/*`、`~/.gstack/*` 与 `CLAUDE.md`，这里不把它们改写成新的独立机制。

## `setup-browser-cookies` 工作流

把真实 Chromium 浏览器中的已登录会话导入当前 browse 会话。

## CDP 模式检查

先检查 browse 是否已经连接到用户的真实浏览器：

```bash
$B status 2>/dev/null | grep -q "Mode: cdp" && echo "CDP_MODE=true" || echo "CDP_MODE=false"
```

如果 `CDP_MODE=true`，告诉用户：不需要再导入 cookies，你已经通过 CDP 连接到了真实浏览器，现有 cookies 和会话可以直接使用。然后停止。

## 工作方式

1. 找到 `browse` 二进制。
2. 运行 `cookie-import-browser`，检测已安装浏览器并打开 cookie picker。
3. 由用户在浏览器里选择要导入的域名。
4. 把 cookies 解密并加载到 Playwright 会话中。

## 操作步骤

### 1. 找到 browse 二进制

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
2. 得到确认后运行：`cd <SKILL_DIR> && ./setup`
3. 如果缺少 `bun`，继续沿用原 skill 中给出的安装脚本和 checksum 检查流程。

### 2. 打开 cookie picker

```bash
$B cookie-import-browser
```

这个命令会自动检测已安装的 Chromium 浏览器，并在默认浏览器里打开交互式 picker。用户可以：

- 切换不同浏览器
- 搜索域名
- 点击 `+` 导入某个域名的 cookies
- 点击垃圾桶按钮移除已导入 cookies

告诉用户：`Cookie picker 已打开，请在浏览器里选择要导入的域名，完成后告诉我。`

### 3. 直接导入（可选）

如果用户直接指定域名，例如 `/setup-browser-cookies github.com`，跳过 UI，直接运行：

```bash
$B cookie-import-browser comet --domain github.com
```

如果用户指定了别的浏览器，把 `comet` 替换为对应浏览器名。

### 4. 验证

用户确认完成后运行：

```bash
$B cookies
```

把导入结果摘要给用户，重点看各域名的 cookie 数量。

## 说明

- 在 macOS 上，某些浏览器第一次导入时可能会弹出 Keychain 对话框，点击 `Allow` 或 `Always Allow`
- 在 Linux 上，`v11` cookies 可能依赖 `secret-tool` 或 libsecret，`v10` cookies 使用 Chromium 的标准回退密钥
- cookie picker 复用 browse server 的同一个端口，不会额外再拉一个服务
- UI 里只显示域名和 cookie 数量，不显示 cookie 值
- browse 会话会在命令之间保持 cookies，因此导入完成后可立即使用
