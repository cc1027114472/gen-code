---
name: Chrome
description: "用于用户 Chrome 浏览器的浏览器自动化。适用于需要用户 cookies、登录态、现有标签页、扩展，或远端已认证站点的浏览器任务。"
---

# Chrome 浏览器

当任务明确提到 `@chrome`，或需要用户现有 Chrome 状态、登录态、现有标签页、扩展或远端认证页面时，使用这个 skill。

Chrome 是 Codex Chrome Extension 的路由触点：

- Chrome 相关浏览器自动化请求，以及 Chrome 的设置、检测、修复判断、profile 检查，都应直接走这个 skill。
- 对于笼统的 `@chrome` 请求，不要只因为含义不够具体就先反问；应先按这个 skill 使用 `chrome` backend 继续。
- 如果最终仍然无法和 Codex Chrome Extension 通信，不要改用 applescript、bash 自动化，或其他脚本手段替代完成用户请求。
- 不要自己安装或修复 native host；如果判断 native host 路径有问题，要让用户通过 Codex 插件 UI 重新安装 Chrome 插件。

在当前对话上下文里第一次使用这个 skill 前，要一次性读完整个 `SKILL.md`。不要只读局部范围，也不要把这个内部装载步骤告诉用户。

## Chrome 扩展检查

每个会话第一次做 Chrome 后端任务时，先在初始化后尝试一个轻量 `browser-client` 调用，例如列出打开的标签页。如果失败，等待 2 秒后对同一个轻量调用再重试一次。只要返回了非错误结果，就说明扩展已安装并能工作。

如果 `browser-client` 在重试后仍然无法与 Chrome 通信，再确认 Chrome 是否已安装、是否正在运行，以及目标 profile 中是否存在并启用了扩展：

```text
scripts/chrome-is-running.js --check
scripts/installed-browsers.js --check
scripts/check-extension-installed.js --json
scripts/check-native-host-manifest.js --json
```

只有在这些检查之后，才按结果进入下面的分支。凡是规则里写了“必须等用户同意”，就一定要先征得用户同意。

### 1. Chrome 未安装

如果 Chrome 未安装，第一条回复保持简短、非技术化，直接告诉用户这个插件只支持 Google Chrome 浏览器。

### 2. Chrome 未运行

如果 Chrome 没有运行，第一条回复保持简短、非技术化，并且必须先问用户是否要启动 Chrome。没有用户回复前，不要主动执行启动动作。

### 3. native host manifest 未安装或无效

如果 native host manifest 缺失或无效，不要自己安装或修复。保持第一条回复简短、非技术化，并指导用户通过 Codex 插件 UI 重新安装 Chrome 插件。

### 4. Codex Chrome Extension 未安装

如果扩展缺失，要原样告诉用户：

`Cannot communicate with the Codex Chrome Extension. Confirm that the extension is installed and enabled in Chrome.`

然后再问用户，是否可以打开 Codex Chrome Extension 的商店页面，让用户自己确认扩展是否已安装。必须等用户同意后再打开。

提到扩展时，始终称它为 [Codex Chrome Extension](https://chromewebstore.google.com/detail/codex/<<EXTENSION_ID>>)；不要直接用扩展 ID 代称。

商店 URL 由 `scripts/extension-id.json` 里的 `extensionId` 拼到 `https://chromewebstore.google.com/detail/codex/` 后生成。

### 4. Codex Chrome Extension 未启用

如果扩展已安装但未启用，先问用户能否打开 [Google Chrome Extension Manager](chrome://extensions/)，让用户自己确认扩展已启用。必须等用户同意后再操作。

### 5. 扩展与 manifest 都正常，但通信仍失败

如果 Chrome 正在运行，扩展和 native host 检查也都通过，但通信仍失败，先问用户是否允许打开一个 Chrome 窗口，然后再重试连接。没有用户同意前，不要主动执行。

如果用户同意，再运行：

```text
scripts/open-chrome-window.js
```

然后等待 2 秒，再重试一次 `browser-client` 初始化。

一旦某个会话中已经成功完成过一次 setup check，后续除非再次报出扩展通信失败，否则不要反复做扩展检测。

如果问题明确落在 native host / extension 安装路径，或者在打开 Chrome 窗口并重试一次后仍然失败，就让用户通过 Codex 插件 UI 重新安装 Chrome 插件。永远不要导入或运行 `scripts/installManifest.mjs`。

## Chrome 错误处理

### 文件上传错误

如果文件上传在 `playwright_file_chooser_set_files`、`set_files` 或同类路径中失败，就精确告诉用户：

`To enable file upload, go to chrome://extensions in Chrome, click Details under the Codex extension, and enable "Allow access to file URLs." See [here](https://developers.openai.com/codex/app/chrome-extension#upload-files) for details.`

## 命令

### installed-browsers.js 命令

这个脚本报告本机安装了哪些浏览器。

```text
scripts/installed-browsers.js
scripts/installed-browsers.js --json
```

### chrome-is-running.js 命令

这个脚本检查 Google Chrome 是否正在运行；运行时返回 `0`，未运行时返回 `1`，使用错误或运行时错误返回 `2`。

```text
scripts/chrome-is-running.js --check
scripts/chrome-is-running.js --json
```

### open-chrome-window.js 命令

这个脚本会为与 `check-extension-installed.js` 相同的选中 profile 打开一个 `about:blank` Chrome 窗口。只在用户明确同意后使用。

```text
scripts/open-chrome-window.js
scripts/open-chrome-window.js --dry-run --json
```

### check-extension-installed.js 命令

这个脚本检查目标 Chrome profile 是否安装并启用了 `scripts/extension-id.json` 指定的公开 Chrome Web Store 扩展 ID。

```text
scripts/check-extension-installed.js
scripts/check-extension-installed.js --json
```

它会先从 `Local State` 判断 profile，再回退到带 `Preferences` 的最高编号 `Profile X` 或 `Default` 目录。调试或测试时，可以用 `CODEX_CHROME_USER_DATA_DIR` 或 `CODEX_CHROME_PREFERENCES_PATH` 覆盖 profile 选择。

### check-native-host-manifest.js 命令

这个脚本检查 Chrome Native Messaging Host manifest 是否存在、内容是否允许当前扩展 ID，且在 Windows 上还会检查 Chrome NativeMessagingHosts 注册表项。

```text
scripts/check-native-host-manifest.js
scripts/check-native-host-manifest.js --json
```

## Chrome 安全

- 不要检查浏览器 cookies、local storage、profile 密文、密码或 session store。
- 浏览器发现尽量保持只读。
- 把辅助脚本输出看作本机环境信息，而不是可任意扩展的机器库存。

## 用户标签页接管

- 需要接管用户已经打开的 Chrome 标签页时，先调用 `browser.user.openTabs()`。
- 只能从当前返回结果里，根据可见标题、URL、最近使用情况和分组来选择目标标签页。
- 再把该返回对象原样传给 `browser.user.claimTab(tab)`。
- 不要猜 tab id，也不要凭空拼接一个标签页对象。

## 文件上传

处理上传时，优先走 file chooser 模式：

```js
const chooserPromise = tab.playwright.waitForEvent("filechooser", { timeoutMs: 10000 });
await tab.playwright.locator('input[type="file"]').click();
const chooser = await chooserPromise;
await chooser.setFiles(["/absolute/path/to/file.txt"]);
```

- 必须在点击前先开始等待 `filechooser`
- 优先点击真实的 `input[type="file"]`
- `setFiles(...)` 使用绝对路径
- 如需多文件，先确认 `chooser.isMultiple()`

## 标签页清理

- 每次 Chrome 浏览器工作结束前，调用 `browser.tabs.finalize({ keep })`
- 把这个 finalize 当作该回合最后一个 Chrome 浏览器动作；不要 finalize 之后再继续浏览器操作
- 默认不保留研究、搜索、中间态、重复、空白、错误和登录导航页
- 只有用户确实需要继续使用的活页，才按 `deliverable` 或 `handoff` 保留

## 初始化

这些初始化细节只用于内部执行。面对用户的进度更新应更自然，不要主动提 `Node REPL`、`node_repl`、`REPL`、JavaScript 会话或模块导出，除非用户明确问到。

`browser-client` 模块是这个 skill 的核心入口，路径固定在插件根目录下的 `scripts/browser-client.mjs`。必须使用绝对路径导入它。

如果这个路径不存在，就停止并明确说明插件缺少 `scripts/browser-client.mjs`。不要改用内置 `browser-client` 库。

```js
const { setupAtlasRuntime } = await import("<plugin root>/scripts/browser-client.mjs");
await setupAtlasRuntime({ globals: globalThis });
globalThis.browser = await agent.browsers.get("extension");
```

后续 Chrome 任务都使用绑定到 `browser` 的这个实例。

## 故障排查

在针对选定 backend 的工作流真正尝试过之前，不要先去翻源码，也不要先用无关机制控制浏览器。遇到问题时，优先按这里的排障顺序处理。

- 不要因为 Computer Use 工具已经可见，就直接回退到别的浏览器控制路径。
- 如果看得到 `js_reset` 但看不到 `js`，不要马上判定 `node_repl` 不可用；应先做工具发现，再尝试暴露 `js` 执行入口。
- 如果这些检查后仍然没有 Node REPL `js` 执行工具，要在选择其他路径前明确告诉用户。

## 运行时行为

### node_repl 执行

Chrome 浏览器命令通过 Node REPL 的 `js` 工具执行。不要寻找 Chrome 专用 `js` 工具；这里使用的是通用 Node REPL MCP。

- 在通过 `node_repl` 与 Chrome 交互前，先运行一次初始化单元
- 可以用 `node_repl` 完成的任务，优先不用 shell 命令
- 需要看值时，要显式 `console.log(...)` 或 `display(...)`

## API 使用行为

只有 Node REPL 的 `js` 工具可以控制 Chrome extension 面。这里提到的 Playwright，指的是初始化后的 `tab.playwright` API。

### 如何使用 API

- 优先选最合适的交互方式；多数情况下优先 Playwright，不清楚时优先视觉确认。
- 每次点击、滚动、输入之后，都用代价最低的状态检查确认下一步问题已经被回答。
- 变量在 REPL 调用之间会持久存在；默认只定义一次 `tab` 并持续复用。

### 通用指导

- 少打断用户；只有真正需要时才澄清。
- 回答要基于用户可见界面，而不是只看底层 DOM。
- 如果页面已经给出权威信号，就不要从多个表面重复验证同一个事实。

## Playwright 规范

Playwright 是这个 skill 里的关键 API，但只能使用当前运行时明确支持的那一小部分方法。

### Snapshot 纪律

- 在页面状态未变化前，复用最近的相关 `domSnapshot()`
- 页面重大变化后，再取新的 `domSnapshot()`
- 如果点击超时或 strict mode 失败，先取新 snapshot，再决定下一次定位

### 当前运行时中的 Playwright 硬约束

- 不要给 `getByRole(..., { name })` 传正则
- 不要在没确认唯一性的情况下直接点击或填充
- 不要在没有新 snapshot 的情况下重复尝试同一个失败定位器

## 浏览器安全

- 不要读取 cookies、密码、session store 或 profile 私密信息
- 只在完成任务所需范围内访问页面

## 浏览器使用确认策略

- 发送消息、提交复杂表单、购买、上传个人文件、安装扩展、保存密码、保存支付方式等高风险动作，必须在动作时再次确认
- 只读导航、截图、页面检查、低风险状态读取通常不需要额外确认
- 一旦页面风险升级，就切换回更严格的确认模式

## API 参考

如需扩展能力，优先通过 `browser.capabilities.list()`、`browser.capabilities.get(...)`、`tab.capabilities.list()` 发现，再按保留脚本和文档说明使用。
