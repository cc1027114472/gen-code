---
name: unfreeze
version: 0.1.0
description: 清除由 freeze 设置的编辑边界，让当前会话恢复为可编辑所有目录。
allowed-tools:
  - Bash
  - Read
---

# `/unfreeze` 清除 freeze 边界

移除由 `/freeze` 设置的编辑限制，让当前会话重新可以编辑所有目录中的文件。

```bash
mkdir -p ~/.gstack/analytics
echo '{"skill":"unfreeze","ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","repo":"'$(basename "$(git rev-parse --show-toplevel 2>/dev/null)" 2>/dev/null || echo "unknown")'"}' >> ~/.gstack/analytics/skill-usage.jsonl 2>/dev/null || true
```

## 清除边界

```bash
STATE_DIR="${CLAUDE_PLUGIN_DATA:-$HOME/.gstack}"
if [ -f "$STATE_DIR/freeze-dir.txt" ]; then
  PREV=$(cat "$STATE_DIR/freeze-dir.txt")
  rm -f "$STATE_DIR/freeze-dir.txt"
  echo "Freeze boundary cleared (was: $PREV). Edits are now allowed everywhere."
else
  echo "No freeze boundary was set."
fi
```

执行后，把结果直接告诉用户。

要注意的是，`/freeze` 注册过的 hooks 仍然会保留到当前会话结束；只是当状态文件不存在时，它们会默认放行全部编辑。如果之后还要重新限制目录，再次运行 `/freeze` 即可。
