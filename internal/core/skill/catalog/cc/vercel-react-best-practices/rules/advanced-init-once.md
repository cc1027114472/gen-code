---
title: Initialize App Once, Not Per Mount
impact: LOW-MEDIUM
impactDescription: avoids duplicate init in development
tags: initialization, useEffect, app-startup, side-effects
---

## 初始化应用程序一次，而不是每次安装

不要将每次应用程序加载必须运行一次的应用程序范围初始化放入其中`useEffect([])`一个组件的。组件可以重新安装并且效果将重新运行。请在入口模块中使用模块级防护或顶级 init。

**不正确（在开发中运行两次，重新安装时重新运行）：**

```tsx
function Comp() {
  useEffect(() => {
    loadFromStorage()
    checkAuthToken()
  }, [])

  // ...
}
```

**正确（每次应用程序加载一次）：**

```tsx
let didInit = false

function Comp() {
  useEffect(() => {
    if (didInit) return
    didInit = true
    loadFromStorage()
    checkAuthToken()
  }, [])

  // ...
}
```

参考：[Initializing the application](https://react.dev/learn/you-might-not-need-an-effect#initializing-the-application)
