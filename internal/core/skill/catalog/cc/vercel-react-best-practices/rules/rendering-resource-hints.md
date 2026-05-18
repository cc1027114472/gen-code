---
title: Use React DOM Resource Hints
impact: HIGH
impactDescription: reduces load time for critical resources
tags: rendering, preload, preconnect, prefetch, resource-hints
---

## 使用 React DOM 资源提示

**影响：高（减少关键资源的加载时间）**

React DOM 提供 API 来提示浏览器所需的资源。这些在服务器组件中特别有用，可以在客户端收到 HTML 之前开始加载资源。

- **`prefetchDNS(href)`**：解析您希望连接的域的 DNS
- **`preconnect(href)`**：建立到服务器的连接（DNS + TCP + TLS）
- **`preload(href, options)`**：获取您即将使用的资源（样式表、字体、脚本、图像）
- **`preloadModule(href)`**：获取您即将使用的 ES 模块
- **`preinit(href, options)`**：获取并评估样式表或脚本
- **`preinitModule(href)`**：获取并评估 ES 模块

**示例（预连接到第三方 API）：**

```tsx
import { preconnect, prefetchDNS } from 'react-dom'

export default function App() {
  prefetchDNS('https://analytics.example.com')
  preconnect('https://api.example.com')

  return <main>{/* content */}</main>
}
```

**示例（预加载关键字体和样式）：**

```tsx
import { preload, preinit } from 'react-dom'

export default function RootLayout({ children }) {
  // Preload font file
  preload('/fonts/inter.woff2', { as: 'font', type: 'font/woff2', crossOrigin: 'anonymous' })

  // Fetch and apply critical stylesheet immediately
  preinit('/styles/critical.css', { as: 'style' })

  return (
    <html>
      <body>{children}</body>
    </html>
  )
}
```

**示例（代码分割路由的预加载模块）：**

```tsx
import { preloadModule, preinitModule } from 'react-dom'

function Navigation() {
  const preloadDashboard = () => {
    preloadModule('/dashboard.js', { as: 'script' })
  }

  return (
    <nav>
      <a href="/dashboard" onMouseEnter={preloadDashboard}>
        Dashboard
      </a>
    </nav>
  )
}
```

**何时使用每个：**

|应用程序接口 |使用案例 |
|-----|----------|
| `prefetchDNS`|您稍后将连接到的第三方域 |
| `preconnect`|您将立即获取的 API 或 CDN |
| `preload`|当前页面所需的关键资源 |
| `preloadModule`| JS 模块可能用于下一个导航 |
| `preinit`|必须尽早执行的样式表/脚本 |
| `preinitModule`|必须提前执行的 ES 模块 |

参考：[React DOM Resource Preloading APIs](https://react.dev/reference/react-dom#resource-preloading-apis)
