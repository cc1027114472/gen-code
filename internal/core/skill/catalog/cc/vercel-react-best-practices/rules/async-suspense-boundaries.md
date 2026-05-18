---
title: Strategic Suspense Boundaries
impact: HIGH
impactDescription: faster initial paint
tags: async, suspense, streaming, layout-shift
---

## 战略悬念边界

不要在返回 JSX 之前等待异步组件中的数据，而是使用 Suspense 边界在数据加载时更快地显示包装器 UI。

**不正确（包装器被数据获取阻止）：**

```tsx
async function Page() {
  const data = await fetchData() // Blocks entire page
  
  return (
    <div>
      <div>Sidebar</div>
      <div>Header</div>
      <div>
        <DataDisplay data={data} />
      </div>
      <div>Footer</div>
    </div>
  )
}
```

即使只有中间部分需要数据，整个布局也会等待数据。

**正确（包装器立即显示，数据流入）：**

```tsx
function Page() {
  return (
    <div>
      <div>Sidebar</div>
      <div>Header</div>
      <div>
        <Suspense fallback={<Skeleton />}>
          <DataDisplay />
        </Suspense>
      </div>
      <div>Footer</div>
    </div>
  )
}

async function DataDisplay() {
  const data = await fetchData() // Only blocks this component
  return <div>{data.content}</div>
}
```

侧边栏、页眉和页脚立即呈现。只有DataDisplay等待数据。

**替代方案（跨组件共享承诺）：**

```tsx
function Page() {
  // Start fetch immediately, but don't await
  const dataPromise = fetchData()
  
  return (
    <div>
      <div>Sidebar</div>
      <div>Header</div>
      <Suspense fallback={<Skeleton />}>
        <DataDisplay dataPromise={dataPromise} />
        <DataSummary dataPromise={dataPromise} />
      </Suspense>
      <div>Footer</div>
    </div>
  )
}

function DataDisplay({ dataPromise }: { dataPromise: Promise<Data> }) {
  const data = use(dataPromise) // Unwraps the promise
  return <div>{data.content}</div>
}

function DataSummary({ dataPromise }: { dataPromise: Promise<Data> }) {
  const data = use(dataPromise) // Reuses the same promise
  return <div>{data.summary}</div>
}
```

两个组件共享相同的承诺，因此只发生一次获取。当两个组件一起等待时，布局立即呈现。

**何时不使用此模式：**

- 布局决策所需的关键数据（影响定位）
- 首屏 SEO 关键内容
- 小而快速的查询，其中悬念开销不值得
- 当你想避免布局移位（加载→内容跳转）时

**权衡：**更快的初始绘制与潜在的布局转变。根据您的用户体验优先级进行选择。
