---
title: Narrow Effect Dependencies
impact: LOW
impactDescription: minimizes effect re-runs
tags: rerender, useEffect, dependencies, optimization
---

## 窄效应依赖性

指定原始依赖项而不是对象，以最大限度地减少重新运行的影响。

**不正确（在任何用户字段更改时重新运行）：**

```tsx
useEffect(() => {
  console.log(user.id)
}, [user])
```

**正确（仅当 id 更改时重新运行）：**

```tsx
useEffect(() => {
  console.log(user.id)
}, [user.id])
```

**对于派生状态，计算外部效应：**

```tsx
// Incorrect: runs on width=767, 766, 765...
useEffect(() => {
  if (width < 768) {
    enableMobileMode()
  }
}, [width])

// Correct: runs only on boolean transition
const isMobile = width < 768
useEffect(() => {
  if (isMobile) {
    enableMobileMode()
  }
}, [isMobile])
```
