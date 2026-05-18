# Browser Capability: viewport

浏览器视口覆盖控制。正常浏览器初始化时不要主动设置视口；绝大多数任务都应该直接使用现有的默认 `1280x720` 视口。只有在用户明确要求某个尺寸、要求测试响应式断点或设备大小，或者在没有特定视口时任务无法被正确回答的情况下，才调用 `set()`。不要只是为了让截图更大、更好看，或一次性装下更多内容，就调整浏览器尺寸。优先使用默认视口、普通截图或整页截图。如果你临时设置了视口，而用户又没有要求保留它，就在结束前调用 `reset()`。

```ts
const capability = await browser.capabilities.get("viewport");

interface ViewportSize {
  height: number;
  width: number;
}

interface ViewportBrowserCapability {
  reset(): Promise<void>; // 清除显式视口覆盖，回到默认浏览器尺寸。
  set(options: ViewportSize): Promise<void>; // 应用显式浏览器视口覆盖。
}
```
