---
name: remote-tests
description: How to run tests using remote executor.
---

一些 codex 集成测试支持针对远程执行器运行。
这意味着当设置了 `CODEX_TEST_REMOTE_ENV` 环境变量时，它们会尝试在 `CODEX_TEST_REMOTE_ENV` 指向的 docker 容器中启动一个执行器进程，并在测试中使用它。

Docker 容器通过 `./scripts/test-remote-env.sh` 构建并初始化。

当前仅支持在 Linux 上运行远程测试，因此你需要使用 devbox 来运行它们。

你可以通过 `applied_devbox ls` 列出 devbox，选择名称中带有 `codex` 的那个。
通过 `ssh <devbox_name>` 连接到 devbox。
复用位于 `~/code/codex` 的同一个 codex 检出。如有需要，重置文件。多个检出会花更长时间构建，并占用更多空间。
检查远端和本地之间的 SHA 与已修改文件是否保持同步。
