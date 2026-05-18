---
title: Avoid Duplicate Serialization in RSC Props
impact: LOW
impactDescription: reduces network payload by avoiding duplicate serialization
tags: server, rsc, serialization, props, client-components
---

## 避免 RSC Props 中的重复序列化

**影响：低（通过避免重复序列化减少网络负载）**

RSC→客户端序列化通过对象引用而不是值来消除重复。相同的引用=序列化一次；新参考=再次序列化。进行变换（`.toSorted()`, `.filter()`, `.map()`）在客户端，而不是服务器。

**不正确（重复数组）：**

```tsx
// RSC: sends 6 strings (2 arrays × 3 items)
<ClientList usernames={usernames} usernamesOrdered={usernames.toSorted()} />
```

**正确（发送 3 个字符串）：**

```tsx
// RSC: send once
<ClientList usernames={usernames} />

// Client: transform there
'use client'
const sorted = useMemo(() => [...usernames].sort(), [usernames])
```

**嵌套重复数据删除行为：**

重复数据删除以递归方式进行。影响因数据类型而异：

- `string[]`, `number[]`, `boolean[]`：**高影响** - 数组+所有基元完全重复
- `object[]`：**低影响** - 数组重复，但嵌套对象通过引用进行重复数据删除

```tsx
// string[] - duplicates everything
usernames={['a','b']} sorted={usernames.toSorted()} // sends 4 strings

// object[] - duplicates array structure only
users={[{id:1},{id:2}]} sorted={users.toSorted()} // sends 2 arrays + 2 unique objects (not 4)
```

**破坏重复数据删除的操作（创建新引用）：**

- 数组：`.toSorted()`, `.filter()`, `.map()`, `.slice()`, `[...arr]`
- 对象：`{...obj}`, `Object.assign()`, `structuredClone()`, `JSON.parse(JSON.stringify())`

**更多示例：**

```tsx
// ❌ Bad
<C users={users} active={users.filter(u => u.active)} />
<C product={product} productName={product.name} />

// ✅ Good
<C users={users} />
<C product={product} />
// Do filtering/destructuring in client
```

**例外：** 当转换成本高昂或客户端不需要原始数据时传递派生数据。
