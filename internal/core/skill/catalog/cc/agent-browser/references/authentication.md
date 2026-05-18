# 认证模式

登录流程、会话持久化、OAuth、2FA 与已认证浏览。

**相关内容：** 状态持久化细节见 [session-management.md](session-management.md)，快速开始见 [SKILL.md](../SKILL.md)。

## 目录

- [从你的浏览器导入认证状态](#从你的浏览器导入认证状态)
- [持久化 Profile](#持久化-profile)
- [会话持久化](#会话持久化)
- [基础登录流程](#基础登录流程)
- [保存认证状态](#保存认证状态)
- [恢复认证状态](#恢复认证状态)
- [OAuth / SSO 流程](#oauth--sso-流程)
- [双因素认证](#双因素认证)
- [HTTP Basic Auth](#http-basic-auth)
- [基于 Cookie 的认证](#基于-cookie-的认证)
- [处理 Token 刷新](#处理-token-刷新)
- [安全最佳实践](#安全最佳实践)

## 从你的浏览器导入认证状态

最快的认证方式，是复用你已经登录的 Chrome 会话中的 cookies。

**步骤 1：以远程调试模式启动 Chrome**

```bash
# macOS
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --remote-debugging-port=9222

# Linux
google-chrome --remote-debugging-port=9222

# Windows
"C:\Program Files\Google\Chrome\Application\chrome.exe" --remote-debugging-port=9222
```

像平时一样，在这个 Chrome 窗口里登录目标站点。

> **安全提示：** `--remote-debugging-port` 会在 localhost 上暴露完整浏览器控制能力。任何本地进程都可以连接并读取 cookies、执行 JS 等。只应在受信任的机器上使用，并在完成后关闭 Chrome。

**步骤 2：抓取认证状态**

```bash
# 自动发现正在运行的 Chrome，并保存其 cookies + localStorage
agent-browser --auto-connect state save ./my-auth.json
```

**步骤 3：在自动化中复用**

```bash
# 启动时加载认证状态
agent-browser --state ./my-auth.json open https://app.example.com/dashboard

# 或加载到现有会话中
agent-browser state load ./my-auth.json
agent-browser open https://app.example.com/dashboard
```

这适用于任何站点，包括具有复杂 OAuth 流程、SSO 或 2FA 的站点，只要 Chrome 中已经有有效的会话 cookies 即可。

> **安全提示：** 状态文件会以明文形式包含会话令牌。请把它们加入 `.gitignore`，在不再需要时删除，并设置 `AGENT_BROWSER_ENCRYPTION_KEY` 以实现静态加密。参见 [安全最佳实践](#安全最佳实践)。

**提示：** 可结合 `--session-name` 使用，让导入的认证状态在重启之间自动持久化：

```bash
agent-browser --session-name myapp state load ./my-auth.json
# 从现在开始，"myapp" 的状态会自动保存/恢复
```

## 持久化 Profile

使用 `--profile` 让 agent-browser 指向一个 Chrome 用户数据目录。这样无需显式保存/加载，就能在浏览器重启之间持久化所有内容（cookies、IndexedDB、service workers、cache）：

```bash
# 首次运行：登录一次
agent-browser --profile ~/.myapp-profile open https://app.example.com/login
# ... 完成登录流程 ...

# 后续所有运行：都已认证
agent-browser --profile ~/.myapp-profile open https://app.example.com/dashboard
```

不同项目或测试用户应使用不同路径：

```bash
agent-browser --profile ~/.profiles/admin open https://app.example.com
agent-browser --profile ~/.profiles/viewer open https://app.example.com
```

也可以通过环境变量设置：

```bash
export AGENT_BROWSER_PROFILE=~/.myapp-profile
agent-browser open https://app.example.com/dashboard
```

## 会话持久化

使用 `--session-name` 可按名称自动保存并恢复 cookies + localStorage，无需手动管理文件：

```bash
# 关闭时自动保存状态，下次启动时自动恢复
agent-browser --session-name twitter open https://twitter.com
# ... 登录流程 ...
agent-browser close  # 状态已保存到 ~/.agent-browser/sessions/

# 下次：自动恢复状态
agent-browser --session-name twitter open https://twitter.com
```

对静态存储的状态进行加密：

```bash
export AGENT_BROWSER_ENCRYPTION_KEY=$(openssl rand -hex 32)
agent-browser --session-name secure open https://app.example.com
```

## 基础登录流程

```bash
# 导航到登录页
agent-browser open https://app.example.com/login
agent-browser wait --load networkidle

# 获取表单元素
agent-browser snapshot -i
# Output: @e1 [input type="email"], @e2 [input type="password"], @e3 [button] "Sign In"

# 填写凭据
agent-browser fill @e1 "user@example.com"
agent-browser fill @e2 "password123"

# 提交
agent-browser click @e3
agent-browser wait --load networkidle

# 验证登录成功
agent-browser get url  # 应为 dashboard，而不是 login
```

## 保存认证状态

登录后保存状态以便复用：

```bash
# 先完成登录（见上文）
agent-browser open https://app.example.com/login
agent-browser snapshot -i
agent-browser fill @e1 "user@example.com"
agent-browser fill @e2 "password123"
agent-browser click @e3
agent-browser wait --url "**/dashboard"

# 保存已认证状态
agent-browser state save ./auth-state.json
```

## 恢复认证状态

通过加载已保存状态来跳过登录：

```bash
# 加载已保存的认证状态
agent-browser state load ./auth-state.json

# 直接访问受保护页面
agent-browser open https://app.example.com/dashboard

# 验证已认证
agent-browser snapshot -i
```

## OAuth / SSO 流程

针对 OAuth 重定向：

```bash
# 启动 OAuth 流程
agent-browser open https://app.example.com/auth/google

# 自动处理重定向
agent-browser wait --url "**/accounts.google.com**"
agent-browser snapshot -i

# 填写 Google 凭据
agent-browser fill @e1 "user@gmail.com"
agent-browser click @e2  # Next button
agent-browser wait 2000
agent-browser snapshot -i
agent-browser fill @e3 "password"
agent-browser click @e4  # Sign in

# 等待重定向回应用
agent-browser wait --url "**/app.example.com**"
agent-browser state save ./oauth-state.json
```

## 双因素认证

通过人工介入处理 2FA：

```bash
# 使用凭据登录
agent-browser open https://app.example.com/login --headed  # 显示浏览器
agent-browser snapshot -i
agent-browser fill @e1 "user@example.com"
agent-browser fill @e2 "password123"
agent-browser click @e3

# 等待用户在浏览器窗口中手动完成 2FA
echo "请在浏览器窗口中完成 2FA..."
agent-browser wait --url "**/dashboard" --timeout 120000

# 在 2FA 完成后保存状态
agent-browser state save ./2fa-state.json
```

## HTTP Basic Auth

针对使用 HTTP Basic Authentication 的站点：

```bash
# 在导航前设置凭据
agent-browser set credentials username password

# 访问受保护资源
agent-browser open https://protected.example.com/api
```

## 基于 Cookie 的认证

手动设置认证 cookies：

```bash
# 设置认证 cookie
agent-browser cookies set session_token "abc123xyz"

# 导航到受保护页面
agent-browser open https://app.example.com/dashboard
```

## 处理 Token 刷新

对于会过期的会话：

```bash
#!/bin/bash
# 负责处理 token 刷新的包装脚本

STATE_FILE="./auth-state.json"

# 尝试加载已有状态
if [[ -f "$STATE_FILE" ]]; then
    agent-browser state load "$STATE_FILE"
    agent-browser open https://app.example.com/dashboard

    # 检查会话是否仍然有效
    URL=$(agent-browser get url)
    if [[ "$URL" == *"/login"* ]]; then
        echo "会话已过期，重新认证..."
        # 在这里执行重新登录
    fi
fi
```

## 安全最佳实践

1. **绝不要提交状态文件** - 它们包含有效会话令牌
2. **对静态存储加密** - 设置 `AGENT_BROWSER_ENCRYPTION_KEY`
3. **使用一次性测试账号** - 尤其在 CI 或共享环境中
4. **完成后删除认证产物** - 状态文件、临时 profile、下载的敏感文件
5. **优先使用 `--session-name` 或 `--profile`** - 避免散落多个临时状态文件
