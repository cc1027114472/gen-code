---
title: Use toSorted() Instead of sort() for Immutability
impact: MEDIUM-HIGH
impactDescription: prevents mutation bugs in React state
tags: javascript, arrays, immutability, react, state, mutation
---

## 使用 toSorted() 而不是 sort() 来实现不变性

`.sort()`就地改变数组，这可能会导致 React state 和 props 出现错误。使用`.toSorted()`创建一个没有突变的新排序数组。

**不正确（改变原始数组）：**

```typescript
function UserList({ users }: { users: User[] }) {
  // Mutates the users prop array!
  const sorted = useMemo(
    () => users.sort((a, b) => a.name.localeCompare(b.name)),
    [users]
  )
  return <div>{sorted.map(renderUser)}</div>
}
```

**正确（创建新数组）：**

```typescript
function UserList({ users }: { users: User[] }) {
  // Creates new sorted array, original unchanged
  const sorted = useMemo(
    () => users.toSorted((a, b) => a.name.localeCompare(b.name)),
    [users]
  )
  return <div>{sorted.map(renderUser)}</div>
}
```

**为什么这在 React 中很重要：**

1. Props/state 突变破坏了 React 的不变性模型 - React 希望 props 和 state 被视为只读
2. 导致过时的闭包错误 - 闭包内的数组变化（回调、效果）可能会导致意外行为

**浏览器支持（旧版浏览器的后备）：**

`.toSorted()`适用于所有现代浏览器（Chrome 110+、Safari 16+、Firefox 115+、Node.js 20+）。对于较旧的环境，请使用扩展运算符：

```typescript
// Fallback for older browsers
const sorted = [...items].sort((a, b) => a.value - b.value)
```

**其他不可变数组方法：**

- `.toSorted()`- 不可变排序
- `.toReversed()`- 不可变的反向
- `.toSpliced()`- 不可变的拼接
- `.with()`- 不可变元素替换
