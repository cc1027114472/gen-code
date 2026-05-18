# Desktop 文案编码与运行态一致性验收报告

## 范围

- UI 入口: `http://127.0.0.1:5174/`
- Runtime API: `http://127.0.0.1:10008`
- Fallback lane: `desktop local-fallback`
- 本阶段只收口 desktop 展示层、文案层和验收层，不扩 runtime API，不新增桌面信息架构

## 编码判断原则

本阶段固定遵循“先判编码，再判内容”的原则：

1. 先确认终端、编辑器或读取命令是否以错误编码打开文件。
2. 再确认仓库中的原始字节是否真的已经被错误写入。
3. 只有在确认文件内容本身受损时，才修改 repo 内文本。

本轮检查结论：

- [2026-05-17-desktop-copy-encoding-and-runtime-alignment-plan.md](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-desktop-copy-encoding-and-runtime-alignment-plan.md) 需要作为可读中文文档重新收口。
- [runtimeBridge.ts](/D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/runtimeBridge.ts) 与 [App.tsx](/D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/App.tsx) 的问题属于用户可见文案收口，不涉及 runtime wire shape 变更。
- [2026-05-18-desktop-smoke-first-run-playbook.md](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-18-desktop-smoke-first-run-playbook.md) 与 [2026-05-17-desktop-remote-acceptance-entrypoint.md](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-desktop-remote-acceptance-entrypoint.md) 的 Markdown 文件字节当前仍是正常 UTF-8；若 PowerShell 直接读取显示乱码，应优先按 UTF-8 方式复核，而不是直接判断文件内容受损。

## UTF-8 复核约定

若 PowerShell 默认 `Get-Content` 输出疑似乱码，本阶段统一按以下顺序处理：

1. 先用明确的 UTF-8 读取方式复核同一文件。
2. 再确认 Python / 编辑器按 UTF-8 打开时是否正常。
3. 只有在 UTF-8 原始字节也无法正确还原内容时，才判定 repo 中文本真正损坏。

这条约定同样适用于：

- smoke 首跑手册
- remote 验收默认入口
- 本验收报告自身

## 手工验收项

### Remote 路径

1. 打开 `http://127.0.0.1:5174/`。
2. 确认顶部状态、项目卡、线程区、结果抽屉、本地预览区中文文案可正常显示。
3. 确认运行链路明确显示 `remote-app-server`，并与 `canonical` 语义对齐。
4. 确认 `supportsSSE=true` 时显示 `SSE 实时刷新`，而不是手动刷新模式。
5. 确认 Provider、Skill 治理、状态存储、运行链路摘要可读且前后一致。

### Fallback 路径

1. 让 canonical runtime 不可用，触发 `desktop local-fallback`。
2. 确认页面明确显示本地降级链路，而不是把 fallback 说成 canonical remote runtime。
3. 确认 `supportsSSE=false` 时显示“手动刷新模式”。
4. 确认 skill、tool、provider、state store 摘要仍可见。
5. 确认 fallback 提示明确说明它只代表本地 SQLite snapshot，不代表 canonical shared runtime。

## 回归结果

本阶段要求执行的基础回归：

- `npm run build` in `desktop/frontend`
- `$env:GOTOOLCHAIN='auto'; go test ./...` in `desktop`
- `$env:GOTOOLCHAIN='auto'; go test ./internal/core/runtime ./cmd/cli -run "Skill|Governance|Discovery|RuntimeStatus|SkillsList" -v` in repo root

若其中任一失败，需要区分是代码问题还是环境问题。

## 自动化边界

当前事实保持明确：

- `desktop/frontend` 还没有单独内置 Playwright 资产。
- 仓库级 remote 浏览器验收资产已存在：
  - [run-desktop-smoke-check.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-smoke-check.ps1)
  - [run-desktop-smoke-with-bootstrap.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-smoke-with-bootstrap.ps1)
  - [run-desktop-live-refresh-check.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-live-refresh-check.ps1)
  - [verify-desktop-live-refresh.py](/D:/GOWorks/gen-code-heji/gen-code/scripts/verify-desktop-live-refresh.py)
- 默认 smoke gate workflow 已补：
  - [.github/workflows/desktop-smoke.yml](/D:/GOWorks/gen-code-heji/gen-code/.github/workflows/desktop-smoke.yml)
- 默认命令入口文档见：
  - [Desktop Remote 验收默认入口](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-desktop-remote-acceptance-entrypoint.md)
- 这条自动化链当前覆盖 canonical remote `5174 + 10008`，并区分：
  - smoke：首屏文案、runtime lane、刷新方式
  - full：任务流、审批、写执行与 `agent.run` failure-state matrix
- full summary 当前拆分为：
  - `remote.agentFailureMatrix`：success resume baseline、approval rejected、child task failed，以及 `recovered_as_failed` evidence reference
  - `fallback.agentFailureEvidence`：evidence-only，当前显式记录 `TestDesktopFallbackAgentRecoveredAsFailedPersistsAcrossRestart`
- smoke gate 在 CI 失败时会上传前后端日志产物，便于定位启动或验收失败原因。
- smoke gate 会稳定产出 `desktop-smoke-summary.json`；失败时还会补 `desktop-smoke-failure.json` 与失败截图。
- fallback lane 仍不作为同级 browser 自动化通过条件，继续以手工验收和 Go 测试证据为主。
- fallback browser-visible checks 仍然只算 evidence，不会抬升为 canonical browser pass/fail gate。

当前验证结论补充：

- `powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-smoke-check.ps1` 已在当前环境跑通。
- `powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-smoke-with-bootstrap.ps1` 是默认 smoke gate 的本地对等入口。
- smoke lane 适合作为本地快速预检和默认 CI smoke gate。
- full lane 仍保留为更重的完整链路验收，当前更适合作为手动触发或发布前检查。
- full lane 现在会额外记录 `agent.run` failure-state matrix：remote canonical live 覆盖 success / approval rejected / child task failed，`recovered_as_failed` 继续通过 evidence-only 方式收口。
- smoke summary 中的 `refreshMode` 已从不稳定的 `unknown` 收敛为可用的连接态信号，例如 `SSE 已连接`。
- 2026-05-18 已完成一次真实 GitHub 远端 `desktop-smoke.yml` 首跑，run id 为 `26040617047`。
- 首跑结果为 `success`，并已下载核对 `desktop-smoke-summary` artifact。
- summary 已确认：
  - `runtimeSource=remote-app-server`
  - `runtimeTrust=canonical`
  - `refreshMode.label=SSE 已连接`
- 当前 closeout 仍应明确记录：
  - 远端默认 gate 已有真实证据
  - full lane 仍是手动 / 发布前检查，不是默认 CI gate

## 通过标准

本阶段通过标准固定为：

- desktop 文案可读，不再把编码误读当成文件损坏
- remote / fallback 展示语义一致
- `supportsSSE=false` 的手动刷新模式表达明确
- workflow / approval / write execution 摘要仍可读
- 验收边界和自动化缺口记录清楚
