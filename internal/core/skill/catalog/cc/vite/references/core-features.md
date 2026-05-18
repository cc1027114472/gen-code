---
name: vite-features
description: Vite 专属导入模式与运行时功能
---

# Vite 功能

## Glob 导入

导入匹配某个模式的多个模块：

```ts
const modules = import.meta.glob('./dir/*.ts')
// { './dir/foo.ts': () => import('./dir/foo.ts'), ... }

for (const path in modules) {
  modules[path]().then((mod) => {
    console.log(path, mod)
  })
}
```

### 预先加载

```ts
const modules = import.meta.glob('./dir/*.ts', { eager: true })
// Modules loaded immediately, no dynamic import
```

### 命名导入

```ts
const modules = import.meta.glob('./dir/*.ts', { import: 'setup' })
// Only imports the 'setup' export from each module

const defaults = import.meta.glob('./dir/*.ts', { import: 'default', eager: true })
```

### 多个模式

```ts
const modules = import.meta.glob(['./dir/*.ts', './another/*.ts'])
```

### 负向模式

```ts
const modules = import.meta.glob(['./dir/*.ts', '!**/ignored.ts'])
```

### 自定义查询

```ts
const svgRaw = import.meta.glob('./icons/*.svg', { query: '?raw', import: 'default' })
const svgUrls = import.meta.glob('./icons/*.svg', { query: '?url', import: 'default' })
```

## 资源导入查询

### URL 导入

```ts
import imgUrl from './img.png'
// Returns resolved URL: '/src/img.png' (dev) or '/assets/img.2d8efhg.png' (build)
```

### 显式 URL

```ts
import workletUrl from './worklet.js?url'
```

### 原始字符串

```ts
import shaderCode from './shader.glsl?raw'
```

### Inline/No-Inline 行为

```ts
import inlined from './small.png?inline'    // Force base64 inline
import notInlined from './large.png?no-inline'  // Force separate file
```

### Web Workers 支持

```ts
import Worker from './worker.ts?worker'
const worker = new Worker()

// Or inline:
import InlineWorker from './worker.ts?worker&inline'
```

推荐使用如下构造函数模式：

```ts
const worker = new Worker(new URL('./worker.ts', import.meta.url), {
  type: 'module',
})
```

## 环境变量

### 内建常量

```ts
import.meta.env.MODE      // 'development' | 'production' | custom
import.meta.env.BASE_URL  // Base URL from config
import.meta.env.PROD      // true in production
import.meta.env.DEV       // true in development
import.meta.env.SSR       // true when running in server
```

### 自定义变量

只有带 `VITE_` 前缀的变量会暴露给客户端：

```
# .env
VITE_API_URL=https://api.example.com
DB_PASSWORD=secret  # NOT exposed to client
```

```ts
console.log(import.meta.env.VITE_API_URL) // works
console.log(import.meta.env.DB_PASSWORD)  // undefined
```

### 按 mode 划分的文件

```
.env                # always loaded
.env.local          # always loaded, gitignored
.env.[mode]         # only in specified mode
.env.[mode].local   # only in specified mode, gitignored
```

### TypeScript 支持

```ts
// vite-env.d.ts
interface ImportMetaEnv {
  readonly VITE_API_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
```

### HTML 替换

```html
<p>Running in %MODE%</p>
<script>window.API = "%VITE_API_URL%"</script>
```

## CSS Modules 支持

任何 `.module.css` 文件都会被视为 CSS module：

```ts
import styles from './component.module.css'
element.className = styles.button
```

配合 camelCase 转换：

```ts
// .my-class -> myClass (if css.modules.localsConvention configured)
import { myClass } from './component.module.css'
```

## JSON 导入

```ts
import pkg from './package.json'
import { version } from './package.json'  // Named import with tree-shaking
```

## HMR API

```ts
if (import.meta.hot) {
  import.meta.hot.accept((newModule) => {
    // Handle update
  })
  
  import.meta.hot.dispose((data) => {
    // Cleanup before module is replaced
  })
  
  import.meta.hot.invalidate()  // Force full reload
}
```

<!--
来源参考：
- https://vite.dev/guide/features
- https://vite.dev/guide/env-and-mode
- https://vite.dev/guide/assets
- https://vite.dev/guide/api-hmr
-->
