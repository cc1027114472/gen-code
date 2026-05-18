---
name: vite-plugin-api
description: 使用 Vite 专属 hooks 进行 Vite 插件开发
---

# Vite 插件 API

Vite 插件在 Rolldown 的插件接口之上，扩展了 Vite 专属的 hooks。

## 基础结构

```ts
function myPlugin(): Plugin {
  return {
    name: 'my-plugin',
    // hooks...
  }
}
```

## Vite 专属 Hooks

### config Hook

在配置解析前修改配置：

```ts
const plugin = () => ({
  name: 'add-alias',
  config: () => ({
    resolve: {
      alias: { foo: 'bar' },
    },
  }),
})
```

### configResolved Hook

访问最终解析后的配置：

```ts
const plugin = () => {
  let config: ResolvedConfig
  return {
    name: 'read-config',
    configResolved(resolvedConfig) {
      config = resolvedConfig
    },
    transform(code, id) {
      if (config.command === 'serve') { /* dev */ }
    },
  }
}
```

### configureServer Hook

为开发服务器添加自定义中间件：

```ts
const plugin = () => ({
  name: 'custom-middleware',
  configureServer(server) {
    server.middlewares.use((req, res, next) => {
      // handle request
      next()
    })
  },
})
```

返回函数以便在**内部中间件之后**执行：

```ts
configureServer(server) {
  return () => {
    server.middlewares.use((req, res, next) => {
      // runs after Vite's middlewares
    })
  }
}
```

### transformIndexHtml Hook

转换 HTML 入口文件：

```ts
const plugin = () => ({
  name: 'html-transform',
  transformIndexHtml(html) {
    return html.replace(/<title>(.*?)<\/title>/, '<title>New Title</title>')
  },
})
```

注入标签：

```ts
transformIndexHtml() {
  return [
    { tag: 'script', attrs: { src: '/inject.js' }, injectTo: 'body' },
  ]
}
```

### handleHotUpdate Hook

自定义 HMR 处理：

```ts
handleHotUpdate({ server, modules, timestamp }) {
  server.ws.send({ type: 'custom', event: 'special-update', data: {} })
  return [] // empty = skip default HMR
}
```

## 虚拟模块

在磁盘上不存在真实文件的情况下提供虚拟内容：

```ts
const plugin = () => {
  const virtualModuleId = 'virtual:my-module'
  const resolvedId = '\0' + virtualModuleId

  return {
    name: 'virtual-module',
    resolveId(id) {
      if (id === virtualModuleId) return resolvedId
    },
    load(id) {
      if (id === resolvedId) {
        return `export const msg = "from virtual module"`
      }
    },
  }
}
```

使用方式：

```ts
import { msg } from 'virtual:my-module'
```

约定：面向用户的路径以前缀 `virtual:` 表示，解析后的 id 以前缀 `\0` 表示。

## 插件顺序

使用 `enforce` 控制执行顺序：

```ts
{
  name: 'pre-plugin',
  enforce: 'pre',  // runs before core plugins
}

{
  name: 'post-plugin',
  enforce: 'post', // runs after build plugins
}
```

顺序为：Alias -> `enforce: 'pre'` -> Core -> User（无 enforce）-> Build -> `enforce: 'post'` -> Post-build

## 条件应用

```ts
{
  name: 'build-only',
  apply: 'build',  // or 'serve'
}

// Function form:
{
  apply(config, { command }) {
    return command === 'build' && !config.build.ssr
  }
}
```

## 通用 Hooks（来自 Rolldown）

这些 hooks 在开发和构建中都可用：

- `resolveId(id, importer)` - 解析导入路径
- `load(id)` - 加载模块内容
- `transform(code, id)` - 转换模块代码

```ts
transform(code, id) {
  if (id.endsWith('.custom')) {
    return { code: compile(code), map: null }
  }
}
```

## 客户端与服务端通信

服务端到客户端：

```ts
configureServer(server) {
  server.ws.send('my:event', { msg: 'hello' })
}
```

客户端侧：

```ts
if (import.meta.hot) {
  import.meta.hot.on('my:event', (data) => {
    console.log(data.msg)
  })
}
```

客户端到服务端：

```ts
// Client
import.meta.hot.send('my:from-client', { msg: 'Hey!' })

// Server
server.ws.on('my:from-client', (data, client) => {
  client.send('my:ack', { msg: 'Got it!' })
})
```

<!--
来源参考：
- https://vite.dev/guide/api-plugin
-->
