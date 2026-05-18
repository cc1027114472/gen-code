---
name: agent-browser
description: 面向 AI agent 的浏览器自动化命令行工具。当用户需要与网站交互时使用，包括页面导航、填写表单、点击按钮、截图、提取数据、测试 Web 应用，或执行任何需要程序化网页交互的任务。触发场景包括“打开一个网站”“填写表单”“点击按钮”“截图”“抓取页面数据”“测试这个 Web 应用”“登录某个站点”“自动化浏览器操作”，或任何要求程序化网页交互的任务。
allowed-tools: Bash(npx agent-browser:*), Bash(agent-browser:*)
---

# 使用 agent-browser 进行浏览器自动化

这个 CLI 直接通过 CDP 使用 Chrome/Chromium。可通过 `npm i -g agent-browser`、`brew install agent-browser` 或 `cargo install agent-browser` 安装。运行 `agent-browser install` 下载 Chrome，运行 `agent-browser upgrade` 更新到最新版本。

## 核心工作流

每一次浏览器自动化都遵循这一模式：

1. **导航**：`agent-browser open <url>`
2. **快照**：`agent-browser snapshot -i`（获取类似 `@e1`、`@e2` 的元素引用）
3. **交互**：使用这些引用执行点击、填写、选择
4. **再次快照**：在导航或 DOM 变化后获取新的引用

```bash
agent-browser open https://example.com/form
agent-browser snapshot -i
# Output: @e1 [input type="email"], @e2 [input type="password"], @e3 [button] "Submit"

agent-browser fill @e1 "user@example.com"
agent-browser fill @e2 "password123"
agent-browser click @e3
agent-browser wait --load networkidle
agent-browser snapshot -i  # Check result
```

## 命令串联

命令可以通过 `&&` 在一次 shell 调用中串联执行。浏览器会通过后台守护进程在多个命令之间保持存活，因此串联调用是安全的，而且通常比拆成多个单独调用更高效。

```bash
# 在一次调用里串联 open + wait + snapshot
agent-browser open https://example.com && agent-browser wait --load networkidle && agent-browser snapshot -i

# 串联多个交互动作
agent-browser fill @e1 "user@example.com" && agent-browser fill @e2 "password123" && agent-browser click @e3

# 导航并截图
agent-browser open https://example.com && agent-browser wait --load networkidle && agent-browser screenshot page.png
```

**何时使用串联：** 当你不需要先读取中间命令的输出再继续时，可以使用 `&&`（例如 open + wait + screenshot）。当你必须先解析输出再决定下一步时，应分开执行（例如先 snapshot 发现 refs，再依据这些 refs 执行交互）。

## 认证处理

当自动化一个需要登录的网站时，选择最适合的方案：

**方案 1：从用户浏览器导入认证状态（最适合一次性任务）**

```bash
# 连接到用户正在运行的 Chrome（其已处于登录状态）
agent-browser --auto-connect state save ./auth.json
# 使用该认证状态
agent-browser --state ./auth.json open https://app.example.com/dashboard
```

状态文件会以明文存储会话令牌，因此应把它加入 `.gitignore`，并在不再需要时删除。可设置 `AGENT_BROWSER_ENCRYPTION_KEY` 实现静态加密。

**方案 2：持久化 profile（最适合重复性任务）**

```bash
# 首次运行：手动或自动完成登录
agent-browser --profile ~/.myapp open https://app.example.com/login
# ... fill credentials, submit ...

# 之后所有运行：都已处于登录状态
agent-browser --profile ~/.myapp open https://app.example.com/dashboard
```

**方案 3：会话名称（自动保存/恢复 cookies 和 localStorage）**

```bash
agent-browser --session-name myapp open https://app.example.com/login
# ... login flow ...
agent-browser close  # 自动保存状态

# 下次运行：自动恢复状态
agent-browser --session-name myapp open https://app.example.com/dashboard
```

**方案 4：认证保险箱（凭据加密保存，按名称登录）**

```bash
echo "$PASSWORD" | agent-browser auth save myapp --url https://app.example.com/login --username user --password-stdin
agent-browser auth login myapp
```

`auth login` 会先导航并等待登录表单相关 selector 出现，再执行填写与点击，因此在存在延迟加载的 SPA 登录页上更可靠。

**方案 5：状态文件（手动保存/加载）**

```bash
# 登录后：
agent-browser state save ./auth.json
# 在未来某次会话中：
agent-browser state load ./auth.json
agent-browser open https://app.example.com/dashboard
```

关于 OAuth、2FA、基于 cookie 的认证以及 token 刷新模式，请参见 [references/authentication.md](references/authentication.md)。

## 常用命令

```bash
# Navigation
agent-browser open <url>              # 导航（别名：goto, navigate）
agent-browser close                   # 关闭浏览器
agent-browser close --all             # 关闭所有活动会话

# Snapshot
agent-browser snapshot -i             # 交互元素 + refs（推荐）
agent-browser snapshot -s "#selector" # 限定到 CSS selector 范围

# Interaction (use @refs from snapshot)
agent-browser click @e1               # 点击元素
agent-browser click @e1 --new-tab     # 点击并在新标签页打开
agent-browser fill @e2 "text"         # 清空后输入文本
agent-browser type @e2 "text"         # 不清空直接输入
agent-browser select @e1 "option"     # 选择下拉选项
agent-browser check @e1               # 勾选复选框
agent-browser press Enter             # 按键
agent-browser keyboard type "text"    # 在当前焦点处输入（无 selector）
agent-browser keyboard inserttext "text"  # 不触发按键事件直接插入
agent-browser scroll down 500         # 滚动页面
agent-browser scroll down 500 --selector "div.content"  # 在指定容器内滚动

# Get information
agent-browser get text @e1            # 获取元素文本
agent-browser get url                 # 获取当前 URL
agent-browser get title               # 获取页面标题
agent-browser get cdp-url             # 获取 CDP WebSocket URL

# Wait
agent-browser wait @e1                # 等待元素
agent-browser wait --load networkidle # 等待网络空闲
agent-browser wait --url "**/page"    # 等待 URL 模式
agent-browser wait 2000               # 等待若干毫秒
agent-browser wait --text "Welcome"    # 等待文本出现（子串匹配）
agent-browser wait --fn "!document.body.innerText.includes('Loading...')"  # 等待文本消失
agent-browser wait "#spinner" --state hidden  # 等待元素消失

# Downloads
agent-browser download @e1 ./file.pdf          # 点击元素触发下载
agent-browser wait --download ./output.zip     # 等待任意下载完成
agent-browser --download-path ./downloads open <url>  # 设定默认下载目录

# Network
agent-browser network requests                 # 检查已跟踪请求
agent-browser network requests --type xhr,fetch  # 按资源类型过滤
agent-browser network requests --method POST   # 按 HTTP 方法过滤
agent-browser network requests --status 2xx    # 按状态码过滤（200, 2xx, 400-499）
agent-browser network request <requestId>      # 查看完整请求/响应详情
agent-browser network route "**/api/*" --abort  # 阻止匹配请求
agent-browser network har start                # 开始 HAR 录制
agent-browser network har stop ./capture.har   # 停止并保存 HAR 文件

# Viewport & Device Emulation
agent-browser set viewport 1920 1080          # 设置视口大小（默认：1280x720）
agent-browser set viewport 1920 1080 2        # 2x retina（相同 CSS 尺寸，更高分辨率截图）
agent-browser set device "iPhone 14"          # 模拟设备（视口 + user agent）

# Capture
agent-browser screenshot              # 截图到临时目录
agent-browser screenshot --full       # 全页截图
agent-browser screenshot --annotate   # 带编号元素标记的标注截图
agent-browser screenshot --screenshot-dir ./shots  # 保存到自定义目录
agent-browser screenshot --screenshot-format jpeg --screenshot-quality 80
agent-browser pdf output.pdf          # 保存为 PDF

# Live preview / streaming
agent-browser stream enable           # 在自动选择的端口上启动运行时 WebSocket 流
agent-browser stream enable --port 9223  # 绑定指定 localhost 端口
agent-browser stream status           # 查看启用状态、端口、连接和 screencasting 状态
agent-browser stream disable          # 停止运行时流并删除 .stream 元数据文件

# Clipboard
agent-browser clipboard read                      # 读取剪贴板文本
agent-browser clipboard write "Hello, World!"     # 写入剪贴板文本
agent-browser clipboard copy                      # 复制当前选区
agent-browser clipboard paste                     # 粘贴剪贴板内容

# Dialogs (alert, confirm, prompt, beforeunload)
# 默认情况下，alert 和 beforeunload 对话框会被自动接受，因此不会阻塞 agent。
# confirm 和 prompt 仍需要显式处理。
# 可使用 --no-auto-dialog 关闭自动处理。
agent-browser dialog accept              # 接受对话框
agent-browser dialog accept "my input"   # 带文本接受 prompt 对话框
agent-browser dialog dismiss             # 取消对话框
agent-browser dialog status              # 查看当前是否有打开的对话框

# Diff (compare page states)
agent-browser diff snapshot                          # 比较当前状态与上一次 snapshot
agent-browser diff snapshot --baseline before.txt    # 比较当前状态与保存文件
agent-browser diff screenshot --baseline before.png  # 视觉像素 diff
agent-browser diff url <url1> <url2>                 # 比较两个页面
agent-browser diff url <url1> <url2> --wait-until networkidle  # 自定义等待策略
agent-browser diff url <url1> <url2> --selector "#main"  # 限定元素范围
```

## 参考资料

| 文件 | 内容 |
|------|------|
| [references/commands.md](references/commands.md) | 带完整参数的命令参考 |
| [references/snapshot-refs.md](references/snapshot-refs.md) | refs 生命周期、失效规则与排障 |
| [references/session-management.md](references/session-management.md) | 并行会话、状态持久化与并发抓取 |
| [references/authentication.md](references/authentication.md) | 登录流程、OAuth、2FA 处理与状态复用 |
| [references/video-recording.md](references/video-recording.md) | 面向调试和留档的录屏流程 |
| [references/profiling.md](references/profiling.md) | 使用 Chrome DevTools profiling 做性能分析 |
| [references/proxy-support.md](references/proxy-support.md) | 代理配置、地理位置测试与轮换代理 |
