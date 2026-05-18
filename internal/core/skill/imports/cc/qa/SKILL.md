---
name: qa
description: 系统化执行 Web 应用 QA 测试、记录问题，并在需要时按严重级别推进修复与复测。
allowed-tools:
  - Bash
  - Read
  - Write
  - Edit
  - Glob
  - Grep
  - AskUserQuestion
  - WebSearch
---

# `qa`

系统化测试一个 Web 应用，记录发现的问题，并在需要时按优先级修复、复测和回归验证。

这个项目内 copied skill 保留 gstack-heavy preamble、浏览器工作流、分层严重级别、报告模板、问题分类和修复闭环语义，但当前完成的是 copied-skill 的静态治理、中文化与 runtime-visible promotion，不代表真实浏览器执行链路已经做了执行级验收。

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
  echo '{"skill":"qa","ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","repo":"'$(basename "$(git rev-parse --show-toplevel 2>/dev/null)" 2>/dev/null || echo "unknown")'"}'  >> ~/.gstack/analytics/skill-usage.jsonl 2>/dev/null || true
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
- 如果出现 `UPGRADE_AVAILABLE` 或 `JUST_UPGRADED`，继续沿用 `gstack-upgrade/SKILL.md` 中的升级流程。
- telemetry、Completeness Principle、routing rules、contributor mode 等一次性初始化逻辑，继续按 preamble 中的原规则处理。
- 这些说明仍依赖 `~/.claude/skills/gstack/bin/*`、`~/.gstack/*` 与 `CLAUDE.md`，这里不把它们重写成新的 project-local 机制。

## 语气与风格

- 直接说重点，说明它做什么、为什么重要、对构建者会改变什么。
- 始终把用户体验、反馈回路、真实使用场景和缺陷成本说清楚。
- 代码质量和缺陷都要认真对待，不要把最后几处边角问题当成可以忽略的小事。

## AskUserQuestion 约束

- 先重新锚定当前项目、分支和正在执行的 QA 计划。
- 用足够直白的话解释问题，不堆实现术语。
- 给出明确推荐，并同时标注 human / CC 两种成本估算。
- 优先推荐完整选项，而不是明显留坑的捷径。

## 模式

### 按 diff 感知

- 当用户没有给 URL、但仓库在特性分支上时，优先从 `git diff main...HEAD` 和提交历史推导受影响页面、路由或 API。
- 如果 diff 无法明确映射到页面，也不能跳过浏览器验证，至少退回到 Quick 模式做首页和主要导航烟雾测试。

### 完整模式

- 有明确 URL 时，按系统化探索方式访问可达页面，记录 5 到 10 个有证据的问题，并输出健康分。

### 快速模式

- 只做首页和前 5 个导航目标的快速烟雾测试，确认页面能打开、没有明显 console 错误、关键入口没断。

### 回归模式

- 在 Full 模式基础上读取上一次 `baseline.json`，对比健康分变化、已修复问题和新出现问题。

## 工作流

### Phase 1：初始化

1. 找到 browse 二进制入口。
2. 创建报告输出目录和截图目录。
3. 从 [qa-report-template](templates/qa-report-template.md) 复制报告模板。
4. 启动耗时统计。

### Phase 2：认证

- 用户提供账号密码时，使用浏览器自动化完成登录。
- 用户提供 cookie 文件时，先导入 cookie，再进入目标站点。
- 如果遇到 2FA、OTP 或 CAPTCHA，需要暂停并向用户请求继续所需信息。

### Phase 3：定向

- 打开目标站点，截图、收集链接、检查 console 错误。
- 推断技术栈，并记录在报告元数据中。
- SPA 站点不要只依赖 `links`，也要用 `snapshot -i` 找到客户端导航元素。

### Phase 4：探索

- 逐页访问，持续截图、检查 console，并按 [issue-taxonomy](references/issue-taxonomy.md) 里的分类和检查表探索页面。
- 核心页面需要更深测试，次要页面可以降低深度，但不能完全跳过。
- 如有必要，补移动端 viewport 和响应式截图。

### Phase 5：记录

- 每发现一个问题就立即记录，不要等全部测完再回忆。
- 交互型问题必须保留前后截图、关键操作和 `snapshot -D` 证据。
- 静态问题至少要保留一张能说明问题的标注截图。

### Phase 6：收尾

- 根据 rubric 计算健康分。
- 写出 Top 3 Things to Fix、console 健康汇总、严重级别统计和 ship-readiness 总结。
- 写入 `baseline.json` 作为后续 regression 模式对比基线。

### Phase 7：分诊

- Quick 只修 critical/high。
- Standard 继续覆盖 medium。
- Exhaustive 把 low/cosmetic 也一起处理。
- 不能从源码修复的问题继续标记为 deferred。

### Phase 8：修复循环

- 先定位源码，再做最小修复。
- 一次修复一个问题，一次提交一个 commit。
- 修完要复测、截图、检查 console，并把结果标记为 verified、best-effort 或 reverted。
- 纯视觉问题可以跳过回归测试文件生成；需要逻辑保护的问题要补最小回归测试。

### Phase 9：最终 QA

- 对所有受影响页面重新跑一遍 QA。
- 如果最终健康分比初始基线更差，要明确提示出现了回归。

### Phase 10：报告

- 报告同时写入本地 `.gstack/qa-reports/` 和项目级 `~/.gstack/projects/{slug}/...`。
- 汇总总问题数、修复数、deferred 数、健康分变化和 PR Summary。

### Phase 11：更新 TODOS.md

- 新发现但 deferred 的 bug，如果仓库里有 `TODOS.md`，就按约定格式补进去。
- 如果某个 TODO 已由这轮 QA 修复，也要在 `Completed` 或原条目上记录完成信息。

## 测试计划上下文

- 如果项目或历史会话里已经有更丰富的测试计划，优先复用那些测试计划。
- 只有在没有更好来源时，才退回到纯 diff 推导测试范围。

## 健康分与问题分类

- 严重级别、类别说明和每页检查表，使用 [issue-taxonomy](references/issue-taxonomy.md)。
- 报告结构、字段和 Fixes Applied / Regression Tests / Ship Readiness 格式，使用 [qa-report-template](templates/qa-report-template.md)。

## 重要规则

- 每个问题都要有可复现证据，至少一张截图。
- 每次交互后都要重新看 console。
- 绝不在报告里泄露真实密码，统一写 `[REDACTED]`。
- 不要删除输出文件，截图和报告要保留。
- 用户调用 `/qa` 或 `/qa-only` 时，不能用单元测试或源码阅读替代浏览器验证。
- capability verified 只代表这个 copied skill 的静态文档资产、引用链和治理基线成立，不代表真实 QA、浏览器会话或修复链路已经做了执行级验收。
