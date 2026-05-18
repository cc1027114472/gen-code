# Pre-Landing Review Checklist

## 说明

评审 `git diff origin/main` 时，只标真实问题，给出具体 `file:line` 和修法。不要为了显得勤奋而造噪音。

输出格式：

```text
Pre-Landing Review: N issues (X critical, Y informational)

AUTO-FIXED:
- [file:line] 问题 → 已应用的修法

NEEDS INPUT:
- [file:line] 问题描述
  Recommended fix: 建议修法
```

如果没有问题，直接写：

```text
Pre-Landing Review: No issues found.
```

## Pass 1，Critical

### SQL 与数据安全

- SQL 拼接是否仍然把变量直接插进语句
- 是否存在 check-then-set 这类 TOCTOU 写法
- 是否绕过了本应存在的模型校验
- 是否引入 N+1 查询

### 竞争条件与并发

- find-or-create 是否缺唯一索引保护
- 状态切换是否需要原子 `WHERE old_status = ?` 约束
- 用户可控 HTML 是否被直接 unsafe 渲染

### LLM 信任边界

- LLM 产出的 email、URL、name、结构化数据，是否在落库或发请求前做了轻量校验
- LLM 产出的 URL 是否可能形成 SSRF
- 写入知识库或向量库的内容是否存在 stored prompt injection 风险

### Shell Injection（Python）

- `shell=True` 是否和变量插值混用
- `os.system()` 是否被变量拼接污染
- `eval()` / `exec()` 是否在不安全上下文里执行了模型输出

### Enum / Value 完整性

- 新增枚举值、状态串、类型常量后，所有消费者是否都同步处理
- allowlist、filter array、case / if-elsif 链是否有漏改

## Pass 2，Informational

### 条件副作用

- 某些分支是否少做了应该一起做的状态更新、日志或 URL 绑定

### Magic Numbers 与字符串耦合

- 裸数字是否已经在多个文件里重复出现
- 错误消息字符串是否被别处当作查询条件

### Dead Code 与一致性

- 赋值后从未使用的变量
- 版本号、CHANGELOG、注释、docstring 是否与实际行为不一致

### Test Gaps

- 负路径测试是否只断言了状态，没断言副作用
- 安全控制、限流、授权、边界行为是否缺集成测试

### Completeness Gaps

- 是否存在 80% 完成度、但再花很少时间就能补到 100% 的实现

### Performance / Bundle / CI

- 是否引入明显过重的新依赖
- 是否把同步脚本、超大静态资源或分发路径不一致的问题带进来了

## Fix-First Heuristic

### AUTO-FIX

- 死代码
- 注释与代码不一致
- 魔法值提常量
- 缺少轻量 LLM 输出校验
- 简单路径或版本不一致
- 其他明显机械性修正

### ASK

- 安全、竞态、设计取舍
- 超过约 20 行的大修
- 删除功能
- 会改变明显用户可见行为的修改

## Suppressions

不要标以下噪音：

- 已经在当前 diff 里修掉的问题
- 只是在可读性上略显冗余、但没有真实风险的写法
- 纯“这条注释也许还能再解释一点”的低价值建议
