---
title: Use useDeferredValue for Expensive Derived Renders
impact: MEDIUM
impactDescription: keeps input responsive during heavy computation
tags: rerender, useDeferredValue, optimization, concurrent
---

## 使用 useDeferredValue 进行昂贵的派生渲染

当用户输入触发昂贵的计算或渲染时，使用`useDeferredValue`保持输入响应。延迟值滞后，允许 React 优先考虑输入更新并在空闲时渲染昂贵的结果。

**不正确（过滤时输入感觉滞后）：**

```tsx
function Search({ items }: { items: Item[] }) {
  const [query, setQuery] = useState('')
  const filtered = items.filter(item => fuzzyMatch(item, query))

  return (
    <>
      <input value={query} onChange={e => setQuery(e.target.value)} />
      <ResultsList results={filtered} />
    </>
  )
}
```

**正确（输入保持快速，结果在准备好时呈现）：**

```tsx
function Search({ items }: { items: Item[] }) {
  const [query, setQuery] = useState('')
  const deferredQuery = useDeferredValue(query)
  const filtered = useMemo(
    () => items.filter(item => fuzzyMatch(item, deferredQuery)),
    [items, deferredQuery]
  )
  const isStale = query !== deferredQuery

  return (
    <>
      <input value={query} onChange={e => setQuery(e.target.value)} />
      <div style={{ opacity: isStale ? 0.7 : 1 }}>
        <ResultsList results={filtered} />
      </div>
    </>
  )
}
```

**何时使用：**

- 过滤/搜索大型列表
- 对输入做出反应的昂贵的可视化（图表、图形）
- 任何导致明显渲染延迟的派生状态

**注意：** 将昂贵的计算包含在`useMemo`将延迟值作为依赖项，否则它仍然在每个渲染上运行。

参考：[React useDeferredValue](https://react.dev/reference/react/useDeferredValue)
