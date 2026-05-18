---
name: vite-environment-api
description: 面向多运行环境的 Vite 6+ Environment API
---

# Environment API（Vite 6+）说明

Environment API 将传统的 client/SSR 二分模式，扩展为可以正式表达多个运行环境的模型。

## 概念

Vite 6 之前：有两个隐式环境（`client` 和 `ssr`）。

Vite 6+：可以根据需要配置任意数量的环境（浏览器、Node 服务端、边缘服务端等）。

## 基础配置

对于 SPA/MPA，行为没有变化——这些选项会作用于隐式 `client` 环境：

```ts
export default defineConfig({
  build: { sourcemap: false },
  optimizeDeps: { include: ['lib'] },
})
```

## 多环境

```ts
export default defineConfig({
  build: { sourcemap: false },  // Inherited by all environments
  optimizeDeps: { include: ['lib'] },  // Client only
  environments: {
    // SSR environment
    server: {},
    // Edge runtime environment
    edge: {
      resolve: { noExternal: true },
    },
  },
})
```

环境会继承顶层配置。有些选项（例如 `optimizeDeps`）默认只作用于 `client`。

## 环境选项

```ts
interface EnvironmentOptions {
  define?: Record<string, any>
  resolve?: EnvironmentResolveOptions
  optimizeDeps: DepOptimizationOptions
  consumer?: 'client' | 'server'
  dev: DevOptions
  build: BuildOptions
}
```

## 自定义环境实例

运行时提供方可以定义自定义环境：

```ts
import { customEnvironment } from 'vite-environment-provider'

export default defineConfig({
  environments: {
    ssr: customEnvironment({
      build: { outDir: '/dist/ssr' },
    }),
  },
})
```

示例：Cloudflare 的 Vite 插件会在开发阶段于 `workerd` 运行时中执行代码。

## 向后兼容

- `server.moduleGraph` 返回混合的 client/SSR 视图
- `ssrLoadModule` 仍然可用
- 现有 SSR 应用可保持不变继续运行

## 何时使用

- **终端用户：** 通常无需自行配置——框架会处理
- **插件作者：** 用于实现感知环境的转换逻辑
- **框架作者：** 用于为其运行时需求创建自定义环境

## 插件中的环境访问

插件可以在 hooks 中访问环境信息：

```ts
{
  name: 'env-aware',
  transform(code, id, options) {
    if (options?.ssr) {
      // SSR-specific transform
    }
  },
}
```

<!--
来源参考：
- https://vite.dev/guide/api-environment
- https://vite.dev/blog/announcing-vite6
-->
