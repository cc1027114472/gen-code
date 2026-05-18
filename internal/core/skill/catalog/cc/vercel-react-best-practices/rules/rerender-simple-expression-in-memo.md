---
title: Do not wrap a simple expression with a primitive result type in useMemo
impact: LOW-MEDIUM
impactDescription: wasted computation on every render
tags: rerender, useMemo, optimization
---

## 不要在 useMemo 中使用原始结果类型包装简单表达式

当表达式很简单（很少逻辑或算术运算符）并且具有原始结果类型（布尔值、数字、字符串）时，不要将其包装在`useMemo`.
呼唤`useMemo`并且比较钩子依赖关系可能比表达式本身消耗更多的资源。

**错误：**

```tsx
function Header({ user, notifications }: Props) {
  const isLoading = useMemo(() => {
    return user.isLoading || notifications.isLoading
  }, [user.isLoading, notifications.isLoading])

  if (isLoading) return <Skeleton />
  // return some markup
}
```

**正确的：**

```tsx
function Header({ user, notifications }: Props) {
  const isLoading = user.isLoading || notifications.isLoading

  if (isLoading) return <Skeleton />
  // return some markup
}
```
