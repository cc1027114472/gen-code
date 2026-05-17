---
name: test-tui
description: Guide for testing Codex TUI interactively
---

你可以启动并使用 Codex TUI 来验证更改。

重要说明：

以交互方式启动。
启动进程时始终设置 `RUST_LOG="trace"`。
传入 `-c log_dir=<some_temp_dir>` 参数，将日志写入特定目录，以帮助调试。
以编程方式发送测试消息时，先发送文本，再在单独一次写入中发送 Enter（不要在一次突发写入中同时发送文本 + Enter）。
使用 `just codex` target 来运行，即 `just codex -c ...`
