---
title: Use flatMap to Map and Filter in One Pass
impact: LOW-MEDIUM
impactDescription: eliminates intermediate array
tags: javascript, arrays, flatMap, filter, performance
---

## 使用 flatMap 一次性进行映射和过滤

**影响：低-中（消除中间阵列）**

链接`.map().filter(Boolean)`创建一个中间数组并迭代两次。使用`.flatMap()`在一次传递中进行转换和过滤。

**不正确（2次迭代，中间数组）：**

```typescript
const userNames = users
  .map(user => user.isActive ? user.name : null)
  .filter(Boolean)
```

**正确（1次迭代，无中间数组）：**

```typescript
const userNames = users.flatMap(user =>
  user.isActive ? [user.name] : []
)
```

**更多示例：**

```typescript
// Extract valid emails from responses
// Before
const emails = responses
  .map(r => r.success ? r.data.email : null)
  .filter(Boolean)

// After
const emails = responses.flatMap(r =>
  r.success ? [r.data.email] : []
)

// Parse and filter valid numbers
// Before
const numbers = strings
  .map(s => parseInt(s, 10))
  .filter(n => !isNaN(n))

// After
const numbers = strings.flatMap(s => {
  const n = parseInt(s, 10)
  return isNaN(n) ? [] : [n]
})
```

**何时使用：**
- 变换项目，同时过滤掉一些项目
- 某些输入不产生输出的条件映射
- 解析/验证应跳过无效输入的位置
