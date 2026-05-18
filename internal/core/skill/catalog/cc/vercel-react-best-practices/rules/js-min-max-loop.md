---
title: Use Loop for Min/Max Instead of Sort
impact: LOW
impactDescription: O(n) instead of O(n log n)
tags: javascript, arrays, performance, sorting, algorithms
---

## 使用循环求最小值/最大值而不是排序

查找最小或最大元素只需要遍历数组一次。排序既浪费又慢。

**不正确（O(n log n) - 排序以查找最新的）：**

```typescript
interface Project {
  id: string
  name: string
  updatedAt: number
}

function getLatestProject(projects: Project[]) {
  const sorted = [...projects].sort((a, b) => b.updatedAt - a.updatedAt)
  return sorted[0]
}
```

对整个数组进行排序只是为了找到最大值。

**不正确（O(n log n) - 按最旧和最新排序）：**

```typescript
function getOldestAndNewest(projects: Project[]) {
  const sorted = [...projects].sort((a, b) => a.updatedAt - b.updatedAt)
  return { oldest: sorted[0], newest: sorted[sorted.length - 1] }
}
```

当只需要最小/最大时，仍然进行不必要的排序。

**正确（O(n) - 单循环）：**

```typescript
function getLatestProject(projects: Project[]) {
  if (projects.length === 0) return null
  
  let latest = projects[0]
  
  for (let i = 1; i < projects.length; i++) {
    if (projects[i].updatedAt > latest.updatedAt) {
      latest = projects[i]
    }
  }
  
  return latest
}

function getOldestAndNewest(projects: Project[]) {
  if (projects.length === 0) return { oldest: null, newest: null }
  
  let oldest = projects[0]
  let newest = projects[0]
  
  for (let i = 1; i < projects.length; i++) {
    if (projects[i].updatedAt < oldest.updatedAt) oldest = projects[i]
    if (projects[i].updatedAt > newest.updatedAt) newest = projects[i]
  }
  
  return { oldest, newest }
}
```

单次遍历数组，不复制，不排序。

**替代方案（小数组的 Math.min/Math.max）：**

```typescript
const numbers = [5, 2, 8, 1, 9]
const min = Math.min(...numbers)
const max = Math.max(...numbers)
```

这适用于小型数组，但由于扩展运算符的限制，速度可能会较慢，或者对于非常大的数组会引发错误。最大数组长度在 Chrome 143 中约为 124000，在 Safari 18 中约为 638000；确切的数字可能有所不同 - 请参阅[the fiddle](https://jsfiddle.net/qw1jabsx/4/)。使用循环方法来提高可靠性。
