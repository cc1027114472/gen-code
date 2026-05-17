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
  - [run-desktop-live-refresh-check.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-live-refresh-check.ps1)
  - [verify-desktop-live-refresh.py](/D:/GOWorks/gen-code-heji/gen-code/scripts/verify-desktop-live-refresh.py)
- 这条自动化链当前覆盖 canonical remote `5174 + 10008`，并校验 desktop 文案、runtime lane、刷新方式、任务流、审批与写执行可见性。
- fallback lane 仍不作为同级 browser 自动化通过条件，继续以手工验收和 Go 测试证据为主。

## 通过标准

本阶段通过标准固定为：

- desktop 文案可读，不再把编码误读当成文件损坏
- remote / fallback 展示语义一致
- `supportsSSE=false` 的手动刷新模式表达明确
- workflow / approval / write execution 摘要仍可读
- 验收边界和自动化缺口记录清楚
