#!/usr/bin/env bash
# check-freeze.sh：/freeze 的 PreToolUse hook。
# 从标准输入读取 JSON，检查 file_path 是否位于 freeze 边界内。
# 越界时返回 {"permissionDecision":"deny","message":"..."}，否则返回 {}。
set -euo pipefail

INPUT=$(cat)

# 读取 freeze 状态文件位置。
STATE_DIR="${CLAUDE_PLUGIN_DATA:-$HOME/.gstack}"
FREEZE_FILE="$STATE_DIR/freeze-dir.txt"

# 尚未设置边界时默认放行。
if [ ! -f "$FREEZE_FILE" ]; then
  echo '{}'
  exit 0
fi

FREEZE_DIR=$(tr -d '[:space:]' < "$FREEZE_FILE")

if [ -z "$FREEZE_DIR" ]; then
  echo '{}'
  exit 0
fi

# 提取 tool_input.file_path。
FILE_PATH=$(printf '%s' "$INPUT" | grep -o '"file_path"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*:[[:space:]]*"//;s/"$//' || true)

if [ -z "$FILE_PATH" ]; then
  FILE_PATH=$(printf '%s' "$INPUT" | python3 -c 'import sys,json; print(json.loads(sys.stdin.read()).get("tool_input",{}).get("file_path",""))' 2>/dev/null || true)
fi

# 无法解析路径时不阻断，避免误伤正常流程。
if [ -z "$FILE_PATH" ]; then
  echo '{}'
  exit 0
fi

# 相对路径补成绝对路径。
case "$FILE_PATH" in
  /*) ;;
  *)
    FILE_PATH="$(pwd)/$FILE_PATH"
    ;;
esac

# 规范化路径表示。
FILE_PATH=$(printf '%s' "$FILE_PATH" | sed 's|/\+|/|g;s|/$||')

_resolve_path() {
  local _dir _base
  _dir="$(dirname "$1")"
  _base="$(basename "$1")"
  _dir="$(cd "$_dir" 2>/dev/null && pwd -P || printf '%s' "$_dir")"
  printf '%s/%s' "$_dir" "$_base"
}

FILE_PATH=$(_resolve_path "$FILE_PATH")
FREEZE_DIR=$(_resolve_path "$FREEZE_DIR")

case "$FILE_PATH" in
  "${FREEZE_DIR}/"*|"${FREEZE_DIR}")
    echo '{}'
    ;;
  *)
    mkdir -p ~/.gstack/analytics 2>/dev/null || true
    echo '{"event":"hook_fire","skill":"freeze","pattern":"boundary_deny","ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","repo":"'$(basename "$(git rev-parse --show-toplevel 2>/dev/null)" 2>/dev/null || echo "unknown")'"}' >> ~/.gstack/analytics/skill-usage.jsonl 2>/dev/null || true

    printf '{"permissionDecision":"deny","message":"[freeze] 已阻止：%s 不在 freeze 边界内（%s）。当前只允许编辑冻结目录中的文件。"}\n' "$FILE_PATH" "$FREEZE_DIR"
    ;;
esac
