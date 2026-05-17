# Desktop Live Refresh Check

## Purpose

为 desktop 首页的 thread 审批与回退链路提供一条稳定的本地验收路径。

当前这条验收重点覆盖：

- 创建 `workspace.apply_patch` 写任务
- 页面无需手动刷新即可出现待审批卡片
- approve 后无需手动刷新即可出现写执行记录
- 点击 `回退最近一次`
- rollback 待审批出现后再次 approve
- 页面无需手动刷新即可看到 rollback 完成

## Why

当前 `agent-browser` 在这台 Windows 机器上仍有外部环境不稳定问题：

- CLI 可以拉起 Chrome
- 但 `open about:blank` 和本地页面导航仍会挂起

因此当前稳定验收路径切换为 Python Playwright。

## Scripts

- Python 验收脚本：
  [verify-desktop-live-refresh.py](/D:/GOWorks/gen-code-heji/gen-code/scripts/verify-desktop-live-refresh.py)
- PowerShell 包装脚本：
  [run-desktop-live-refresh-check.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-live-refresh-check.ps1)

## Usage

先确保：

- app-server 运行在 `http://127.0.0.1:10008`
- desktop frontend 运行在 `http://127.0.0.1:5174/`
- Python Playwright 已可用

运行：

```powershell
.\scripts\run-desktop-live-refresh-check.ps1
```

默认行为：

- 脚本会自动创建一个新的 `ask-user` thread
- 在该隔离 thread 内完成 `apply -> approve -> write execution -> rollback -> approve rollback`
- 所有断言只绑定本次 run 产生的 `thread/task/write execution`
- 避免历史写执行和旧 rollback 任务污染验收结果

可选环境变量：

```powershell
$env:GEN_CODE_UI_BASE_URL = "http://127.0.0.1:5174/"
$env:GEN_CODE_API_BASE_URL = "http://127.0.0.1:10008"
.\scripts\run-desktop-live-refresh-check.ps1
```

当前脚本已经固定为“新建隔离 thread 验收”模式，不再依赖复用历史 thread。

## Output

成功时输出一段 JSON，至少包含：

- `ok`
- `threadId`
- `taskId`
- `taskTitle`
- `createdStatus`
- `approvedStatus`
- `writeExecutionId`
- `rollbackTaskId`
- `rollbackTaskTitle`
- `rollbackStatus`
- `rollbackWriteExecutionId`

## Latest Verification

2026-05-17 本地串行连续验证 3 次，均通过。

最新一次样本：

- `threadId = thread-82`
- `taskId = task-118`
- `writeExecutionId = writeexec-22`
- `rollbackTaskId = task-119`
- `rollbackWriteExecutionId = writeexec-23`

结论：

- `workspace.apply_patch` 待审批可实时出现
- approve 后写执行卡可实时出现
- `回退最近一次` 可创建 rollback 任务
- rollback approve 后可实时完成并写回 UI

## Scope

这条脚本当前覆盖“待审批 -> approve -> 写执行出现 -> rollback -> approve rollback -> rollback 完成”的实时刷新链路。

它不替代：

- Go 单元测试
- `npm run build`
- 更细粒度的异常路径验收，例如 reject rollback、漂移冲突、非最近记录回退失败
