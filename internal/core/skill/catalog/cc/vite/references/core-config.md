---
name: vite-config
description: 使用 `vite.config.ts` 的 Vite 配置模式
---

# Vite 配置

## 基础设置

```ts
// vite.config.ts
import { defineConfig } from 'vite'

export default defineConfig({
  // config options
})
```

Vite 会自动从项目根目录解析 `vite.config.ts`。无论 `package.json` 的 `type` 如何，都支持 ES modules 语法。

## 条件配置

导出函数以访问 command 和 mode：

```ts
export default defineConfig(({ command, mode, isSsrBuild, isPreview }) => {
  if (command === 'serve') {
    return { /* dev config */ }
  } else {
    return { /* build config */ }
  }
})
```

- `command`：开发时为 `'serve'`，生产构建时为 `'build'`
- `mode`：`'development'` 或 `'production'`（也可以通过 `--mode` 使用自定义值）

## 异步配置

```ts
export default defineConfig(async ({ command, mode }) => {
  const data = await fetchSomething()
  return { /* config */ }
})
```

## 在配置中使用环境变量

`.env` 文件会在**配置解析之后**加载。要在配置中访问它们，请使用 `loadEnv`：

```ts
import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ mode }) => {
  // Load env files from cwd, include all vars (empty prefix)
  const env = loadEnv(mode, process.cwd(), '')
  
  return {
    define: {
      __APP_ENV__: JSON.stringify(env.APP_ENV),
    },
    server: {
      port: env.APP_PORT ? Number(env.APP_PORT) : 5173,
    },
  }
})
```

## 关键配置项

### resolve.alias 配置

```ts
export default defineConfig({
  resolve: {
    alias: {
      '@': '/src',
      '~': '/src',
    },
  },
})
```

### define（全局常量）

```ts
export default defineConfig({
  define: {
    __APP_VERSION__: JSON.stringify('1.0.0'),
    __API_URL__: 'window.__backend_api_url',
  },
})
```

值必须可被 JSON 序列化，或者是单个标识符。非字符串值会自动包裹 `JSON.stringify`。

### plugins 配置

```ts
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
})
```

插件数组会被拍平；falsy 值会被忽略。

### server.proxy 配置

```ts
export default defineConfig({
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:3000',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
```

### build.target 配置

默认值：Baseline Widely Available 浏览器。你也可以自定义：

```ts
export default defineConfig({
  build: {
    target: 'esnext', // or 'es2020', ['chrome90', 'firefox88']
  },
})
```

## TypeScript 智能提示

对于纯 JS 配置文件：

```js
/** @type {import('vite').UserConfig} */
export default {
  // ...
}
```

或者使用 `satisfies`：

```ts
import type { UserConfig } from 'vite'

export default {
  // ...
} satisfies UserConfig
```

<!--
来源参考：
- https://vite.dev/config/
- https://vite.dev/guide/
-->
