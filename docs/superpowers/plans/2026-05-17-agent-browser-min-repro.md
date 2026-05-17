# agent-browser Windows 最小复现与环境报告

## Summary

本报告用于记录 `agent-browser` 在当前 Windows 开发机上的最小复现现象，方便后续：

- 向上游提 issue
- 判断是否继续投入排查
- 为本项目切换或保留替代工具提供依据

结论先说：

- 当前 `gen-code` 产品页面不是问题主体
- `agent-browser` 在这台 Windows 机器上的核心问题更像是 CLI / daemon / session 元数据一致性问题
- 当前可用替代方案已经存在，建议继续使用 Python Playwright 作为本项目浏览器验收主路径

## Environment

- OS: `Windows 10 22H2` (`Microsoft Windows NT 10.0.19045.0`)
- PowerShell: `5.1.19041.3803`
- Node.js: `v25.2.1`
- npm: `11.6.2`
- Python: `3.12.7`
- Go: `go1.24.0 windows/amd64`
- agent-browser: `0.27.0`

命令入口：

- `C:\Users\Administrator\AppData\Roaming\npm\agent-browser.ps1`
- 实际二进制：
  `C:\Users\Administrator\AppData\Roaming\npm\node_modules\agent-browser\bin\agent-browser-win32-x64.exe`

本机已安装 agent-browser 管理的 Chrome 二进制：

- `C:\Users\Administrator\.agent-browser\browsers\chrome-147.0.7727.56`
- `C:\Users\Administrator\.agent-browser\browsers\chrome-148.0.7778.167`

## Minimal Repro Commands

### Case 1: `open about:blank`

先清理 session：

```powershell
agent-browser --session repro1 close --all
```

输出：

```text
No active sessions
```

然后执行：

```powershell
agent-browser --session repro1 --debug open about:blank
```

实际结果：

- 进程退出码为 `0`
- 命令几乎不输出任何内容
- 没有预期中的成功提示

继续验证：

```powershell
agent-browser --session repro1 get url
```

输出：

```text
about:blank
```

说明：

- `open about:blank` 并非完全失败
- 但 CLI 成功反馈缺失，表现异常

### Case 2: `close` 后 session 元数据不一致

在 `repro1` 已存在的情况下执行：

```powershell
agent-browser --session repro1 close
```

输出：

```text
✓ Browser closed
```

然后检查 session：

```powershell
agent-browser session list
```

输出：

```text
Active sessions:
  repro1
```

继续检查浏览器进程数：

```powershell
Get-Process | Where-Object {
  $_.ProcessName -eq 'chrome' -and $_.Path -like "$env:USERPROFILE\.agent-browser\browsers\*"
} | Measure-Object | Select-Object -ExpandProperty Count
```

输出：

```text
0
```

说明：

- CLI 提示浏览器已关闭
- 实际 agent-browser 自带 Chrome 进程也确实为 `0`
- 但 `session list` 仍显示 `repro1`
- 这是明显的 session 元数据或 daemon 状态清理不一致

### Case 3: `close` 后继续访问同 session

在上一步关闭后继续执行：

```powershell
agent-browser --session repro1 get url
```

实际结果：

- 退出码为 `0`
- 命令无输出
- 行为既不像明确报错，也不像返回有效页面状态

这进一步说明：

- session 已不处于健康可用状态
- 但 CLI 没有把它明确标记为不可用

### Case 4: 本地页面打开本身不是核心问题

执行：

```powershell
agent-browser --debug open http://127.0.0.1:5174/
```

输出：

```text
✓ Gen Code Desktop
  http://127.0.0.1:5174/
```

说明：

- 本地页面可被打开
- 当前核心问题不是 `gen-code` 页面不可达
- 更像是 `agent-browser` session 生命周期和 CLI 反馈本身不稳定

### Case 5: `doctor` 不能稳定收敛

执行：

```powershell
agent-browser doctor
```

实际结果：

- 超过 60 秒未完成
- 本次采样中命令超时退出

说明：

- 官方自检命令本身在当前环境也不稳定
- 这会降低继续本地自修的性价比

## Session Metadata Evidence

当前用户目录下可见的 session 元数据样本：

- `C:\Users\Administrator\.agent-browser\repro1.engine`
- `C:\Users\Administrator\.agent-browser\repro1.pid`
- `C:\Users\Administrator\.agent-browser\repro1.port`
- `C:\Users\Administrator\.agent-browser\repro1.stream`
- `C:\Users\Administrator\.agent-browser\repro1.version`

即使 `close` 后浏览器进程已归零，这些 session 元数据仍然存在，且 `session list` 仍可看到该 session。

## Root Cause Hypothesis

当前最合理的根因假设是：

1. `agent-browser` 在 Windows 下的 session 关闭流程没有完整清理 session 元数据或 daemon 注册状态
2. CLI 某些命令在“浏览器已关但 session 仍被登记”为活动状态时，没有返回明确错误，而是无输出退出
3. `open about:blank` 的空输出和 `doctor` 超时，很可能与同一类 session/daemon 状态异常有关

这更像 `agent-browser` 工具层问题，而不是：

- `gen-code` 前端页面问题
- 本地 app-server 不可用
- Chrome 二进制未安装

## Impact on This Project

对本项目的直接影响：

- 会干扰 browser 验收结论判断
- 容易把工具环境问题误判成页面实时刷新问题
- 不适合作为当前 desktop 审批 / rollback / SSE 链路的主验收工具

## Recommended Workaround

当前建议固定为：

1. `agent-browser` 不再作为 `gen-code` 主验收路径
2. 本项目浏览器级验收继续使用 Python Playwright
3. 若后续仍需保留 `agent-browser`，仅用于一次性、低可靠性要求的辅助检查
4. 若要继续追上游问题，建议把本报告连同以下命令输出一并提交：
   - `agent-browser --version`
   - `agent-browser --session repro1 close --all`
   - `agent-browser --session repro1 --debug open about:blank`
   - `agent-browser --session repro1 get url`
   - `agent-browser --session repro1 close`
   - `agent-browser session list`
   - `agent-browser doctor`

## Status

- `gen-code` 页面链路：正常
- `agent-browser` Windows 环境：不稳定，且存在 session 清理异常
- 当前项目默认验收工具：Python Playwright
