---
title: Parallel Nested Data Fetching
impact: CRITICAL
impactDescription: eliminates server-side waterfalls
tags: server, rsc, parallel-fetching, promise-chaining
---

## 并行嵌套数据获取

当并行获取嵌套数据时，链相关的获取会在每个项目的承诺内进行，因此缓慢的项目不会阻塞其余的项目。

**不正确（单个慢速项目阻止所有嵌套获取）：**

```tsx
const chats = await Promise.all(
  chatIds.map(id => getChat(id))
)

const chatAuthors = await Promise.all(
  chats.map(chat => getUser(chat.author))
)
```

如果有一个`getChat(id)`100 条聊天记录的速度非常慢，其他 99 条聊天记录的作者即使数据已准备好也无法开始加载。

**正确（每个项目链接其自己的嵌套获取）：**

```tsx
const chatAuthors = await Promise.all(
  chatIds.map(id => getChat(id).then(chat => getUser(chat.author)))
)
```

每个项目独立链接`getChat` → `getUser`，因此缓慢的聊天不会阻止作者对其他人的获取。
