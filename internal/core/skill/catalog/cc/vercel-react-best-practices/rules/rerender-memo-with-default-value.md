---

title: Extract Default Non-primitive Parameter Value from Memoized Component to Constant
impact: MEDIUM
impactDescription: restores memoization by using a constant for default value
tags: rerender, memo, optimization

---

## 将默认非原始参数值从记忆组件提取到常量

当记忆组件具有某些非原始可选参数（例如数组、函数或对象）的默认值时，调用没有该参数的组件会导致记忆损坏。这是因为每次重新渲染时都会创建新的值实例，并且它们不会通过严格的相等比较`memo()`.

要解决此问题，请将默认值提取到常量中。

**不正确（`onClick`每次重新渲染时都有不同的值）：**

```tsx
const UserAvatar = memo(function UserAvatar({ onClick = () => {} }: { onClick?: () => void }) {
  // ...
})

// Used without optional onClick
<UserAvatar />
```

**正确（稳定的默认值）：**

```tsx
const NOOP = () => {};

const UserAvatar = memo(function UserAvatar({ onClick = NOOP }: { onClick?: () => void }) {
  // ...
})

// Used without optional onClick
<UserAvatar />
```
