# Canonical Live Acceptance 收口记录

## 摘要

本阶段主线是把 canonical remote desktop/browser 验收链收成稳定的发布入口，而不是扩功能。当前已经完成的部分包括：

- `scripts/run-desktop-live-refresh-check.ps1` 现在会优先复用健康 canonical 实例；若 `10008` 不可用或 MCP baseline 不完整，则自动自举当前 repo 的 server/frontend 后再跑 full verifier。
- `scripts/verify-desktop-live-refresh.py` 已补 canonical MCP verified lane preflight，并把结果写入 `mcpVerifiedLanePreflight`。
- canonical full lane 已在本地通过，覆盖 direct tools、MCP verified lanes、`agent.run`、approval、write execution、rollback，以及 agent failure matrix。
- `Runtime Entry Release Checklist` 已补齐“当前代码优先”的 wrapper 行为与证据记录要求。

当前仍未完成的唯一 P0 项是：**真实 GitHub 远端 smoke 首跑证据**。原因不是代码阻塞，而是当前环境没有 GitHub 登录，也没有 GitHub 远端。

## 已完成收口

### 1. Full wrapper 命中当前代码实例

- full wrapper 不再盲目信任常驻 `10008`。
- preflight 固定检查：
  - `/api/runtime/status`
  - `/api/mcp/servers`
- MCP baseline 固定要求：
  - `external-fixture`
  - `sdk-external-fixture`
  - `third-party-time`
- 当现有实例不满足条件时，wrapper 会自动启动：
  - `go run ./cmd/server`
  - `npm run dev -- --host 127.0.0.1 --port 5174`
- 本轮自举的前后端进程在 wrapper 退出时会自动清理。

### 2. Full verifier 环境前置断言

- full lane 在进入 MCP execution scenarios 前固定做 baseline preflight。
- 当前 summary 会记录：
  - `acceptanceMode`
  - `runtimeSource`
  - `runtimeTrust`
  - `refreshMode`
  - `fallbackEvidenceMode`
  - `uiFirstCanonicalAgentScenario`
  - `agentFailureMatrix`
  - `mcpVerifiedLanePreflight`
- 当 baseline 不满足时，应该优先判定为 canonical 实例或环境不完整，而不是把 MCP 阶段 failure 混成笼统 timeout。

### 3. 本地验收结果

本轮已实际通过：

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/mcp ./internal/core/runtime ./internal/core/runner ./cmd/cli
```

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
$env:GOTOOLCHAIN='auto'
go test ./...
```

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build
```

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-smoke-with-bootstrap.ps1
```

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

当前 full canonical lane 的关键信号已经成立：

- `runtimeSource=remote-app-server`
- `runtimeTrust=canonical`
- `refreshMode.label=SSE 已连接`
- `mcpVerifiedLanePreflight` 能看到三条 required MCP lanes
- `agentFailureMatrix` 包含：
  - `successResumeBaseline`
  - `approvalRejected`
  - `childTaskFailed`

### 4. 本轮额外修复

本轮 full lane 回归中暴露了一个 agent 动作别名问题：模型偶发返回 `type: "result"`，导致 `approvalRejected` 场景提前失败，脚本等不到 `waiting_for_approval`。

已收口为：

- `agent_loop.go` 将 `result` 视作 `respond` 的别名
- 对应单测已补，避免同类 alias 再打断 canonical full lane

## 远端首跑当前阻塞

当前机器上的事实：

- `gh` 已安装
- `gh auth status` 显示 **未登录**
- 当前仓库 `git remote -v` 只有：
  - `gitee https://gitee.com/cc1027114472/gen-code.git`
- 当前仓库没有 GitHub 远端，因此不能伪装成“已完成真实 GitHub smoke 首跑”

因此本阶段对远端首跑的真实结论是：

- runbook 已具备
- workflow 已具备
- 本地对等入口已通过
- 但 **GitHub 登录和 GitHub 远端绑定尚未满足**，所以还不能在本机完成真实首跑证据记录

## 首跑时需要补录的固定信息

待环境满足后，首次真实远端 smoke 需要在验收报告中补以下记录：

- run 日期
- branch / ref
- workflow：`desktop-smoke.yml`
- run id
- 结论：success / failed
- `desktop-smoke-summary` artifact 是否可下载
- 若失败：
  - `desktop-smoke-failure.json` 的 `category`
  - 首个有效定位证据
  - 是否属于 CI-only 问题

## 当前边界

- `smoke` 继续是默认 CI gate
- `full` 继续是手动 / 发布前检查，不提升为默认 GitHub workflow
- `remote-app-server` 继续是唯一 canonical browser gate
- `desktop local-fallback` 继续只作为 evidence-only / manual refresh lane
- 当前 MCP baseline 继续明确为“三条已验证 lane”，不是任意第三方 MCP server 全兼容
