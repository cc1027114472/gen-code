# 会话管理

多个相互隔离的浏览器会话，支持状态持久化与并发浏览。

**相关内容：** 登录模式见 [authentication.md](authentication.md)，快速开始见 [SKILL.md](../SKILL.md)。

## 目录

- [命名会话](#命名会话)
- [会话隔离属性](#会话隔离属性)
- [会话状态持久化](#会话状态持久化)
- [常见模式](#常见模式)
- [默认会话](#默认会话)
- [会话清理](#会话清理)
- [最佳实践](#最佳实践)

## 命名会话

使用 `--session` 标志隔离浏览器上下文：

```bash
# 会话 1：认证流程
agent-browser --session auth open https://app.example.com/login

# 会话 2：公共浏览（独立 cookies、存储）
agent-browser --session public open https://example.com

# 命令按会话隔离
agent-browser --session auth fill @e1 "user@example.com"
agent-browser --session public get text body
```

## 会话隔离属性

每个会话都拥有独立的：

- Cookies
- LocalStorage / SessionStorage
- IndexedDB
- Cache
- 浏览历史
- 打开的标签页

## 会话状态持久化

### 保存会话状态

```bash
# 保存 cookies、存储和认证状态
agent-browser state save /path/to/auth-state.json
```

### 加载会话状态

```bash
# 恢复已保存的状态
agent-browser state load /path/to/auth-state.json

# 继续使用已认证会话
agent-browser open https://app.example.com/dashboard
```

### 状态文件内容

```json
{
  "cookies": [...],
  "localStorage": {...},
  "sessionStorage": {...},
  "origins": [...]
}
```

## 常见模式

### 复用已认证会话

```bash
#!/bin/bash
# 保存一次登录状态，之后多次复用

STATE_FILE="/tmp/auth-state.json"

# 检查是否已有保存的状态
if [[ -f "$STATE_FILE" ]]; then
    agent-browser state load "$STATE_FILE"
    agent-browser open https://app.example.com/dashboard
else
    # 执行登录
    agent-browser open https://app.example.com/login
    agent-browser snapshot -i
    agent-browser fill @e1 "$USERNAME"
    agent-browser fill @e2 "$PASSWORD"
    agent-browser click @e3
    agent-browser wait --load networkidle

    # 保存供后续复用
    agent-browser state save "$STATE_FILE"
fi
```

### 并发抓取

```bash
#!/bin/bash
# 并发抓取多个站点

# 启动所有会话
agent-browser --session site1 open https://site1.com &
agent-browser --session site2 open https://site2.com &
agent-browser --session site3 open https://site3.com &
wait

# 分别提取内容
agent-browser --session site1 get text body > site1.txt
agent-browser --session site2 get text body > site2.txt
agent-browser --session site3 get text body > site3.txt

# 清理
agent-browser --session site1 close
agent-browser --session site2 close
agent-browser --session site3 close
```

### A/B 测试会话

```bash
# 测试不同用户体验
agent-browser --session variant-a open "https://app.com?variant=a"
agent-browser --session variant-b open "https://app.com?variant=b"

# 对比
agent-browser --session variant-a screenshot /tmp/variant-a.png
agent-browser --session variant-b screenshot /tmp/variant-b.png
```

## 默认会话

当省略 `--session` 时，命令会使用默认会话：

```bash
# 这些命令都作用于同一个默认会话
agent-browser open https://example.com
agent-browser snapshot -i
agent-browser close  # 关闭默认会话
```

## 会话清理

```bash
# 关闭指定会话
agent-browser --session auth close

# 列出活动会话
agent-browser session list
```

## 最佳实践

### 1. 使用有语义的会话名

```bash
# GOOD：用途明确
agent-browser --session github-auth open https://github.com
agent-browser --session docs-scrape open https://docs.example.com

# AVOID：过于泛化
agent-browser --session s1 open https://github.com
```

### 2. 始终做清理

```bash
# 用完后关闭会话
agent-browser --session auth close
agent-browser --session scrape close
```

### 3. 安全处理状态文件

```bash
# 不要提交状态文件（其中包含认证令牌！）
echo "*.auth-state.json" >> .gitignore

# 使用后删除
rm /tmp/auth-state.json
```

### 4. 为长会话设置超时

```bash
# 给自动化脚本设置超时
timeout 60 agent-browser --session long-task get text body
```
