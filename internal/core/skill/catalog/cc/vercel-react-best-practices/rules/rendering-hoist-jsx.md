---
title: Hoist Static JSX Elements
impact: LOW
impactDescription: avoids re-creation
tags: rendering, jsx, static, optimization
---

## 提升静态 JSX 元素

提取静态 JSX 外部组件以避免重新创建。

**不正确（每次渲染重新创建元素）：**

```tsx
function LoadingSkeleton() {
  return <div className="animate-pulse h-20 bg-gray-200" />
}

function Container() {
  return (
    <div>
      {loading && <LoadingSkeleton />}
    </div>
  )
}
```

**正确（重复使用相同的元素）：**

```tsx
const loadingSkeleton = (
  <div className="animate-pulse h-20 bg-gray-200" />
)

function Container() {
  return (
    <div>
      {loading && loadingSkeleton}
    </div>
  )
}
```

这对于大型静态 SVG 节点尤其有用，因为在每次渲染时重新创建这些节点的成本可能很高。

**注意：** 如果您的项目有[React Compiler](https://react.dev/learn/react-compiler)启用后，编译器会自动提升静态 JSX 元素并优化组件重新渲染，从而无需手动提升。
