# 参考：Manus 的上下文工程原则

这个 skill 建立在 Manus 的上下文工程原则之上。Manus 是 2025 年 12 月被 Meta 以 20 亿美元收购的 AI agent 公司。

## Manus 的 6 条原则

### 原则 1：围绕 KV-Cache 设计

> “KV-cache hit rate is THE single most important metric for production AI agents.”

**数据：**
- 输入与输出 token 比大约是 100:1
- 已缓存 tokens：$0.30/MTok；未缓存：$3/MTok
- 成本差距达到 10 倍

**落地做法：**
- 保持 prompt 前缀稳定（单个 token 改动也会让缓存失效）
- 不要在 system prompt 里放时间戳
- 让上下文采用追加式、可确定序列化的结构

### 原则 2：不要移除，要做遮罩

不要动态删除工具（这会破坏 KV-cache）。应使用 logit masking。

**最佳实践：** 为动作使用统一前缀（例如 `browser_`、`shell_`、`file_`），方便遮罩控制。

### 原则 3：把文件系统当作外部记忆

> “Markdown is my 'working memory' on disk.”

**公式：**
```text
Context Window = RAM（易失、有限）
Filesystem = Disk（持久、近乎无限）
```

**压缩必须可恢复：**
- 即使网页内容被丢弃，也要保留 URL
- 即使文档正文被丢弃，也要保留文件路径
- 绝不能丢失回到完整数据的指针

### 原则 4：通过复述操纵注意力

> “Creates and updates todo.md throughout tasks to push global plan into model's recent attention span.”

**问题：** 在大约 50 次工具调用后，模型会忘记最初目标（典型的 “lost in the middle” 效应）。

**解决办法：** 在每次决策前重读 `task_plan.md`。这样目标会重新进入注意力窗口。

```text
上下文开头：[最初目标——太远了，容易被忘记]
...很多工具调用...
上下文结尾：[刚刚读取的 task_plan.md——重新获得注意力]
```

### 原则 5：把错误内容留在上下文里

> “Leave the wrong turns in the context.”

**原因：**
- 失败动作及其堆栈信息能让模型隐式更新判断
- 可以降低重复犯错概率
- 错误恢复是 “真正 agentic 行为最清晰的信号之一”

### 原则 6：不要被 few-shot 模式绑死

> “Uniformity breeds fragility.”

**问题：** 重复的 action-observation 模式会导致漂移和幻觉。

**解决办法：** 引入受控变化：
- 适度变化措辞
- 不要盲目复制粘贴模式
- 在重复任务中主动重新校准

---

## 3 种上下文工程策略

基于 Lance Martin 对 Manus 架构的分析。

### 策略 1：上下文收缩

**压缩：**
```text
工具调用有两种表示：
├── FULL：原始工具内容（保存在文件系统）
└── COMPACT：只保留引用或文件路径

规则：
- 对旧的、已经不新鲜的工具结果做压缩
- 对最近的结果保留完整内容，以指导下一步决策
```

**摘要化：**
- 当压缩的收益开始递减时才使用
- 基于完整工具结果生成
- 产出标准化摘要对象

### 策略 2：上下文隔离（多代理）

**架构：**
```text
┌─────────────────────────────────┐
│         PLANNER AGENT           │
│  └─ 把任务分配给子代理          │
├─────────────────────────────────┤
│       KNOWLEDGE MANAGER         │
│  └─ 审查对话                    │
│  └─ 决定写入文件系统的内容      │
├─────────────────────────────────┤
│      EXECUTOR SUB-AGENTS        │
│  └─ 执行被分配的任务            │
│  └─ 拥有各自独立的上下文窗口    │
└─────────────────────────────────┘
```

**关键洞察：** Manus 最初用 `todo.md` 做任务规划，但发现大约 33% 的动作都花在更新它上面，于是转向由专门的 planner agent 调用 executor sub-agents。

### 策略 3：上下文卸载

**工具设计：**
- 总原子函数数量控制在 20 个以内
- 把完整结果存到文件系统，而不是一直塞在上下文里
- 用 `glob` 和 `grep` 做搜索
- 渐进式披露：只在需要时加载信息

---

## Agent 循环

Manus 按以下 7 步循环工作：

```text
┌─────────────────────────────────────────┐
│  1. ANALYZE CONTEXT                      │
│     - 理解用户意图                       │
│     - 评估当前状态                       │
│     - 审查最近观察结果                   │
├─────────────────────────────────────────┤
│  2. THINK                                │
│     - 是否需要更新计划？                 │
│     - 下一步最合适的动作是什么？         │
│     - 是否存在阻塞？                     │
├─────────────────────────────────────────┤
│  3. SELECT TOOL                          │
│     - 选择一个工具                       │
│     - 确认参数已齐备                     │
├─────────────────────────────────────────┤
│  4. EXECUTE ACTION                       │
│     - 工具在沙箱中运行                   │
├─────────────────────────────────────────┤
│  5. RECEIVE OBSERVATION                  │
│     - 结果追加到上下文                   │
├─────────────────────────────────────────┤
│  6. ITERATE                              │
│     - 返回步骤 1                         │
│     - 持续直到完成                       │
├─────────────────────────────────────────┤
│  7. DELIVER OUTCOME                      │
│     - 把结果发送给用户                   │
│     - 附带所有相关文件                   │
└─────────────────────────────────────────┘
```

---

## Manus 会创建的文件类型

| 文件 | 用途 | 创建时机 | 更新时机 |
|------|------|----------|----------|
| `task_plan.md` | 阶段追踪、进度 | 任务开始时 | 每完成一个阶段后 |
| `findings.md` | 发现、决策 | 每次有新发现后 | 查看图片/PDF 后 |
| `progress.md` | 会话日志、已完成内容 | 在关键断点创建 | 持续更新 |
| 代码文件 | 实现产物 | 执行前创建 | 出错后修改 |

---

## 关键约束

- **单动作执行：** 每个回合只做一次工具调用，不做并行执行
- **计划是必须的：** agent 必须始终知道目标、当前阶段和剩余阶段
- **文件即记忆：** 上下文是易失的，文件系统是持久的
- **不要重复失败：** 如果某个动作失败，下一步必须不同
- **沟通也是工具：** 消息类型包括 `info`（进度）、`ask`（阻塞）、`result`（最终结果）

---

## Manus 数据

| 指标 | 数值 |
|------|------|
| 每个任务平均工具调用数 | 约 50 |
| 输入/输出 token 比 | 100:1 |
| 收购价格 | 20 亿美元 |
| 到达 1 亿美元收入所用时间 | 8 个月 |
| 上线后框架重构次数 | 5 次 |

---

## 关键引语

> “Context window = RAM (volatile, limited). Filesystem = Disk (persistent, unlimited). Anything important gets written to disk.”

> “if action_failed: next_action != same_action. Track what you tried. Mutate the approach.”

> “Error recovery is one of the clearest signals of TRUE agentic behavior.”

> “KV-cache hit rate is the single most important metric for a production-stage AI agent.”

> “Leave the wrong turns in the context.”

---

## 来源

基于 Manus 官方关于上下文工程的文档：
https://manus.im/blog/Context-Engineering-for-AI-Agents-Lessons-from-Building-Manus
