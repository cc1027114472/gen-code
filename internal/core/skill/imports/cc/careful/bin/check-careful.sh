#!/usr/bin/env bash
# check-careful.sh：/careful 的 PreToolUse hook。
# 从标准输入读取 JSON，检查 Bash 命令是否包含破坏性模式。
# 命中时返回 {"permissionDecision":"ask","message":"..."}，否则返回 {}。
set -euo pipefail

# 读取标准输入中的工具调用 JSON。
INPUT=$(cat)

# 提取 tool_input.command。
# 先使用 grep/sed 处理常见场景，再在必要时退回 Python 解析转义引号。
CMD=$(printf '%s' "$INPUT" | grep -o '"command"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*:[[:space:]]*"//;s/"$//' || true)

if [ -z "$CMD" ]; then
  CMD=$(printf '%s' "$INPUT" | python3 -c 'import sys,json; print(json.loads(sys.stdin.read()).get("tool_input",{}).get("command",""))' 2>/dev/null || true)
fi

# 无法提取命令时放行，避免因解析失败误拦截。
if [ -z "$CMD" ]; then
  echo '{}'
  exit 0
fi

# 统一转成小写，便于做不区分大小写的 SQL 关键字检查。
CMD_LOWER=$(printf '%s' "$CMD" | tr '[:upper:]' '[:lower:]')

# 安全例外：仅删除常见构建产物时不警告。
if printf '%s' "$CMD" | grep -qE 'rm\s+(-[a-zA-Z]*r[a-zA-Z]*\s+|--recursive\s+)' 2>/dev/null; then
  SAFE_ONLY=true
  RM_ARGS=$(printf '%s' "$CMD" | sed -E 's/.*rm\s+(-[a-zA-Z]+\s+)*//;s/--recursive\s*//')
  for target in $RM_ARGS; do
    case "$target" in
      */node_modules|node_modules|*/\.next|\.next|*/dist|dist|*/__pycache__|__pycache__|*/\.cache|\.cache|*/build|build|*/\.turbo|\.turbo|*/coverage|coverage)
        ;;
      -*)
        ;;
      *)
        SAFE_ONLY=false
        break
        ;;
    esac
  done
  if [ "$SAFE_ONLY" = true ]; then
    echo '{}'
    exit 0
  fi
fi

WARN=""
PATTERN=""

if printf '%s' "$CMD" | grep -qE 'rm\s+(-[a-zA-Z]*r|--recursive)' 2>/dev/null; then
  WARN="高风险操作：检测到递归删除命令（rm -r）。该操作会永久删除文件。"
  PATTERN="rm_recursive"
fi

if [ -z "$WARN" ] && printf '%s' "$CMD_LOWER" | grep -qE 'drop\s+(table|database)' 2>/dev/null; then
  WARN="高风险操作：检测到 SQL DROP。该操作会永久删除数据库对象。"
  PATTERN="drop_table"
fi

if [ -z "$WARN" ] && printf '%s' "$CMD_LOWER" | grep -qE '\btruncate\b' 2>/dev/null; then
  WARN="高风险操作：检测到 SQL TRUNCATE。该操作会删除表中的全部数据。"
  PATTERN="truncate"
fi

if [ -z "$WARN" ] && printf '%s' "$CMD" | grep -qE 'git\s+push\s+.*(-f\b|--force)' 2>/dev/null; then
  WARN="高风险操作：git force-push 会改写远端历史，其他协作者可能丢失工作。"
  PATTERN="git_force_push"
fi

if [ -z "$WARN" ] && printf '%s' "$CMD" | grep -qE 'git\s+reset\s+--hard' 2>/dev/null; then
  WARN="高风险操作：git reset --hard 会丢弃全部未提交修改。"
  PATTERN="git_reset_hard"
fi

if [ -z "$WARN" ] && printf '%s' "$CMD" | grep -qE 'git\s+(checkout|restore)\s+\.' 2>/dev/null; then
  WARN="高风险操作：该命令会丢弃工作区中的未提交修改。"
  PATTERN="git_discard"
fi

if [ -z "$WARN" ] && printf '%s' "$CMD" | grep -qE 'kubectl\s+delete' 2>/dev/null; then
  WARN="高风险操作：kubectl delete 会移除 Kubernetes 资源，可能影响线上环境。"
  PATTERN="kubectl_delete"
fi

if [ -z "$WARN" ] && printf '%s' "$CMD" | grep -qE 'docker\s+(rm\s+-f|system\s+prune)' 2>/dev/null; then
  WARN="高风险操作：Docker 强制删除或清理会移除容器或缓存镜像。"
  PATTERN="docker_destructive"
fi

if [ -n "$WARN" ]; then
  mkdir -p ~/.gstack/analytics 2>/dev/null || true
  echo '{"event":"hook_fire","skill":"careful","pattern":"'"$PATTERN"'","ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","repo":"'$(basename "$(git rev-parse --show-toplevel 2>/dev/null)" 2>/dev/null || echo "unknown")'"}' >> ~/.gstack/analytics/skill-usage.jsonl 2>/dev/null || true

  WARN_ESCAPED=$(printf '%s' "$WARN" | sed 's/"/\\"/g')
  printf '{"permissionDecision":"ask","message":"[careful] %s"}\n' "$WARN_ESCAPED"
else
  echo '{}'
fi
