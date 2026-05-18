---
name: freeze
version: 0.1.0
description: 将当前会话的文件编辑限制在指定目录内，阻止越界修改。
allowed-tools:
  - Bash
  - Read
  - AskUserQuestion
hooks:
  PreToolUse:
    - matcher: "Edit"
      hooks:
        - type: command
          command: "bash ${CLAUDE_SKILL_DIR}/bin/check-freeze.sh"
          statusMessage: "检查 freeze 边界..."
    - matcher: "Write"
      hooks:
        - type: command
          command: "bash ${CLAUDE_SKILL_DIR}/bin/check-freeze.sh"
          statusMessage: "检查 freeze 边界..."
---

# `/freeze` 限制编辑目录

将当前会话的文件编辑锁定到一个指定目录。任何指向边界之外文件的 `Edit` 或 `Write` 操作都会被直接阻止，而不是仅给出提醒。

```bash
mkdir -p ~/.gstack/analytics
echo '{"skill":"freeze","ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","repo":"'$(basename "$(git rev-parse --show-toplevel 2>/dev/null)" 2>/dev/null || echo "unknown")'"}' >> ~/.gstack/analytics/skill-usage.jsonl 2>/dev/null || true
```

## 设置步骤

先询问用户要限制到哪个目录，使用 `AskUserQuestion`：

- 问题：`要把编辑限制在哪个目录？该路径之外的文件将被阻止编辑。`
- 输入方式：文本输入，由用户直接提供路径

拿到路径后，按下面步骤执行：

1. 解析为绝对路径：

```bash
FREEZE_DIR=$(cd "<user-provided-path>" 2>/dev/null && pwd)
echo "$FREEZE_DIR"
```

2. 追加尾部斜杠并写入状态文件：

```bash
FREEZE_DIR="${FREEZE_DIR%/}/"
STATE_DIR="${CLAUDE_PLUGIN_DATA:-$HOME/.gstack}"
mkdir -p "$STATE_DIR"
echo "$FREEZE_DIR" > "$STATE_DIR/freeze-dir.txt"
echo "Freeze boundary set: $FREEZE_DIR"
```

然后告诉用户：编辑已限制到 `<path>/`。这个目录之外的 `Edit` 和 `Write` 会被阻止；如果要更换边界，再运行一次 `/freeze`；如果要解除限制，运行 `/unfreeze` 或结束会话。

## 工作方式

hook 会从工具输入 JSON 里读取 `file_path`，判断目标路径是否位于 freeze 目录之内。若不在边界内，就返回 `permissionDecision: "deny"` 并附带稳定阻止文案。

freeze 边界通过状态文件在本会话中持续生效；hook 脚本会在每次 `Edit` 或 `Write` 调用时重新读取该状态。

## 说明

- 目录尾部的 `/` 用来避免 `/src` 误匹配 `/src-old`
- freeze 只限制 `Edit` 和 `Write`，不会限制 `Read`、`Bash`、搜索或枚举操作
- 这是一层防误改护栏，不是安全沙箱；例如 Bash 内部的 `sed` 仍可能改动其他文件
- 要解除限制，运行 `/unfreeze` 或结束当前会话
