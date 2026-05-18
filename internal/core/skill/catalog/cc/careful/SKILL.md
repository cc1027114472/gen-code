---
name: careful
version: 0.1.0
description: 对破坏性命令增加安全护栏，在执行前发出警告并允许用户决定是否继续。
allowed-tools:
  - Bash
  - Read
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "bash ${CLAUDE_SKILL_DIR}/bin/check-careful.sh"
          statusMessage: "检查破坏性命令..."
---

# `/careful` 破坏性命令护栏

安全模式现已启用。每一条 Bash 命令在执行前都会检查是否包含高风险破坏性模式；如果命中规则，需要先向用户发出警告，再决定是否继续。

```bash
mkdir -p ~/.gstack/analytics
echo '{"skill":"careful","ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","repo":"'$(basename "$(git rev-parse --show-toplevel 2>/dev/null)" 2>/dev/null || echo "unknown")'"}' >> ~/.gstack/analytics/skill-usage.jsonl 2>/dev/null || true
```

## 保护范围

| 模式 | 示例 | 风险 |
| --- | --- | --- |
| `rm -rf` / `rm -r` / `rm --recursive` | `rm -rf /var/data` | 递归删除文件 |
| `DROP TABLE` / `DROP DATABASE` | `DROP TABLE users;` | 数据永久丢失 |
| `TRUNCATE` | `TRUNCATE orders;` | 清空表数据 |
| `git push --force` / `-f` | `git push -f origin main` | 改写远端历史 |
| `git reset --hard` | `git reset --hard HEAD~3` | 丢失未提交修改 |
| `git checkout .` / `git restore .` | `git checkout .` | 丢失未提交修改 |
| `kubectl delete` | `kubectl delete pod` | 影响线上资源 |
| `docker rm -f` / `docker system prune` | `docker system prune -a` | 删除容器或镜像 |

## 安全例外

以下模式默认允许，不额外警告：

- `rm -rf node_modules`
- `rm -rf .next`
- `rm -rf dist`
- `rm -rf __pycache__`
- `rm -rf .cache`
- `rm -rf build`
- `rm -rf .turbo`
- `rm -rf coverage`

## 工作方式

hook 会从工具输入 JSON 中读取命令内容，匹配上面的高风险模式。若命中规则，就返回 `permissionDecision: "ask"` 和稳定警告文案；用户仍可明确覆盖并继续执行。

结束会话或切换到新的会话后，这个保护会随会话一同失效。
