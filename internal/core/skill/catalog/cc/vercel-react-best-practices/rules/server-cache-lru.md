---
title: Cross-Request LRU Caching
impact: HIGH
impactDescription: caches across requests
tags: server, cache, lru, cross-request
---

## 跨请求 LRU 缓存

`React.cache()`只能在一个请求内起作用。对于跨顺序请求共享的数据（用户单击按钮 A，然后单击按钮 B），请使用 LRU 缓存。

**执行：**

```typescript
import { LRUCache } from 'lru-cache'

const cache = new LRUCache<string, any>({
  max: 1000,
  ttl: 5 * 60 * 1000  // 5 minutes
})

export async function getUser(id: string) {
  const cached = cache.get(id)
  if (cached) return cached

  const user = await db.user.findUnique({ where: { id } })
  cache.set(id, user)
  return user
}

// Request 1: DB query, result cached
// Request 2: cache hit, no DB query
```

当连续的用户操作在几秒钟内到达需要相同数据的多个端点时使用。

**结合 Vercel 的 [Fluid Compute](https://vercel.com/docs/fluid-compute)：** LRU 缓存尤其有效，因为多个并发请求可以共享同一个函数实例和缓存。这意味着缓存可以跨请求持续存在，而不需要 Redis 之类的外部存储。

**在传统的无服务器中：** 每个调用都是独立运行的，因此请考虑使用 Redis 进行跨进程缓存。

参考：[https://github.com/isaacs/node-lru-cache](https://github.com/isaacs/node-lru-cache)
