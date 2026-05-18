# 浏览器操作配方

本文件列出具体的浏览器操作，以及它们最接近的工具等价物。

## 处理后读取层与原始抓取层

当任务暂时还不需要实时浏览器状态时，这两层应先于完整浏览器接入。

### 处理后读取

在以下情况下使用处理后读取：

- URL 已经明确
- 页面主要是文章、文档或 PDF 内容
- 你更需要页面文本，而不是原始响应形态

优先选项：

- 标准处理后页面优先使用 `web.open`
- 只有在 token 节省很重要、且任务仍能接受精度损失时，才使用外部 markdown 镜像

### 原始抓取

在以下情况下使用原始抓取：

- 你需要源 HTML、headers、JSON-LD 或直接资源
- 渲染后文本与原始响应数据之间的区别很重要

优先选项：

- 使用带原生抓取工具的 `shell_command`，例如 `Invoke-WebRequest` 或 `curl.exe`
- 当响应很大、直接内联输出会很吵时，把结果保存到文件

## 页面与标签页控制

| 目标 | 等价工具或动作 |
| --- | --- |
| 列出已打开标签页 | `chrome-devtools.list_pages` |
| 新建一个标签页 | `chrome-devtools.new_page` |
| 切换到现有标签页 | `chrome-devtools.select_page` |
| 关闭你自己打开的标签页 | `chrome-devtools.close_page` |
| 在当前标签页访问某个 URL | `chrome-devtools.navigate_page` |
| 返回上一页 | 带返回语义的 `chrome-devtools.navigate_page` |

## 检查与提取

| 目标 | 等价工具或动作 |
| --- | --- |
| 运行任意页面脚本 | `chrome-devtools.evaluate_script` |
| 读取可访问的页面结构 | `chrome-devtools.take_snapshot` |
| 获取渲染后的图像状态 | `chrome-devtools.take_screenshot` |
| 检查 Network 中当前选中的请求 | 不带 `reqid` 的 `chrome-devtools.get_network_request` |
| 检查更广泛的网络流量 | `chrome-devtools.list_network_requests` |
| 检查控制台输出 | `chrome-devtools.list_console_messages` |

## 交互

| 目标 | 等价工具或动作 |
| --- | --- |
| 点击元素 | `chrome-devtools.click` |
| 悬停元素 | `chrome-devtools.hover` |
| 填写输入框 | `chrome-devtools.fill` 或 `fill_form` |
| 上传文件 | `chrome-devtools.upload_file` |
| 拖拽 | `chrome-devtools.drag` |
| 仅键盘操作 | `chrome-devtools.press_key` |

## 滚动与等待模式

Chrome DevTools 工具集没有专门的 `scroll` 端点。使用以下任一方式：

- `evaluate_script(() => window.scrollBy(0, 1200))`
- `evaluate_script(() => window.scrollTo(0, document.body.scrollHeight))`
- 当键盘式交互更贴近真实用户时，使用 `press_key("PageDown")`

滚动或进行重大交互后：

- 如果你已经知道接下来要等的文本或状态，使用 `wait_for`
- 否则重新执行 `take_snapshot`

## 性能、内存与审计

| 目标 | 等价工具或动作 |
| --- | --- |
| 启动性能 trace | `chrome-devtools.performance_start_trace` |
| 停止并保存 trace | `chrome-devtools.performance_stop_trace` |
| 分析高亮性能洞察 | `chrome-devtools.performance_analyze_insight` |
| 捕获内存快照 | `chrome-devtools.take_memory_snapshot` |
| 运行 Lighthouse 审计 | `chrome-devtools.lighthouse_audit` |

## 何时切换到 Playwright

在以下情况下切换到 Playwright：

- 任务需要干净的浏览器上下文
- 实时 Chrome 会话不可用，且无法接入
- 工作本质是浏览器自动化，而不是复用当前已登录会话

Playwright 是回退方案，不是用户当前浏览器会话或状态的等价替代品。
