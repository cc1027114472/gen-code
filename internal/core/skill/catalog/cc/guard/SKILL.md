---
name: guard
version: 0.1.0
description: 同时启用破坏性命令警告与目录级编辑限制，提供最大化的安全模式。
allowed-tools:
  - Bash
  - Read
  - AskUserQuestion
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "bash ${CLAUDE_SKILL_DIR}/../careful/bin/check-careful.sh"
          statusMessage: "检查破坏性命令..."
    - matcher: "Edit"
      hooks:
        - type: command
          command: "bash ${CLAUDE_SKILL_DIR}/../freeze/bin/check-freeze.sh"
          statusMessage: "检查 freeze 边界..."
    - matcher: "Write"
      hooks:
        - type: command
          command: "bash ${CLAUDE_SKILL_DIR}/../freeze/bin/check-freeze.sh"
          statusMessage: "检查 freeze 边界..."
---

# `/guard` 最大安全模式

同时启用破坏性命令警告和目录级编辑限制。这是把 `/careful` 与 `/freeze` 合并成一次性启用的组合型 skill。

**依赖说明：** 这个 skill 会直接引用 sibling `careful` 与 `freeze` 包中的 hook 脚本。它们必须已经随同 `cc` copied-skill baseline 一起存在。

```bash
mkdir -p ~/.gstack/analytics
echo '{"skill":"guard","ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","repo":"'$(basename "$(git rev-parse --show-toplevel 2>/dev/null)" 2>/dev/null || echo "unknown")'"}' >> ~/.gstack/analytics/skill-usage.jsonl 2>/dev/null || true
```

## 设置步骤

先询问用户要把编辑限制在哪个目录，使用 `AskUserQuestion`：

- 问题：`Guard 模式：要把编辑限制在哪个目录？破坏性命令警告会始终开启，所选路径之外的文件将被阻止编辑。`
- 输入方式：文本输入，由用户直接提供路径

用户给出目录路径后，按以下步骤执行：

1. 解析为绝对路径：

```bash
FREEZE_DIR=$(cd "<user-provided-path>" 2>/dev/null && pwd)
echo "$FREEZE_DIR"
```

2. 追加尾部斜杠并写入 freeze 状态文件：

```bash
FREEZE_DIR="${FREEZE_DIR%/}/"
STATE_DIR="${CLAUDE_PLUGIN_DATA:-$HOME/.gstack}"
mkdir -p "$STATE_DIR"
echo "$FREEZE_DIR" > "$STATE_DIR/freeze-dir.txt"
echo "Freeze boundary set: $FREEZE_DIR"
```

然后告诉用户：

- `Guard 模式已启用。`
- `1. 破坏性命令警告：rm -rf、DROP TABLE、force-push 等操作在执行前会先警告，你可以明确覆盖。`
- `2. 编辑边界：文件编辑已限制到 <path>/，边界之外的编辑会被直接阻止。`
- `要移除编辑边界，运行 /unfreeze；要完全退出保护模式，结束当前会话。`

## 保护范围

完整的破坏性命令模式与安全例外沿用 `/careful`。

完整的编辑边界检查与阻止逻辑沿用 `/freeze`。
