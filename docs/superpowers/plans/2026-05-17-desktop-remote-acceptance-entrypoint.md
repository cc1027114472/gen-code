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
- 后续 CI 预检查的优先候选入口

当前覆盖：

- desktop 首屏关键中文文案
- `remote-app-server / canonical` 运行链路 token
- `SSE 实时刷新` 展示
- canonical remote `5174 + 10008` 基本连通性

### 2. 完整 Full 验收

在仓库根目录执行：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

默认环境变量：

- `GEN_CODE_UI_BASE_URL=http://127.0.0.1:5174/`
- `GEN_CODE_API_BASE_URL=http://127.0.0.1:10008`

## 前置条件

运行这些命令前应满足：

1. `desktop/frontend` 对应本地 UI 已启动并可访问 `5174`
2. canonical runtime / app-server 已启动并可访问 `10008`
3. 当前 Python 环境已安装 Playwright 的 Python 包
4. 当前浏览器验收目标为 canonical remote lane，而不是 `desktop local-fallback`

## 当前覆盖范围

脚本入口：

- [run-desktop-smoke-check.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-smoke-check.ps1)
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
  - approval / write execution / rollback 可见性

## 非覆盖范围

当前默认入口不覆盖：

- `desktop local-fallback` 的同级 browser 自动化
- in-app browser plugin 链路
- 完整 CI 托管执行

其中 fallback lane 继续以手工验收和 Go 测试证据为主，不伪装成同级 browser 自动化通过项。

## 相关文档

- [Desktop 文案编码与运行态一致性验收报告](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-desktop-copy-encoding-acceptance-report.md)
- [Desktop Copy Encoding And Runtime Alignment Implementation Plan](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-desktop-copy-encoding-and-runtime-alignment-plan.md)

## 下一步建议

下一步若要继续推进，优先二选一：

1. 先把 smoke 入口挂进 CI 预检查，再视运行时长决定是否增加 full lane
2. 补 fallback lane 的更强证据链，但继续保持它不是同级 browser 自动化
