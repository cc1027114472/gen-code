# Browser Capability: visibility

浏览器可见性控制。使用 `set(true)` 把浏览器显式展示给用户，使用 `set(false)` 把它隐藏到后台，使用 `get()` 检查当前是否可见。默认优先让浏览器工作保持在后台；只有在用户明确要求看见浏览器，或者实时观看对任务非常重要时，才把它显示出来。为了验证浏览器行为而截图时，能在进度更新里展示就尽量展示；除非用户明确只要文字，否则最终回复里应使用 Markdown 图片语法把相关截图内联展示出来。

```ts
const capability = await browser.capabilities.get("visibility");

interface VisibilityBrowserCapability {
  get(): Promise<boolean>; // 读取浏览器当前是否展示给用户。
  set(visible: boolean): Promise<void>; // 设置浏览器是否展示给用户。
}
```
