# 下一阶段开发计划：Canonical Live Acceptance 收口

## 摘要

目标是在**不新增业务能力、不修改 runtime contract 顶层字段、不扩张 MCP 产品面**的前提下，把当前已经通过代码与 smoke bootstrap 证明、但仍未在 `run-desktop-live-refresh-check.ps1` 这条 full canonical lane 上稳定闭环的部分收口为可重复通过的发布入口。

当前基线已经明确：

- `mcp.tool.invoke` 的三条 `stdio` 执行车道已经在代码与 Go 测试中成立：
  - `external-fixture`
  - `sdk-external-fixture`
  - `third-party-time`
- `go test ./internal/core/mcp ./internal/core/runtime ./internal/core/runner ./cmd/cli` 已通过
- `go test ./desktop/...` 已通过
- `desktop/frontend` 的 `npm run build` 已通过
- `scripts/run-desktop-smoke-with-bootstrap.ps1` 已通过，说明**当前代码启动出来的 app-server + frontend** 可以支撑 canonical smoke lane

当前唯一未收口的问题是：

- `scripts/run-desktop-live-refresh-check.ps1` 默认打到常驻 `http://127.0.0.1:10008`
- 该常驻实例可能仍是旧代码或未重启实例
- 因此 verifier 看到的 `/api/mcp/servers` 只包含 `external-fixture`，不包含 `sdk-external-fixture` 和 `third-party-time`
- 结果导致 full canonical acceptance 在新增 MCP scenarios 处超时

所以本阶段目标不是“继续实现 MCP”，而是让 **full canonical live acceptance 稳定命中当前代码与当前 verified baseline**。

## 关键改动

### 1. 统一 full canonical lane 的服务来源

围绕以下脚本与启动入口收口：

- [scripts/run-desktop-live-refresh-check.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-live-refresh-check.ps1)
- [scripts/run-desktop-smoke-with-bootstrap.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-smoke-with-bootstrap.ps1)
- [cmd/server/main.go](/D:/GOWorks/gen-code-heji/gen-code/cmd/server/main.go)
- [internal/bootstrap/app.go](/D:/GOWorks/gen-code-heji/gen-code/internal/bootstrap/app.go)

本阶段要明确并固定：

- full acceptance 不能再隐式依赖“某个未知状态的常驻 10008 进程”
- full acceptance 必须满足以下二选一之一，并固定为唯一标准：
  - `preferred`：wrapper 自举当前代码的 server / frontend，再跑 full verifier
  - `fallback`：wrapper 在执行前先探测当前 `10008` 实例是否暴露完整 baseline，若不满足则失败并给出明确环境错误

实现要求：

- 不新增新的产品 API
- 不新增新的 acceptance 框架
- 只在现有 wrapper 基础上补最小启动/探测逻辑
- full lane 与 smoke lane 的差异只应在“验收范围”，不应在“服务来源真值”上再分叉

### 2. 为 full verifier 增加环境前置断言，避免“超时型假失败”

围绕 [scripts/verify-desktop-live-refresh.py](/D:/GOWorks/gen-code-heji/gen-code/scripts/verify-desktop-live-refresh.py) 增加最小 preflight，不改主验收结构。

本阶段需要补的前置检查：

- 在进入 MCP execution scenarios 之前，显式读取 `/api/mcp/servers`
- 断言至少存在：
  - `external-fixture`
  - `sdk-external-fixture`
  - `third-party-time`
- 若缺任一项：
  - 直接输出结构化环境失败
  - category 应明确区分为环境/实例未升级，而不是 task timeout

目标是把当前这种：

- `timed out waiting for matching task`

提升为更可诊断的失败，例如：

- canonical runtime instance missing expected MCP verified lanes

这样后续排障、CI 诊断和文档口径都会更稳定。

### 3. 统一 smoke 与 full wrapper 的共享启动能力

当前已有 smoke bootstrap 脚本可证明当前代码能启动成功，因此本阶段优先复用：

- 进程启动
- 端口等待
- 输出日志路径
- 结束时清理进程

建议收口方式：

- 抽取或复用现有 smoke bootstrap 逻辑给 full wrapper
- full wrapper 在需要时也能启动：
  - `go run .\cmd\server`
  - `npm run dev -- --host 127.0.0.1 --port 5174`

本阶段不要求：

- 新建复杂脚本体系
- 加入服务管理器
- 引入长期驻留守护进程模型

只要求“当前 repo 下的 full acceptance 默认可以靠当前代码稳定跑起来”。

### 4. 验收输出与 checklist 收口

围绕以下文档和输出同步口径：

- [docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md)
- [docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md)

本阶段要补清楚：

- canonical full lane 的“通过前提”是当前代码实例，而不是任意存活中的旧实例
- MCP verified baseline 已经是：
  - fixture regression lane
  - official SDK external lane
  - third-party time lane
- 如果 full wrapper 依赖 bootstrap，自举方式要在 checklist 中写明
- 如果 full wrapper 仍允许打现成实例，则必须先做 baseline probe

不新增平行长文档，只更新现有 checklist 的最短必要表述即可。

### 5. 发布前清理与工件治理

本阶段顺手收口以下非产品问题：

- `scripts/__pycache__/`
- `tmp/desktop-smoke-artifacts/`
- bootstrap 产生的 stdout/stderr 日志文件

要求：

- 不删除用户明确要保留的日志
- 默认把这些作为临时工件处理
- 若 wrapper 需要保留失败工件，必须路径稳定、语义明确

## 公开接口 / 类型变更

本阶段默认不做以下变更：

- 不新增 runtime contract 顶层字段
- 不修改 `/api/mcp/servers` wire shape
- 不新增 MCP 专属顶层 API
- 不修改 `mcp.tool.invoke` 输入输出 shape
- 不新增 desktop 专属 payload 类型

允许的最小变更仅限：

- acceptance wrapper 脚本
- verifier 前置环境断言
- 现有 checklist / matrix 文案
- 必要的测试断言更新

## 测试与验收

必须完成：

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

通过标准：

- full wrapper 默认不再因为命中旧常驻实例而漂移
- canonical full verifier 在 MCP 阶段能稳定看到：
  - `external-fixture`
  - `sdk-external-fixture`
  - `third-party-time`
- full canonical acceptance 输出中显式保留三条 MCP execution evidence
- 若环境不满足，失败类别必须是“实例/环境未升级”，而不是笼统 timeout
- runtime checklist、matrix、wrapper 行为三者口径一致

## 建议实施顺序

1. 给 full verifier 增加 MCP lane preflight，先把失败变得可诊断
2. 让 full wrapper 复用 smoke bootstrap 的启动方式，确保默认命中当前代码实例
3. 复跑 full canonical lane，确认三条 MCP lanes 均通过
4. 同步 checklist 的运行前提和环境说明
5. 清理临时工件与日志策略

## 假设与默认值

- 默认当前代码已经具备需要的 MCP multi-server baseline，实现本身不是本阶段重点
- 默认主要问题是 canonical full acceptance 命中了旧实例或未知实例
- 默认 full lane 应优先依赖“当前代码自举实例”，而不是长期存活进程
- 默认 smoke bootstrap 已经是可信参考实现，应优先复用
- 默认完成本阶段后，下一优先级再进入：
  - broader arbitrary third-party MCP compatibility baseline
  - 更细粒度的第三方 server 兼容矩阵与配置治理
