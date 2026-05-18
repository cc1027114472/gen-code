---
title: Animate SVG Wrapper Instead of SVG Element
impact: LOW
impactDescription: enables hardware acceleration
tags: rendering, svg, css, animation, performance
---

## 对 SVG 包装器进行动画处理而不是 SVG 元素

许多浏览器没有针对 SVG 元素上的 CSS3 动画的硬件加速。将 SVG 包裹在`<div>`并对包装器进行动画处理。

**不正确（直接制作 SVG 动画 - 无硬件加速）：**

```tsx
function LoadingSpinner() {
  return (
    <svg 
      className="animate-spin"
      width="24" 
      height="24" 
      viewBox="0 0 24 24"
    >
      <circle cx="12" cy="12" r="10" stroke="currentColor" />
    </svg>
  )
}
```

**正确（动画包装 div - 硬件加速）：**

```tsx
function LoadingSpinner() {
  return (
    <div className="animate-spin">
      <svg 
        width="24" 
        height="24" 
        viewBox="0 0 24 24"
      >
        <circle cx="12" cy="12" r="10" stroke="currentColor" />
      </svg>
    </div>
  )
}
```

这适用于所有 CSS 变换和过渡（`transform`, `opacity`, `translate`, `scale`, `rotate`）。包装器 div 允许浏览器使用 GPU 加速来实现更流畅的动画。
