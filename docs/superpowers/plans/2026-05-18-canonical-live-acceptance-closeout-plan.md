# Canonical Live Acceptance 收口记录

## 摘要

本阶段主线是把 canonical remote desktop/browser 验收链收成稳定的发布入口，而不是扩功能。当前已经完成的部分包括：

- `scripts/run-desktop-live-refresh-check.ps1` 现在默认强制自举当前 repo 的 server/frontend，避免 full lane 误复用“端口健康但代码版本不一致”的旧实例；若显式设置 `GEN_CODE_FORCE_CURRENT_BOOTSTRAP=0`，才回退到“健康实例可复用”的兼容模式。
- `scripts/verify-desktop-live-refresh.py` 已补 canonical MCP verified lane preflight，并把结果写入 `mcpVerifiedLanePreflight`。
- canonical full lane 已在本地通过，覆盖 direct tools、MCP verified lanes、`agent.run`、approval、write execution、rollback，以及 agent failure matrix。
- `Runtime Entry Release Checklist` 已补齐“当前代码优先”的 wrapper 行为与证据记录要求。

当前 P0 已完成的关键补充是：**真实 GitHub 远端 smoke 首跑证据已经落地**。本轮先拿到了第一次真实远端失败证据，再完成了针对 CI-only 编码问题的最小修复，并在第二次远端 run 上确认 smoke gate 成功。

## 已完成收口

### 1. Full wrapper 命中当前代码实例

- full wrapper 不再盲目信任常驻 `10008` / `5174`。
- preflight 固定检查：
  - `/api/runtime/status`
  - `/api/mcp/servers`
- MCP baseline 固定要求：
  - `external-fixture`
  - `sdk-external-fixture`
  - `third-party-time`
- 默认模式下，wrapper 会优先停止当前端口上属于本项目的旧 server / frontend 进程，再启动本 repo 的：
  - `go run ./cmd/server`
  - `npm run dev -- --host 127.0.0.1 --port 5174`
- 对 `10008` 的“是否属于本项目”判定补了 runtime status `projectRoot` 兜底，避免 `go run` 产物落到 `%TEMP%` 后命令行缺少源码路径时误判。
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

### 4. 本轮额外修复与根因澄清

本轮 full lane 回归里，一开始出现过 `agent action "result" is not supported`。继续追查后确认，这不是当前源码缺少 `result -> respond` alias，而是验收入口命中了混合环境：

- `10008` 指向的是本仓库 runtime
- `5174` 却指向另一个项目的 Vite UI

也就是说，full lane 当时在用“当前 repo API + 外部项目 UI”的组合跑验收，所以表象像是 agent 动作兼容问题，根因其实是验收入口没有强制命中当前 repo 实例。

当前代码里的真实状态：

- `agent_loop.go` 已经把 `result` / `response` 归一成 `respond`
- full wrapper 已改成默认强制 current repo bootstrap，并对 `10008` / `5174` 的进程归属做更严格判定

修复后结果：

- `powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1` 已再次本地通过
- 这次通过是在 full wrapper 强制命中当前 repo server/frontend 的前提下得到的，可作为发布前 canonical full lane 的可信入口

## 真实远端首跑记录

### 第一次真实远端运行

- 日期：`2026-05-18`
- branch / ref：`master`
- workflow：`desktop-smoke.yml`
- run id：`26011388873`
- URL：<https://github.com/cc1027114472/gen-code/actions/runs/26011388873>
- 结论：`failed`
- artifact：`desktop-smoke-summary` 已成功下载并留证到 `tmp/github-smoke-run-26011388873`

首个有效失败证据：

- 失败位置：`Run desktop smoke acceptance`
- 失败分类：`CI-only Windows encoding issue`
- 首个定位证据：`UnicodeEncodeError: 'charmap' codec can't encode characters...`
- 直接触发点：`scripts/verify-desktop-live-refresh.py` 中的 `print(json.dumps(result, ensure_ascii=False))`

结论：

- 这不是产品逻辑或 runtime lane 问题，而是 GitHub Actions Windows PowerShell/Python 控制台默认编码导致的输出异常。
- 针对该问题，已在两个 acceptance wrapper 中固定补上 UTF-8 控制台与 `PYTHONIOENCODING=utf-8` 环境约束：
  - `scripts/run-desktop-smoke-with-bootstrap.ps1`
  - `scripts/run-desktop-live-refresh-check.ps1`

### 第二次真实远端运行（修复后复跑）

- 日期：`2026-05-18`
- branch / ref：`master`
- workflow：`desktop-smoke.yml`
- run id：`26011644351`
- URL：<https://github.com/cc1027114472/gen-code/actions/runs/26011644351>
- 结论：`success`
- artifact：`desktop-smoke-summary` 已成功下载并留证到 `tmp/github-smoke-run-26011644351`

本次成功 run 的关键信号：

- `runtimeSource=remote-app-server`
- `runtimeTrust=canonical`
- `canonicalRuntimeUrl=http://127.0.0.1:10008`
- `refreshMode.label=SSE 已连接`
- `copyAndRuntimeConsistency.runtimeLaneLabel=remote-app-server`
- `copyAndRuntimeConsistency.runtimeTrustLabel=canonical`

因此当前可以确认：

- `.github/workflows/desktop-smoke.yml` 已完成一次真实 GitHub 远端通过
- 默认 canonical smoke gate 仍固定为 `UI 5174 + API 10008`
- 首次失败证据与修复后成功证据都已留档，可用于后续发布与排障复盘

## 当前边界

- `smoke` 继续是默认 CI gate
- `full` 继续是手动 / 发布前检查，不提升为默认 GitHub workflow
- `remote-app-server` 继续是唯一 canonical browser gate
- `desktop local-fallback` 继续只作为 evidence-only / manual refresh lane
- 当前 MCP baseline 继续明确为“三条已验证 lane”，不是任意第三方 MCP server 全兼容
