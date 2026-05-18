---
name: Browser
description: "用于 Codex 应用内浏览器的浏览器自动化。适用于打开、导航、检查、测试、点击、输入、截图，或验证 localhost、127.0.0.1、::1、file:// 与当前应用内浏览器标签页。"
---

# 应用内浏览器

当任务需要控制 Codex 应用内浏览器时使用这个 skill，例如检查页面、导航、测试本地应用、点击、输入、截图，以及读取当前可见页面状态。完成初始化后，必须选择 `iab` 浏览器。

如果当前会话里已经列出了这个插件，就把它视为浏览器工作前的必读说明。在判断 Browser 不可用、或回退到独立 Playwright / Computer Use 之前，先打开并遵循这个 skill。

不要因为 Computer Use MCP 调用看起来更直接，或者表面上更容易触发，就跳过这个 skill。Computer Use 工具可见，并不代表它是这个浏览器面的首选控制路径。

在当前对话上下文里第一次使用这个 skill 前，要一次性读完整个 `SKILL.md`。不要用局部范围读取代替完整阅读，也不要把这个内部装载步骤告诉用户。

## 初始化

这些初始化细节只用于内部执行。面对用户的进度更新要更自然，不要主动提 `Node REPL`、`node_repl`、`REPL`、JavaScript 会话或模块导出，除非用户明确问到这些概念。如果需要初始化或恢复，只描述成“连接浏览器”或“重试浏览器连接”。

`browser-client` 模块是这个 skill 的核心入口，路径固定在插件根目录下的 `scripts/browser-client.mjs`。必须使用绝对路径导入它。

如果这个路径不存在，就停止并明确说明插件缺少 `scripts/browser-client.mjs`。不要改用内置 `browser-client` 库。

通过 Node REPL 的 `js` 执行工具运行一次初始化代码：

```js
const { setupAtlasRuntime } = await import("<plugin root>/scripts/browser-client.mjs");
await setupAtlasRuntime({ globals: globalThis });
globalThis.browser = await agent.browsers.get("iab");
```

后续浏览器任务都使用绑定到 `browser` 变量的这个实例。

## 故障排查

在针对选定 backend 的工作流真正尝试过之前，不要先去翻源码，也不要先用无关机制控制浏览器。遇到问题时，优先按这里的排障顺序处理。

- 不要因为 Computer Use 工具已经可见，就直接回退到 Computer Use。
- 如果看得到 `js_reset` 但看不到 `js`，不要马上判定 `node_repl` 不可用；应先做工具发现，再尝试暴露 `js` 执行入口。
- 如果在这些检查之后仍然没有 Node REPL `js` 执行工具，要在选择其他浏览器控制路径前明确告诉用户这一点。
- 如果 `node_repl` 完全不可用，也要先明确告诉用户，再谈回退。

## 运行时行为

### node_repl 执行

浏览器命令通过 Node REPL 的 `js` 工具执行。不要寻找浏览器专用的 `js` 工具；这里使用的是通用 Node REPL MCP。

- 在通过 `node_repl` 与浏览器交互前，先运行一次初始化单元。初始化完成前，不要假定 `display` 或 `tab` 已经存在。
- 如果任务可以直接通过 `node_repl` 完成，优先用它，而不是 shell 命令。
- `node_repl` 不会自动打印最后一个表达式；如果需要查看值，要显式使用 `console.log(...)`、`display(...)` 或同类输出方式。

#### 运行时模式

- 复用已有的 `tab` 绑定；如果 `tab` 已存在，不要为了同一个标签页重新获取一个新变量。
- 运行时初始化和初始 `tab` 获取通常每个会话只做一次，除非内核被重置。
- 如果内核被重置、句柄失效，或者 `tab` 丢失，优先通过 `browser.tabs.list()` 和 `browser.tabs.get(tab.id)` 恢复当前会话标签页。
- 每个浏览器任务开始时，在完成初始化后立刻调用 `await browser.nameSession("...")` 给当前会话命名，再去打开或选择标签页。
- 每个会话第一次建立浏览器单元时，先初始化并获取 `tab`，不要在 `tab` 尚未存在时直接写 `tab = ...`。

#### 首个浏览器单元

如果初始化过程可能重试，使用可重入的初始化单元：

```js
if (!globalThis.agent) {
  const { setupAtlasRuntime } = await import("<plugin root>/scripts/browser-client.mjs");
  await setupAtlasRuntime({ globals: globalThis });
}
if (!globalThis.browser) {
  globalThis.browser = await agent.browsers.get("iab");
}
await browser.nameSession("任务短名");
if (typeof tab === "undefined") {
  globalThis.tab = await browser.tabs.selected();
}
```

如果当前浏览器没有活动标签页，就改为：

```js
if (!globalThis.agent) {
  const { setupAtlasRuntime } = await import("<plugin root>/scripts/browser-client.mjs");
  await setupAtlasRuntime({ globals: globalThis });
}
if (!globalThis.browser) {
  globalThis.browser = await agent.browsers.get("iab");
}
await browser.nameSession("任务短名");
if (typeof tab === "undefined") {
  globalThis.tab = await browser.tabs.new();
}
```

初始化完成后持续复用同一个 `tab` 绑定，不要在重试之间来回切换 `tab = ...`、`let tab = ...`、`const tab = ...` 与 `globalThis.tab = ...`。

#### 变量复用

- 如果前一个 `node_repl` 单元里已经创建了 `tab`，后续继续复用它维持状态。
- 如果你要让主 `tab` 变量有意指向另一个标签页，应只声明一次，再重赋值。
- 如果需要同时保留两个标签页，就给第二个标签页起一个新的描述性变量名。
- 不要仅仅为了避免复用而重新查一次同一个标签页并起 `tab2` 之类的名字。
- 不要在没有命名冲突的情况下把整个单元包进额外块作用域。

#### 文件处理

在 `node_repl` 里需要文件读写时，优先直接使用 Node 的文件系统库，例如 `node:fs/promises`。

#### 浏览器交互

开始浏览器工作时使用上面的受保护初始化单元。它会创建浏览器工作所需的顶层 `agent` 和 `display`。

## API 使用行为

浏览器直接控制能力通过 `browser-client` 运行时暴露在 `agent.browsers.*` API 上。

只有 Node REPL 的 `js` 工具可以控制应用内浏览器。不要改用外部 MCP 浏览器控制工具、独立浏览器自动化服务器，或其他浏览器 skill。这里提到的 Playwright，指的是初始化后的 `tab.playwright` API。

### 如何使用 API

- 优先选最合适的交互方式；多数情况下优先 Playwright，不清楚时优先视觉确认。
- 每次点击、滚动、输入之后，都用代价最低的状态检查确认下一步问题已经被回答。
- 需要定位器真值时优先新的 DOM snapshot；需要视觉确认时优先截图；不要默认两者都取。
- 如果截图是用户应该看到的产物，要在最终回复里以内联 Markdown 图片展示，而不是只给裸链接。
- 变量在 REPL 调用之间会持久存在；默认只定义一次 `tab` 并持续复用。

### 通用指导

- 尽量少打断用户；只有在确实需要时才澄清。
- 回答要基于用户可见界面，而不是只基于底层 DOM 顺序。
- 如果通过节点 ID 就能明确定位，不必把事情复杂化。
- 如果标签页已经在目标 URL，不要重复 `goto` 同一个 URL，除非你有意刷新。
- 测试本地站点代码变更后，如果框架不支持热更新或热更新关闭，要先 `tab.reload()` 再验证。
- 不要穷举一堆猜测 URL、搜索参数或站内候选查询。

## Playwright 规范

Playwright 是这个 skill 里的关键 API，但只能使用当前运行时明确支持的那一小部分方法。

### Snapshot 纪律

- 在页面状态未变化前，复用最近的相关 `domSnapshot()`。
- 发生导航、菜单开关、模态框开关、过滤器变化等重大状态切换后，再取新的 `domSnapshot()`。
- 如果点击超时、strict mode 失败，或选择器解析失败，先取新 snapshot，再决定下一次定位。
- 只根据最新 snapshot 构造定位器，不要猜标签名、accessible name 或选择器。
- 不要把整页大段文本或大范围逐元素读取当作探索手段。

### 当前运行时中的 Playwright 硬约束

- 不要给 `getByRole(..., { name })` 传正则；这里只用普通字符串。
- 不要在没有先确认 `count()` 的前提下使用 `.first()`、`.last()`、`.nth()`。
- 如果唯一性不明显，不要在确认元素唯一前就点击、填充或按键。
- 不要在没有新 snapshot 的情况下重复尝试同一个失败定位器。
- 不要把猜测出来的定位器当成探索探针。

### 推荐交互流程

- 先理解当前屏幕。
- 再缩小到目标区域。
- 然后只做回答下一步问题所需的最小操作。
- 一旦页面上已经出现权威信号，就不要反复从别的表面重复验证同一事实。

## 浏览器安全

- 不要读取浏览器 cookies、local storage 密文、密码或用户 profile 私密内容。
- 只在完成任务所需范围内访问页面。
- 把浏览器可见信息视为用户上下文，而不是可任意扩展搜集的库存。

## 浏览器使用确认策略

### 适用范围

这个确认策略适用于应用内浏览器自动化，不适用于无关工具。

### 确认模式

- 必须用户亲手完成：支付、验证码、明确账户授权、敏感安全设置变更等。
- 即使预授权也必须动作时再确认：发送消息、提交复杂表单、购买、上传个人文件、删除重要数据、安装扩展等。
- 可以接受预授权：范围清晰、风险低、且用户已经明确同意的重复动作。
- 不需要确认：只读导航、截图、页面检查、低风险状态读取。

### 确认卫生规则

- 说明即将执行的动作和影响面。
- 不要把一次授权偷换成更宽的长期授权。
- 一旦页面风险升级，就回到更严格的确认模式。

## API 参考

如需浏览器能力扩展，优先通过 `browser.capabilities.list()`、`browser.capabilities.get(...)` 与 `tab.capabilities.list()` 发现，再按保留文档说明使用。
