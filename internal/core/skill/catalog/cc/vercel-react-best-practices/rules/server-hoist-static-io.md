---
title: Hoist Static I/O to Module Level
impact: HIGH
impactDescription: avoids repeated file/network I/O per request
tags: server, io, performance, next.js, route-handlers, og-image
---

## 将静态 I/O 提升到模块级别

**影响：高（避免每个请求重复的文件/网络 I/O）**

在路由处理程序或服务器函数中加载静态资源（字体、徽标、图像、配置文件）时，将 I/O 操作提升到模块级别。模块级代码在模块首次导入时运行一次，而不是在每个请求时运行。这消除了每次调用时都会运行的冗余文件系统读取或网络获取。

**不正确：根据每个请求读取字体文件**

```typescript
// app/api/og/route.tsx
import { ImageResponse } from 'next/og'

export async function GET(request: Request) {
  // Runs on EVERY request - expensive!
  const fontData = await fetch(
    new URL('./fonts/Inter.ttf', import.meta.url)
  ).then(res => res.arrayBuffer())
  
  const logoData = await fetch(
    new URL('./images/logo.png', import.meta.url)
  ).then(res => res.arrayBuffer())

  return new ImageResponse(
    <div style={{ fontFamily: 'Inter' }}>
      <img src={logoData} />
      Hello World
    </div>,
    { fonts: [{ name: 'Inter', data: fontData }] }
  )
}
```

**正确：在模块初始化时加载一次**

```typescript
// app/api/og/route.tsx
import { ImageResponse } from 'next/og'

// Module-level: runs ONCE when module is first imported
const fontData = fetch(
  new URL('./fonts/Inter.ttf', import.meta.url)
).then(res => res.arrayBuffer())

const logoData = fetch(
  new URL('./images/logo.png', import.meta.url)
).then(res => res.arrayBuffer())

export async function GET(request: Request) {
  // Await the already-started promises
  const [font, logo] = await Promise.all([fontData, logoData])

  return new ImageResponse(
    <div style={{ fontFamily: 'Inter' }}>
      <img src={logo} />
      Hello World
    </div>,
    { fonts: [{ name: 'Inter', data: font }] }
  )
}
```

**替代方案：使用 Node.js fs 同步文件读取**

```typescript
// app/api/og/route.tsx
import { ImageResponse } from 'next/og'
import { readFileSync } from 'fs'
import { join } from 'path'

// Synchronous read at module level - blocks only during module init
const fontData = readFileSync(
  join(process.cwd(), 'public/fonts/Inter.ttf')
)

const logoData = readFileSync(
  join(process.cwd(), 'public/images/logo.png')
)

export async function GET(request: Request) {
  return new ImageResponse(
    <div style={{ fontFamily: 'Inter' }}>
      <img src={logoData} />
      Hello World
    </div>,
    { fonts: [{ name: 'Inter', data: fontData }] }
  )
}
```

**一般 Node.js 示例：加载配置或模板**

```typescript
// Incorrect: reads config on every call
export async function processRequest(data: Data) {
  const config = JSON.parse(
    await fs.readFile('./config.json', 'utf-8')
  )
  const template = await fs.readFile('./template.html', 'utf-8')
  
  return render(template, data, config)
}

// Correct: loads once at module level
const configPromise = fs.readFile('./config.json', 'utf-8')
  .then(JSON.parse)
const templatePromise = fs.readFile('./template.html', 'utf-8')

export async function processRequest(data: Data) {
  const [config, template] = await Promise.all([
    configPromise,
    templatePromise
  ])
  
  return render(template, data, config)
}
```

**何时使用此模式：**

- 加载用于 OG 图像生成的字体
- 加载静态徽标、图标或水印
- 读取运行时不会更改的配置文件
- 加载电子邮件模板或其他静态模板
- 所有请求中都相同的任何静态资源

**何时不使用此模式：**

- 资产因请求或用户而异
- 在运行时可能会更改的文件（使用 TTL 缓存代替）
- 如果保持加载，大文件会消耗太多内存
- 不应保留在内存中的敏感数据

**结合 Vercel 的 [Fluid Compute](https://vercel.com/docs/fluid-compute)：** 模块级缓存尤其有效，因为多个并发请求共享同一个函数实例。静态资源会在请求之间保留在内存中，不会产生冷启动惩罚。

**在传统的无服务器中：** 每次冷启动都会重新执行模块级代码，但后续的热调用会重用已加载的资源，直到实例被回收为止。
