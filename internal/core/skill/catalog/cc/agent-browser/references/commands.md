# 命令参考

agent-browser 全部命令的完整参考。快速开始和常见模式请见 [SKILL.md](../SKILL.md)。

## 导航

```bash
agent-browser open <url>      # 导航到 URL（别名：goto, navigate）
                              # 支持：https://、http://、file://、about:、data://
                              # 未提供协议时会自动补上 https://
agent-browser back            # 后退
agent-browser forward         # 前进
agent-browser reload          # 刷新页面
agent-browser close           # 关闭浏览器（别名：quit, exit）
agent-browser connect 9222    # 通过 CDP 端口连接浏览器
```

## 快照（页面分析）

```bash
agent-browser snapshot            # 完整无障碍树
agent-browser snapshot -i         # 仅交互元素（推荐）
agent-browser snapshot -c         # 紧凑输出
agent-browser snapshot -d 3       # 限制深度为 3
agent-browser snapshot -s "#main" # 将范围限定到 CSS selector
```

## 交互（使用 snapshot 里的 @refs）

```bash
agent-browser click @e1           # 点击
agent-browser click @e1 --new-tab # 点击并在新标签页打开
agent-browser dblclick @e1        # 双击
agent-browser focus @e1           # 聚焦元素
agent-browser fill @e2 "text"     # 清空后输入
agent-browser type @e2 "text"     # 不清空直接输入
agent-browser press Enter         # 按键（别名：key）
agent-browser press Control+a     # 组合键
agent-browser keydown Shift       # 按下按键
agent-browser keyup Shift         # 松开按键
agent-browser hover @e1           # 悬停
agent-browser check @e1           # 勾选复选框
agent-browser uncheck @e1         # 取消勾选复选框
agent-browser select @e1 "value"  # 选择下拉项
agent-browser select @e1 "a" "b"  # 选择多个下拉项
agent-browser scroll down 500     # 滚动页面（默认：向下 300px）
agent-browser scrollintoview @e1  # 将元素滚动到视图中（别名：scrollinto）
agent-browser drag @e1 @e2        # 拖放
agent-browser upload @e1 file.pdf # 上传文件
```

## 获取信息

```bash
agent-browser get text @e1        # 获取元素文本
agent-browser get html @e1        # 获取 innerHTML
agent-browser get value @e1       # 获取输入值
agent-browser get attr @e1 href   # 获取属性
agent-browser get title           # 获取页面标题
agent-browser get url             # 获取当前 URL
agent-browser get cdp-url         # 获取 CDP WebSocket URL
agent-browser get count ".item"   # 统计匹配元素数量
agent-browser get box @e1         # 获取边界框
agent-browser get styles @e1      # 获取计算样式（字体、颜色、背景等）
```

## 检查状态

```bash
agent-browser is visible @e1      # 检查是否可见
agent-browser is enabled @e1      # 检查是否可用
agent-browser is checked @e1      # 检查是否已勾选
```

## 截图与 PDF

```bash
agent-browser screenshot          # 保存到临时目录
agent-browser screenshot path.png # 保存到指定路径
agent-browser screenshot --full   # 全页截图
agent-browser pdf output.pdf      # 保存为 PDF
```

## 视频录制

```bash
agent-browser record start ./demo.webm    # 开始录制
agent-browser click @e1                   # 执行动作
agent-browser record stop                 # 停止并保存视频
agent-browser record restart ./take2.webm # 停止当前录制并开启新的录制
```

## 等待

```bash
agent-browser wait @e1                     # 等待元素
agent-browser wait 2000                    # 等待毫秒数
agent-browser wait --text "Success"        # 等待文本（或 -t）
agent-browser wait --url "**/dashboard"    # 等待 URL 模式（或 -u）
agent-browser wait --load networkidle      # 等待网络空闲（或 -l）
agent-browser wait --fn "window.ready"     # 等待 JS 条件（或 -f）
```

## 鼠标控制

```bash
agent-browser mouse move 100 200      # 移动鼠标
agent-browser mouse down left         # 按下按钮
agent-browser mouse up left           # 松开按钮
agent-browser mouse wheel 100         # 滚轮滚动
```

## 语义定位器（refs 的替代方案）

```bash
agent-browser find role button click --name "Submit"
agent-browser find text "Sign In" click
agent-browser find text "Sign In" click --exact      # 仅精确匹配
agent-browser find label "Email" fill "user@test.com"
agent-browser find placeholder "Search" type "query"
agent-browser find alt "Logo" click
agent-browser find title "Close" click
agent-browser find testid "submit-btn" click
agent-browser find first ".item" click
agent-browser find last ".item" click
agent-browser find nth 2 "a" hover
```

## 浏览器设置

```bash
agent-browser set viewport 1920 1080          # 设置视口大小
agent-browser set viewport 1920 1080 2        # 2x retina（同 CSS 尺寸，更高分辨率截图）
agent-browser set device "iPhone 14"          # 模拟设备
agent-browser set geo 37.7749 -122.4194       # 设置地理位置（别名：geolocation）
agent-browser set offline on                  # 切换离线模式
agent-browser set headers '{"X-Key":"v"}'     # 额外 HTTP headers
agent-browser set credentials user pass       # HTTP basic auth（别名：auth）
agent-browser set media dark                  # 模拟配色方案
agent-browser set media light reduced-motion  # 浅色模式 + reduced motion
```

## Cookies 与存储

```bash
agent-browser cookies                     # 获取全部 cookies
agent-browser cookies set name value      # 设置 cookie
agent-browser cookies clear               # 清空 cookies
agent-browser storage local               # 获取全部 localStorage
agent-browser storage local key           # 获取指定 key
agent-browser storage local set k v       # 设置值
agent-browser storage local clear         # 清空全部
```

## 网络

```bash
agent-browser network route <url>              # 拦截请求
agent-browser network route <url> --abort      # 阻断请求
agent-browser network route <url> --body '{}'  # Mock 响应
agent-browser network unroute [url]            # 移除路由规则
agent-browser network requests                 # 查看已跟踪请求
agent-browser network requests --filter api    # 过滤请求
```

## 标签页与窗口

```bash
agent-browser tab                 # 列出标签页
agent-browser tab new [url]       # 新建标签页
agent-browser tab 2               # 按索引切换到标签页
agent-browser tab close           # 关闭当前标签页
agent-browser tab close 2         # 按索引关闭标签页
agent-browser window new          # 新建窗口
```

## Frame

```bash
agent-browser frame "#iframe"     # 通过 CSS selector 切换到 iframe
agent-browser frame @e3           # 通过元素 ref 切换到 iframe
agent-browser frame main          # 返回主 frame
```

### Iframe 支持

在执行快照时会自动检测 iframe。当主 frame 执行 snapshot 时，`Iframe` 节点会被解析，其内容会以内联方式出现在输出中（只展开一层嵌套；iframe 中的 iframe 不会继续展开）。

```bash
agent-browser snapshot -i
# @e3 [Iframe] "payment-frame"
#   @e4 [input] "Card number"
#   @e5 [button] "Pay"

# 直接交互，iframe 中的 refs 已可用
agent-browser fill @e4 "4111111111111111"
agent-browser click @e5

# 或切换 frame 上下文，以便做局部快照
agent-browser frame @e3               # 使用元素 ref 切换
agent-browser snapshot -i             # 仅对该 iframe 做快照
agent-browser frame main              # 返回主 frame
```

`frame` 命令支持：

- **元素 refs** - `frame @e3` 会把 ref 解析为 iframe 元素
- **CSS selectors** - `frame "#payment-iframe"` 会按 selector 查找 iframe
- **Frame 名称/URL** - 会与浏览器 frame 树进行匹配

## 对话框

默认情况下，`alert` 和 `beforeunload` 对话框会被自动接受，因此不会阻塞 agent。`confirm` 和 `prompt` 仍然需要显式处理。可使用 `--no-auto-dialog` 禁用这一行为。

```bash
agent-browser dialog accept [text]  # 接受对话框
agent-browser dialog dismiss        # 关闭/取消对话框
agent-browser dialog status         # 检查当前是否存在打开的对话框
```

## JavaScript

```bash
agent-browser eval "document.title"          # 仅简单表达式
agent-browser eval -b "<base64>"             # 任意 JavaScript（base64 编码）
agent-browser eval --stdin                   # 从 stdin 读取脚本
```

对复杂场景，应使用 `-b`/`--base64` 或 `--stdin`，这样执行更可靠。带嵌套引号和特殊字符的 shell 转义很容易出错。

```bash
# 先把脚本做 base64 编码，然后：
agent-browser eval -b "ZG9jdW1lbnQucXVlcnlTZWxlY3RvcignW3NyYyo9Il9uZXh0Il0nKQ=="

# 或使用带 heredoc 的 stdin 方式执行多行脚本：
cat <<'EOF' | agent-browser eval --stdin
const links = document.querySelectorAll('a');
Array.from(links).map(a => a.href);
EOF
```

## 状态管理

```bash
agent-browser state save auth.json    # 保存 cookies、存储与认证状态
agent-browser state load auth.json    # 恢复已保存状态
```

## 全局选项

```bash
agent-browser --session <name> ...    # 隔离浏览器会话
agent-browser --json ...              # 以 JSON 输出，便于解析
agent-browser --headed ...            # 显示浏览器窗口（非 headless）
agent-browser --full ...              # 全页截图（-f）
agent-browser --cdp <port> ...        # 通过 Chrome DevTools Protocol 连接
agent-browser -p <provider> ...       # 云浏览器 provider（--provider）
agent-browser --proxy <url> ...       # 使用代理服务器
agent-browser --proxy-bypass <hosts>  # 需要绕过代理的主机
agent-browser --headers <json> ...    # 作用于 URL 源站的 HTTP headers
agent-browser --executable-path <p>   # 自定义浏览器可执行文件
agent-browser --extension <path> ...  # 加载浏览器扩展（可重复）
agent-browser --ignore-https-errors   # 忽略 SSL 证书错误
agent-browser --help                  # 显示帮助（-h）
agent-browser --version               # 显示版本（-V）
agent-browser <command> --help        # 显示某个命令的详细帮助
```

## 调试

```bash
agent-browser --headed open example.com   # 显示浏览器窗口
agent-browser --cdp 9222 snapshot         # 通过 CDP 端口连接
agent-browser connect 9222                # 另一种连接方式：connect 命令
agent-browser console                     # 查看控制台消息
agent-browser console --clear             # 清空控制台
agent-browser errors                      # 查看页面错误
agent-browser errors --clear              # 清空页面错误
agent-browser highlight @e1               # 高亮元素
agent-browser inspect                     # 为当前会话打开 Chrome DevTools
agent-browser trace start                 # 开始录制 trace
agent-browser trace stop trace.zip        # 停止并保存 trace
agent-browser profiler start              # 开始 Chrome DevTools profiling
agent-browser profiler stop trace.json    # 停止并保存 profile
```

## 环境变量

```bash
AGENT_BROWSER_SESSION="mysession"            # 默认会话名
AGENT_BROWSER_EXECUTABLE_PATH="/path/chrome" # 自定义浏览器路径
AGENT_BROWSER_EXTENSIONS="/ext1,/ext2"       # 逗号分隔的扩展路径
AGENT_BROWSER_PROVIDER="browserbase"         # 云浏览器 provider
AGENT_BROWSER_STREAM_PORT="9223"             # 覆盖 WebSocket streaming 端口（默认：由系统分配）
AGENT_BROWSER_HOME="/path/to/agent-browser"  # 自定义安装位置
```
