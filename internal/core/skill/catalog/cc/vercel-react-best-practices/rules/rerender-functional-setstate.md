---
title: Use Functional setState Updates
impact: MEDIUM
impactDescription: prevents stale closures and unnecessary callback recreations
tags: react, hooks, useState, useCallback, callbacks, closures
---

## 使用函数式 setState 更新

当根据当前状态值更新状态时，使用setState的函数式更新形式，而不是直接引用状态变量。这可以防止过时的闭包，消除不必要的依赖关系，并创建稳定的回调引用。

**不正确（需要状态作为依赖）：**

```tsx
function TodoList() {
  const [items, setItems] = useState(initialItems)
  
  // Callback must depend on items, recreated on every items change
  const addItems = useCallback((newItems: Item[]) => {
    setItems([...items, ...newItems])
  }, [items])  // ❌ items dependency causes recreations
  
  // Risk of stale closure if dependency is forgotten
  const removeItem = useCallback((id: string) => {
    setItems(items.filter(item => item.id !== id))
  }, [])  // ❌ Missing items dependency - will use stale items!
  
  return <ItemsEditor items={items} onAdd={addItems} onRemove={removeItem} />
}
```

每次都会重新创建第一个回调`items`更改，这可能会导致子组件不必要地重新渲染。第二个回调有一个过时的关闭错误——它将始终引用初始的回调`items`价值。

**正确（稳定的回调，没有陈旧的闭包）：**

```tsx
function TodoList() {
  const [items, setItems] = useState(initialItems)
  
  // Stable callback, never recreated
  const addItems = useCallback((newItems: Item[]) => {
    setItems(curr => [...curr, ...newItems])
  }, [])  // ✅ No dependencies needed
  
  // Always uses latest state, no stale closure risk
  const removeItem = useCallback((id: string) => {
    setItems(curr => curr.filter(item => item.id !== id))
  }, [])  // ✅ Safe and stable
  
  return <ItemsEditor items={items} onAdd={addItems} onRemove={removeItem} />
}
```

**好处：**

1. **稳定的回调引用** - 状态更改时不需要重新创建回调
2. **没有陈旧的闭包** - 始终按最新的状态值运行
3. **更少的依赖项** - 简化依赖项数组并减少内存泄漏
4. **防止错误** - 消除 React 关闭错误的最常见来源

**何时使用功能更新：**

- 任何依赖于当前状态值的 setState
- 当需要状态时，在 useCallback/useMemo 内部
- 引用状态的事件处理程序
- 更新状态的异步操作

**当直接更新没问题时：**

- 将状态设置为静态值：`setCount(0)`
- 仅从 props/arguments 设置状态：`setName(newName)`
- 状态不依赖于先前的值

**注意：** 如果您的项目有[React Compiler](https://react.dev/learn/react-compiler)启用后，编译器可以自动优化某些情况，但仍建议进行功能更新以确保正确性并防止过时的关闭错误。
