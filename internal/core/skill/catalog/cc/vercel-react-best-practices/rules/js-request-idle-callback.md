---
title: Defer Non-Critical Work with requestIdleCallback
impact: MEDIUM
impactDescription: keeps UI responsive during background tasks
tags: javascript, performance, idle, scheduling, analytics
---

## 使用 requestIdleCallback 推迟非关键工作

**影响：中（在后台任务期间保持 UI 响应）**

使用`requestIdleCallback()`在浏览器空闲期间安排非关键工作。这使得主线程可以自由用于用户交互和动画，从而减少卡顿并提高感知性能。

**不正确（在用户交互期间阻塞主线程）：**

```typescript
function handleSearch(query: string) {
  const results = searchItems(query)
  setResults(results)

  // These block the main thread immediately
  analytics.track('search', { query })
  saveToRecentSearches(query)
  prefetchTopResults(results.slice(0, 3))
}
```

**正确（将非关键工作推迟到空闲时间）：**

```typescript
function handleSearch(query: string) {
  const results = searchItems(query)
  setResults(results)

  // Defer non-critical work to idle periods
  requestIdleCallback(() => {
    analytics.track('search', { query })
  })

  requestIdleCallback(() => {
    saveToRecentSearches(query)
  })

  requestIdleCallback(() => {
    prefetchTopResults(results.slice(0, 3))
  })
}
```

**所需工作超时：**

```typescript
// Ensure analytics fires within 2 seconds even if browser stays busy
requestIdleCallback(
  () => analytics.track('page_view', { path: location.pathname }),
  { timeout: 2000 }
)
```

**分块大型任务：**

```typescript
function processLargeDataset(items: Item[]) {
  let index = 0

  function processChunk(deadline: IdleDeadline) {
    // Process items while we have idle time (aim for <50ms chunks)
    while (index < items.length && deadline.timeRemaining() > 0) {
      processItem(items[index])
      index++
    }

    // Schedule next chunk if more items remain
    if (index < items.length) {
      requestIdleCallback(processChunk)
    }
  }

  requestIdleCallback(processChunk)
}
```

**针对不支持的浏览器提供后备：**

```typescript
const scheduleIdleWork = window.requestIdleCallback ?? ((cb: () => void) => setTimeout(cb, 1))

scheduleIdleWork(() => {
  // Non-critical work
})
```

**何时使用：**

- 分析和遥测
- 将状态保存到 localStorage/IndexedDB
- 为可能的下一步操作预取资源
- 处理非紧急数据转换
- 非关键功能的延迟初始化

**何时不使用：**

- 用户发起的需要立即反馈的操作
- 用户正在等待的渲染更新
- 时间敏感的操作
