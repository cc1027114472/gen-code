# Desktop Remote 验收默认入口

## 目标

把 canonical remote `5174 + 10008` 的 desktop/browser 验收固定为仓库内可复用的默认入口，避免依赖聊天记录或临时口头约定。

## 默认入口

### 1. 快速首屏 Smoke

在仓库根目录执行：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-smoke-check.ps1
```

适用场景：

- 本地快速预检
- 改完 desktop 文案或 runtime 展示后先做一轮低成本确认
- 仓库默认的 desktop smoke gate

当前覆盖：

- desktop 首屏关键中文文案
- `remote-app-server / canonical` 运行链路 token
- `SSE 实时刷新` 展示
- canonical remote `5174 + 10008` 基本连通性

若需要自动启动本地 API 与前端，可执行：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-smoke-with-bootstrap.ps1
```

这条入口会自行启动：

- `go run .\cmd\server`
- `npm run dev -- --host 127.0.0.1 --port 5174`

然后执行 smoke 验收并在结束后清理子进程。

### 2. 完整 Full 验收

在仓库根目录执行：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

默认环境变量：

- `GEN_CODE_UI_BASE_URL=http://127.0.0.1:5174/`
- `GEN_CODE_API_BASE_URL=http://127.0.0.1:10008`

当前 full wrapper 固定遵循“当前代码优先”：

- 若现有 `10008` canonical 实例可访问，且 `/api/mcp/servers` 已暴露完整 MCP baseline，则直接复用该实例
- 若现有实例不可访问，或缺少以下任一 verified lane：
  - `external-fixture`
  - `sdk-external-fixture`
  - `third-party-time`
  wrapper 会自动自举当前 repo 的：
  - `go run .\cmd\server`
  - `npm run dev -- --host 127.0.0.1 --port 5174`
- 自举结束后再进入 full verifier，并在退出时清理本轮启动的子进程

## 前置条件

运行这些命令前应满足：

1. `desktop/frontend` 对应本地 UI 已启动并可访问 `5174`
2. canonical runtime / app-server 已启动并可访问 `10008`
3. 当前 Python 环境已安装 Playwright 的 Python 包
4. 当前浏览器验收目标为 canonical remote lane，而不是 `desktop local-fallback`

## 当前覆盖范围

脚本入口：

- [run-desktop-smoke-check.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-smoke-check.ps1)
- [run-desktop-smoke-with-bootstrap.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-smoke-with-bootstrap.ps1)
- [run-desktop-live-refresh-check.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-live-refresh-check.ps1)
- [verify-desktop-live-refresh.py](/D:/GOWorks/gen-code-heji/gen-code/scripts/verify-desktop-live-refresh.py)

当前自动化覆盖：

- `smoke`
  - desktop 首屏关键中文文案
  - `remote-app-server / canonical` 运行链路 token
  - `SSE 实时刷新` 展示
- `full`
  - 包含 smoke 全覆盖
  - direct tool task 可见性
  - `agent.run` 父子任务可见性
  - `agent.run` failure-state matrix
  - success resume baseline
  - approval rejected
  - child task failed
  - recovered_as_failed evidence
  - approval / write execution / rollback 可见性

职责边界固定为：

- `smoke`
  - 正式默认 CI gate
  - 只要求首屏文案、runtime lane、refresh mode 和 canonical remote 链路成立
- `full`
  - 手动 / 发布前检查
  - 用于完整任务流、审批、写执行、rollback、MCP、agent 和 direct tool 链路
  - `remote-app-server`: canonical browser gate
  - `desktop local-fallback`: browser-visible evidence lane only
  - failure-state coverage会同时出现在 summary 中，但默认 browser pass/fail 只由 remote canonical lane 决定
  - 当前不进入默认 GitHub workflow

## 非覆盖范围

当前默认入口不覆盖：

- `desktop local-fallback` 的同级 browser 自动化
- in-app browser plugin 链路
- full lane 的稳定 CI 托管执行

其中 fallback lane 继续以手工验收和 Go 测试证据为主，不伪装成同级 browser 自动化通过项。

当前 full summary 约定：

- `remote.agentFailureMatrix`
  - canonical browser live scenarios
  - success resume baseline
  - approval rejected
  - child task failed
  - recovered_as_failed evidence reference
- `fallback.agentFailureEvidence`
  - evidence-only
  - browserAutomation=`not attempted`
  - 当前显式记录 `recovered_as_failed` 的持久化证据测试

## CI Gate

仓库已补最小 GitHub Actions workflow：

- [.github/workflows/desktop-smoke.yml](/D:/GOWorks/gen-code-heji/gen-code/.github/workflows/desktop-smoke.yml)

当前定位：

- 只跑 smoke lane
- 通过自举脚本拉起 API 与前端
- 作为 canonical remote `5174 + 10008` 的默认 desktop smoke gate
- 失败时自动上传 smoke 日志产物，便于排查前后端启动或页面验收问题
- 成功时上传 smoke summary JSON，失败时额外上传失败 JSON 与页面截图

当前 artifact 语义固定为：

- `desktop-smoke-summary`
  - 成功时至少应包含 `desktop-smoke-summary.json`
- `desktop-smoke-logs`
  - 失败时上传前后端启动日志
- `desktop-smoke-screenshot`
  - 失败时上传页面截图

## 真实 GitHub 远端首跑状态

当前仓库已经具备：

- smoke workflow：`desktop-smoke.yml`
- 本地 smoke/full 对等入口
- summary / logs / screenshot artifact 语义

当前真实状态已更新为：

- `gh` 已安装并已完成 `gh auth login`
- 当前仓库已绑定 GitHub 远端 `https://github.com/cc1027114472/gen-code.git`
- 已完成一次真实远端 `desktop-smoke.yml` 首跑
- 首跑 run id：`26040617047`
- 首跑结论：`success`
- `desktop-smoke-summary` artifact 已成功下载并核对

因此当前可以明确说：

- 远端 smoke 首跑手册和本地对等入口已准备好
- 真实 GitHub 远端 smoke 首跑也已经完成并留有证据

失败分类当前统一围绕：

- `api-unavailable`
- `page-load-failed`
- `runtime-lane-assertion`
- `refresh-mode-assertion`
- `desktop-copy-assertion`
- `unknown`

## 相关文档

- [Desktop 文案编码与运行态一致性验收报告](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-desktop-copy-encoding-acceptance-report.md)
- [Desktop Copy Encoding And Runtime Alignment Implementation Plan](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-desktop-copy-encoding-and-runtime-alignment-plan.md)
- [Desktop Smoke Gate 首次真实运行手册](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-18-desktop-smoke-first-run-playbook.md)

## 下一步建议

下一步若要继续推进，优先顺序固定为：

1. 完成第一次真实 GitHub 远端 smoke 首跑并记录结果
2. 若远端首跑通过，再决定是否补 full lane 的手动/发布前 runbook
3. fallback lane 继续只补证据链，不升级为同级 browser 自动化
