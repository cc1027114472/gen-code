---
title: Per-Request Deduplication with React.cache()
impact: MEDIUM
impactDescription: deduplicates within request
tags: server, cache, react-cache, deduplication
---

## 使用 React.cache() 按请求重复数据删除

使用`React.cache()`用于服务器端请求重复数据删除。身份验证和数据库查询受益最多。

**用法：**

```typescript
import { cache } from 'react'

export const getCurrentUser = cache(async () => {
  const session = await auth()
  if (!session?.user?.id) return null
  return await db.user.findUnique({
    where: { id: session.user.id }
  })
})
```

在单个请求中，多次调用`getCurrentUser()`仅执行一次查询。

**避免内联对象作为参数：**

`React.cache()`使用浅层相等（`Object.is`) 来确定缓存命中。内联对象每次调用都会创建新的引用，从而防止缓存命中。

**不正确（总是缓存未命中）：**

```typescript
const getUser = cache(async (params: { uid: number }) => {
  return await db.user.findUnique({ where: { id: params.uid } })
})

// Each call creates new object, never hits cache
getUser({ uid: 1 })
getUser({ uid: 1 })  // Cache miss, runs query again
```

**正确（缓存命中）：**

```typescript
const getUser = cache(async (uid: number) => {
  return await db.user.findUnique({ where: { id: uid } })
})

// Primitive args use value equality
getUser(1)
getUser(1)  // Cache hit, returns cached result
```

如果必须传递对象，请传递相同的引用：

```typescript
const params = { uid: 1 }
getUser(params)  // Query runs
getUser(params)  // Cache hit (same reference)
```

**Next.js-具体说明：**

在 Next.js 中，`fetch`API 通过请求记忆自动扩展。具有相同 URL 和选项的请求会在单个请求中自动进行重复数据删除，因此您不需要`React.cache()`为了`fetch`来电。然而，`React.cache()`对于其他异步任务仍然是必不可少的：

- 数据库查询（Prisma、Drizzle 等）
- 繁重的计算
- 身份验证检查
- 文件系统操作
- 任何非获取异步工作

使用`React.cache()`在组件树中删除重复的这些操作。

参考：[React.cache documentation](https://react.dev/reference/react/cache)
