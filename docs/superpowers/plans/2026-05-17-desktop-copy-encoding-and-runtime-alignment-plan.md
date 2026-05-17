# Desktop Copy Encoding And Runtime Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 收口 desktop 前端文案编码、运行态文案一致性和验收说明，避免把“编码误读”误判成“文件内容本身损坏”。

**Architecture:** 以 [desktop/frontend/src/App.tsx](/D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/App.tsx) 和 [desktop/frontend/src/runtimeBridge.ts](/D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/runtimeBridge.ts) 为主战场，先统一 source/trust/refresh 派生文案，再收口工作台可见 copy，最后补齐 fallback 证据和验收文档。

**Tech Stack:** Go, Wails, React, TypeScript, Vite, SQLite fallback runtime

---

## 关键实施点

- [ ] 优先判断编码来源，再决定是否需要改 repo 内文案内容。
- [ ] `runtimeBridge.ts` 作为 source/trust/refresh/manual refresh 的唯一派生口径。
- [ ] `App.tsx` 只消费桥接层派生字段，不再自解释一套 remote/fallback 文案。
- [ ] 不新增 runtime contract 字段，不新增桌面 payload 结构。
- [ ] workflow、approval、write execution 摘要必须保持可读。

## 需要完成的改动

- [ ] 修复 [runtimeBridge.ts](/D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/runtimeBridge.ts) 中 user-visible 文案：
  - `remote-app-server` / `local-fallback`
  - `canonical` / `degraded`
  - `SSE 实时刷新` / `手动刷新模式`
  - fallback note
- [ ] 修复 [App.tsx](/D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/App.tsx) 中主要用户可见文案：
  - 顶栏状态
  - 项目卡与线程导航
  - 工作流概览
  - 任务输入
  - 消息流
  - 结果抽屉
  - 浏览器预览区
- [ ] 保持 [desktop/app.go](/D:/GOWorks/gen-code-heji/gen-code/desktop/app.go) 的现有数据面，不扩散 contract。
- [ ] 在 [desktop/app_test.go](/D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go) 维持或补强最小断言：
  - fallback 下 `runtimeSource=local-fallback`
  - fallback 下 `runtimeTrust=degraded`
  - fallback 下 `RuntimeMessage` 含 `manual refresh`
  - skills / skill governance / tools 摘要非空
  - approval / write execution summary 仍可读
- [ ] 完成 [2026-05-17-desktop-copy-encoding-acceptance-report.md](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-desktop-copy-encoding-acceptance-report.md)。

## 测试与验收

- [ ] `npm run build`
  - Workdir: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend`
- [ ] `$env:GOTOOLCHAIN='auto'; go test ./...`
  - Workdir: `D:\GOWorks\gen-code-heji\gen-code\desktop`
- [ ] `$env:GOTOOLCHAIN='auto'; go test ./internal/core/runtime ./cmd/cli -run "Skill|Governance|Discovery|RuntimeStatus|SkillsList" -v`
  - Workdir: `D:\GOWorks\gen-code-heji\gen-code`

## Closeout 要求

- [ ] 明确说明本轮修的是 desktop 文案、桥接派生和展示一致性。
- [ ] 明确说明没有新增 runtime API。
- [ ] 明确说明 fallback 仍是 supporting lane，不伪装成完整 browser automation。
- [ ] 若仍有残余，限定为：
  - 更大范围的 `App.tsx` 拆分
  - 独立 desktop browser automation 资产
  - 更深的 workflow 可视化 polish
