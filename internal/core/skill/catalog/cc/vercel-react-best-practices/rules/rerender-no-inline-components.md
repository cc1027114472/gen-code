---
title: Don't Define Components Inside Components
impact: HIGH
impactDescription: prevents remount on every render
tags: rerender, components, remount, performance
---

## 不要在组件内定义组件

**影响：高（防止在每次渲染时重新安装）**

在另一个组件内定义一个组件会在每次渲染时创建一个新的组件类型。 React 每次都会看到不同的组件并完全重新安装它，从而销毁所有状态和 DOM。

开发人员这样做的一个常见原因是访问父变量而不传递 props。始终传递道具。

**不正确（每次渲染时重新安装）：**

```tsx
function UserProfile({ user, theme }) {
  // Defined inside to access `theme` - BAD
  const Avatar = () => (
    <img
      src={user.avatarUrl}
      className={theme === 'dark' ? 'avatar-dark' : 'avatar-light'}
    />
  )

  // Defined inside to access `user` - BAD
  const Stats = () => (
    <div>
      <span>{user.followers} followers</span>
      <span>{user.posts} posts</span>
    </div>
  )

  return (
    <div>
      <Avatar />
      <Stats />
    </div>
  )
}
```

每次`UserProfile`呈现，`Avatar`和`Stats`是新的组件类型。 React 卸载旧实例并安装新实例，丢失任何内部状态，再次运行效果，并重新创建 DOM 节点。

**正确（改为传递道具）：**

```tsx
function Avatar({ src, theme }: { src: string; theme: string }) {
  return (
    <img
      src={src}
      className={theme === 'dark' ? 'avatar-dark' : 'avatar-light'}
    />
  )
}

function Stats({ followers, posts }: { followers: number; posts: number }) {
  return (
    <div>
      <span>{followers} followers</span>
      <span>{posts} posts</span>
    </div>
  )
}

function UserProfile({ user, theme }) {
  return (
    <div>
      <Avatar src={user.avatarUrl} theme={theme} />
      <Stats followers={user.followers} posts={user.posts} />
    </div>
  )
}
```

**此错误的症状：**
- 输入字段在每次击键时失去焦点
- 动画意外重新启动
- `useEffect`清理/设置在每个父渲染上运行
- 滚动位置在组件内重置
