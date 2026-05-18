---
title: Split Combined Hook Computations
impact: MEDIUM
impactDescription: avoids recomputing independent steps
tags: rerender, useMemo, useEffect, dependencies, optimization
---

## 拆分组合 Hook 计算

当一个钩子包含多个具有不同依赖关系的独立任务时，将它们拆分为单独的钩子。当任何依赖项发生更改时，组合挂钩会重新运行所有任务，即使某些任务不使用更改后的值。

**不正确（改变`sortOrder`重新计算过滤）：**

```tsx
const sortedProducts = useMemo(() => {
  const filtered = products.filter((p) => p.category === category)
  const sorted = filtered.toSorted((a, b) =>
    sortOrder === "asc" ? a.price - b.price : b.price - a.price
  )
  return sorted
}, [products, category, sortOrder])
```

**正确（过滤仅在产品或类别更改时重新计算）：**

```tsx
const filteredProducts = useMemo(
  () => products.filter((p) => p.category === category),
  [products, category]
)

const sortedProducts = useMemo(
  () =>
    filteredProducts.toSorted((a, b) =>
      sortOrder === "asc" ? a.price - b.price : b.price - a.price
    ),
  [filteredProducts, sortOrder]
)
```

此模式也适用于`useEffect`当组合不相关的副作用时：

**不正确（当任一依赖项更改时，两种效果都会运行）：**

```tsx
useEffect(() => {
  analytics.trackPageView(pathname)
  document.title = `${pageTitle} | My App`
}, [pathname, pageTitle])
```

**正确（效果独立运行）：**

```tsx
useEffect(() => {
  analytics.trackPageView(pathname)
}, [pathname])

useEffect(() => {
  document.title = `${pageTitle} | My App`
}, [pageTitle])
```

**注意：** 如果您的项目有[React Compiler](https://react.dev/learn/react-compiler)启用后，它会自动优化依赖项跟踪，并可以为您处理其中一些情况。
