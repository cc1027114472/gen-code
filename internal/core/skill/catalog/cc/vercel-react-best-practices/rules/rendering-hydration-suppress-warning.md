---
title: Suppress Expected Hydration Mismatches
impact: LOW-MEDIUM
impactDescription: avoids noisy hydration warnings for known differences
tags: rendering, hydration, ssr, nextjs
---

## 抑制预期的水合作用不匹配

在 SSR 框架（例如 Next.js）中，服务器和客户端上的某些值故意不同（随机 ID、日期、区域设置/时区格式）。对于这些*预期的*不匹配，请将动态文本包装在元素中`suppressHydrationWarning`以防止发出噪音警告。不要用它来隐藏真正的错误。不要过度使用它。

**不正确（已知的不匹配警告）：**

```tsx
function Timestamp() {
  return <span>{new Date().toLocaleString()}</span>
}
```

**正确（仅抑制预期的不匹配）：**

```tsx
function Timestamp() {
  return (
    <span suppressHydrationWarning>
      {new Date().toLocaleString()}
    </span>
  )
}
```
