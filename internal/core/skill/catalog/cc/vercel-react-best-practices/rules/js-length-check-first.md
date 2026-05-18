---
title: Early Length Check for Array Comparisons
impact: MEDIUM-HIGH
impactDescription: avoids expensive operations when lengths differ
tags: javascript, arrays, performance, optimization, comparison
---

## 数组比较的早期长度检查

将数组与昂贵的操作（排序、深度相等、序列化）进行比较时，首先检查长度。如果长度不同，则数组不能相等。

在实际应用程序中，当比较在热路径（事件处理程序、渲染循环）中运行时，这种优化尤其有价值。

**不正确（总是进行昂贵的比较）：**

```typescript
function hasChanges(current: string[], original: string[]) {
  // Always sorts and joins, even when lengths differ
  return current.sort().join() !== original.sort().join()
}
```

即使在以下情况下也会运行两个 O(n log n) 排序`current.length`是 5 并且`original.length`是 100。还有连接数组和比较字符串的开销。

**正确（首先检查 O(1) 长度）：**

```typescript
function hasChanges(current: string[], original: string[]) {
  // Early return if lengths differ
  if (current.length !== original.length) {
    return true
  }
  // Only sort when lengths match
  const currentSorted = current.toSorted()
  const originalSorted = original.toSorted()
  for (let i = 0; i < currentSorted.length; i++) {
    if (currentSorted[i] !== originalSorted[i]) {
      return true
    }
  }
  return false
}
```

这种新方法更加有效，因为：
- 它避免了长度不同时排序和连接数组的开销
- 它避免了连接字符串消耗内存（对于大型数组尤其重要）
- 它避免了改变原始数组
- 当发现差异时它会提前返回
