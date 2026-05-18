# 性能分析

在浏览器自动化过程中采集 Chrome DevTools 性能 profile，用于性能分析。

**相关内容：** 完整命令参考见 [commands.md](commands.md)，快速开始见 [SKILL.md](../SKILL.md)。

## 目录

- [基础分析](#基础分析)
- [Profiler 命令](#profiler-命令)
- [分类](#分类)
- [使用场景](#使用场景)
- [输出格式](#输出格式)
- [查看 Profile](#查看-profile)
- [限制](#限制)

## 基础分析

```bash
# 开始 profiling
agent-browser profiler start

# 执行动作
agent-browser navigate https://example.com
agent-browser click "#button"
agent-browser wait 1000

# 停止并保存
agent-browser profiler stop ./trace.json
```

## Profiler 命令

```bash
# 使用默认分类开始 profiling
agent-browser profiler start

# 使用自定义 trace categories 开始 profiling
agent-browser profiler start --categories "devtools.timeline,v8.execute,blink.user_timing"

# 停止 profiling 并保存到文件
agent-browser profiler stop ./trace.json
```

## 分类

`--categories` 标志接受逗号分隔的 Chrome trace categories 列表。默认分类包括：

- `devtools.timeline` - 标准 DevTools 性能 trace
- `v8.execute` - JavaScript 执行耗时
- `blink` - 渲染器事件
- `blink.user_timing` - `performance.mark()` / `performance.measure()` 调用
- `latencyInfo` - 输入到延迟的跟踪
- `renderer.scheduler` - 任务调度与执行
- `toplevel` - 广谱基础事件

还会包含若干 `disabled-by-default-*` 分类，用于更详细的时间线、调用栈和 V8 CPU profiling 数据。

## 使用场景

### 诊断页面加载缓慢

```bash
agent-browser profiler start
agent-browser navigate https://app.example.com
agent-browser wait --load networkidle
agent-browser profiler stop ./page-load-profile.json
```

### 分析用户交互

```bash
agent-browser navigate https://app.example.com
agent-browser profiler start
agent-browser click "#submit"
agent-browser wait 2000
agent-browser profiler stop ./interaction-profile.json
```

### CI 性能回归检查

```bash
#!/bin/bash
agent-browser profiler start
agent-browser navigate https://app.example.com
agent-browser wait --load networkidle
agent-browser profiler stop "./profiles/build-${BUILD_ID}.json"
```

## 输出格式

输出文件是 Chrome Trace Event 格式的 JSON 文件：

```json
{
  "traceEvents": [
    { "cat": "devtools.timeline", "name": "RunTask", "ph": "X", "ts": 12345, "dur": 100, ... },
    ...
  ],
  "metadata": {
    "clock-domain": "LINUX_CLOCK_MONOTONIC"
  }
}
```

`metadata.clock-domain` 字段会根据宿主平台设置（Linux 或 macOS）。在 Windows 上该字段会被省略。

## 查看 Profile

把输出 JSON 文件加载到以下任一工具中：

- **Chrome DevTools**：Performance 面板 > Load profile（Ctrl+Shift+I > Performance）
- **Perfetto UI**：[https://ui.perfetto.dev/](https://ui.perfetto.dev/) - 将 JSON 文件拖放进去
- **Trace Viewer**：任意 Chromium 浏览器中的 `chrome://tracing`

## 限制

- 仅适用于基于 Chromium 的浏览器（Chrome、Edge）。不支持 Firefox 或 WebKit。
- profiling 激活期间，trace 数据会在内存中累积（上限 500 万事件）。在关注区域分析完后应尽快停止。
- 停止时的数据采集有 30 秒超时。如果浏览器无响应，stop 命令可能失败。
