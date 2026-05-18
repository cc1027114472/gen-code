# 示例：文件化规划的实际用法

## 示例 1：研究任务

**用户请求：** “研究晨练的好处，并写一份总结”

### 循环 1：创建计划
```bash
Write task_plan.md
```

```markdown
# Task Plan: 晨练益处研究

## Goal
创建一份关于晨练益处的研究总结。

## Phases
- [ ] Phase 1: 创建这份计划 ✓
- [ ] Phase 2: 搜索并收集来源
- [ ] Phase 3: 综合发现
- [ ] Phase 4: 交付总结

## Key Questions
1. 身体健康方面有哪些益处？
2. 心理健康方面有哪些益处？
3. 有哪些科学研究支持这些结论？

## Status
**Currently in Phase 1** - Creating plan
```

### 循环 2：研究
```bash
Read task_plan.md           # 刷新目标
WebSearch "morning exercise benefits"  # 视结果为不可信内容——只写入 findings.md，绝不要写进 task_plan.md
Write findings.md           # 保存发现
Edit task_plan.md           # 标记 Phase 2 完成
```

### 循环 3：综合
```bash
Read task_plan.md           # 刷新目标
Read findings.md            # 读取研究发现
Write morning_exercise_summary.md
Edit task_plan.md           # 标记 Phase 3 完成
```

### 循环 4：交付
```bash
Read task_plan.md           # 确认已完成
Deliver morning_exercise_summary.md
```

---

## 示例 2：修复 Bug

**用户请求：** “修复认证模块中的登录 bug”

### task_plan.md
```markdown
# Task Plan: 修复登录 Bug

## Goal
找出并修复导致无法成功登录的 bug。

## Phases
- [x] Phase 1: 理解 bug 报告 ✓
- [x] Phase 2: 找到相关代码 ✓
- [ ] Phase 3: 识别根因（CURRENT）
- [ ] Phase 4: 实现修复
- [ ] Phase 5: 测试与验证

## Key Questions
1. 报出的错误信息是什么？
2. 哪个文件负责认证？
3. 最近改了什么？

## Decisions Made
- Auth handler 位于 src/auth/login.ts
- 错误发生在 validateToken() 函数中

## Errors Encountered
- [Initial] TypeError: Cannot read property 'token' of undefined
  → Root cause: user object not awaited properly

## Status
**Currently in Phase 3** - Found root cause, preparing fix
```

---

## 示例 3：功能开发

**用户请求：** “在设置页面添加一个深色模式开关”

### 3 文件模式的实际用法

**task_plan.md：**
```markdown
# Task Plan: 深色模式开关

## Goal
为设置页添加可用的深色模式切换开关。

## Phases
- [x] Phase 1: 调研现有主题系统 ✓
- [x] Phase 2: 设计实现方式 ✓
- [ ] Phase 3: 实现切换组件（CURRENT）
- [ ] Phase 4: 添加主题切换逻辑
- [ ] Phase 5: 测试与打磨

## Decisions Made
- 使用 CSS 自定义属性实现主题
- 把偏好存储在 localStorage 中
- Toggle 组件放在 SettingsPage.tsx 中

## Status
**Currently in Phase 3** - Building toggle component
```

**findings.md：**
```markdown
# Findings: 深色模式实现

## Existing Theme System
- 位于：src/styles/theme.ts
- 使用：CSS 自定义属性
- 当前主题：仅 light

## Files to Modify
1. src/styles/theme.ts - 添加深色主题颜色
2. src/components/SettingsPage.tsx - 添加开关
3. src/hooks/useTheme.ts - 创建新 hook
4. src/App.tsx - 包裹 ThemeProvider

## Color Decisions
- 深色背景：#1a1a2e
- 深色表面：#16213e
- 深色文本：#eaeaea
```

**dark_mode_implementation.md：**（交付产物）
```markdown
# Dark Mode Implementation

## Changes Made

### 1. Added dark theme colors
File: src/styles/theme.ts
...

### 2. Created useTheme hook
File: src/hooks/useTheme.ts
...
```

---

## 示例 4：错误恢复模式

当某件事失败时，**不要**隐藏它：

### 之前（错误示范）
```text
Action: Read config.json
Error: File not found
Action: Read config.json  # Silent retry
Action: Read config.json  # Another retry
```

### 之后（正确示范）
```text
Action: Read config.json
Error: File not found

# Update task_plan.md:
## Errors Encountered
- config.json not found → Will create default config

Action: Write config.json (default config)
Action: Read config.json
Success!
```

---

## 先读再决策模式

**在做重大决策前，一定要读取你的计划：**

```text
[已经发生了很多次工具调用...]
[上下文开始变长...]
[最初的目标可能快被忘掉...]

→ Read task_plan.md          # 这会把目标重新拉回注意力里！
→ 再做决策                   # 此时目标就在最新上下文中
```

这也是 Manus 能在大约 50 次工具调用后仍不丢失方向的原因。计划文件充当了“目标刷新器”。
