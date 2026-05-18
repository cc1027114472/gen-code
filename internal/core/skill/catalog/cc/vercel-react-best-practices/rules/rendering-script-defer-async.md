---
title: Use defer or async on Script Tags
impact: HIGH
impactDescription: eliminates render-blocking
tags: rendering, script, defer, async, performance
---

## 在脚本标签上使用 defer 或 async

**影响：高（消除渲染阻塞）**

脚本标签不带`defer`或者`async`在脚本下载和执行时阻止 HTML 解析。这会延迟首次内容绘制和交互时间。

- **`defer`**：并行下载，HTML解析完成后执行，保持执行顺序
- **`async`**：并行下载，准备好后立即执行，不保证顺序

使用`defer`对于依赖于 DOM 或其他脚本的脚本。使用`async`用于分析等独立脚本。

**不正确（阻止渲染）：**

```tsx
export default function Document() {
  return (
    <html>
      <head>
        <script src="https://example.com/analytics.js" />
        <script src="/scripts/utils.js" />
      </head>
      <body>{/* content */}</body>
    </html>
  )
}
```

**正确（非阻塞）：**

```tsx
export default function Document() {
  return (
    <html>
      <head>
        {/* Independent script - use async */}
        <script src="https://example.com/analytics.js" async />
        {/* DOM-dependent script - use defer */}
        <script src="/scripts/utils.js" defer />
      </head>
      <body>{/* content */}</body>
    </html>
  )
}
```

**注意：** 在 Next.js 中，更喜欢`next/script`组件与`strategy`prop 而不是原始脚本标签：

```tsx
import Script from 'next/script'

export default function Page() {
  return (
    <>
      <Script src="https://example.com/analytics.js" strategy="afterInteractive" />
      <Script src="/scripts/utils.js" strategy="beforeInteractive" />
    </>
  )
}
```

参考：[MDN - Script element](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/script#defer)
