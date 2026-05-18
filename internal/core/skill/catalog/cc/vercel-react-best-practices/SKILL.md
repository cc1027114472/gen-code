---
name: vercel-react-best-practices
description: React and Next.js performance optimization guidelines from Vercel Engineering. This skill should be used when writing, reviewing, or refactoring React/Next.js code to ensure optimal performance patterns. Triggers on tasks involving React components, Next.js pages, data fetching, bundle optimization, or performance improvements.
license: MIT
metadata:
  author: vercel
  version: "1.0.0"
---

# Vercel React 最佳实践

由 Vercel 维护的 React 和 Next.js 应用程序综合性能优化指南。包含 8 个类别的 65 条规则，按影响进行优先级排序，以指导自动重构和代码生成。

## 何时使用

在以下情况下请参考这些指南：
- 编写新的 React 组件或 Next.js 页面
- 实现数据获取（客户端或服务器端）
- 检查代码是否存在性能问题
- 重构现有的 React/Next.js 代码
- 优化包大小或加载时间

## 按优先级划分的规则类别

| 优先级 | 类别 | 影响 | 前缀 |
|----------|----------|--------|--------|
| 1 | 消除瀑布 | 关键 | `async-` |
| 2 | 包体积优化 | 关键 | `bundle-` |
| 3 | 服务器端性能 | 高 | `server-` |
| 4 | 客户端数据获取 | 中高 | `client-` |
| 5 | 重新渲染优化 | 中 | `rerender-` |
| 6 | 渲染性能 | 中 | `rendering-` |
| 7 | JavaScript 性能 | 低-中 | `js-` |
| 8 | 高级模式 | 低 | `advanced-` |

## 快速参考

### 1. 消除瀑布（关键）

- `async-defer-await`- 将等待移动到实际使用的分支中
- `async-parallel`- 使用 `Promise.all()` 处理独立操作
- `async-dependencies`- 使用 better-all 来实现部分依赖
- `async-api-routes`- 在 API 路由中尽早启动 Promise，并尽量延后 `await`
- `async-suspense-boundaries`- 使用 Suspense 传输内容

### 2. 捆绑包大小优化（关键）

- `bundle-barrel-imports`- 直接导入，避免 barrel 文件
- `bundle-dynamic-imports`- 对重型组件使用 next/dynamic
- `bundle-defer-third-party`- 在 hydration 后再加载分析/日志类第三方库
- `bundle-conditional`- 仅在功能激活时加载模块
- `bundle-preload`- 悬停/聚焦时预加载以感知速度

### 3. 服务器端性能（高）

- `server-auth-actions`- 验证 API 路由等服务器操作
- `server-cache-react`- 使用 React.cache() 进行每个请求的重复数据删除
- `server-cache-lru`- 使用 LRU 缓存实现跨请求缓存
- `server-dedup-props`- 避免 RSC 道具中的重复序列化
- `server-hoist-static-io`- 将静态 I/O（字体、徽标）提升到模块级别
- `server-serialization`- 最大限度地减少传递给客户端组件的数据
- `server-parallel-fetching`- 重组组件以并行化数据获取
- `server-parallel-nested-fetching`- Promise.all 中每个项目的链式嵌套获取
- `server-after-nonblocking`- 使用 after() 进行非阻塞操作

### 4. 客户端数据获取（中-高）

- `client-swr-dedup`- 使用 SWR 自动完成请求去重
- `client-event-listeners`- 删除重复的全局事件监听器
- `client-passive-event-listeners`- 使用被动监听器进行滚动
- `client-localstorage-schema`- 版本和最小化本地存储数据

### 5. 重新渲染优化（中）

- `rerender-defer-reads`- 不要订阅仅在回调中使用的状态
- `rerender-memo`- 将昂贵的工作提取到记忆组件中
- `rerender-memo-with-default-value`- 提升默认非原始道具
- `rerender-dependencies`- 在效果中使用原始依赖关系
- `rerender-derived-state`- 订阅派生布尔值，而不是原始值
- `rerender-derived-state-no-effect`- 在渲染期间导出状态，而不是效果
- `rerender-functional-setstate`- 使用函数式 setState 来实现稳定的回调
- `rerender-lazy-state-init`- 将函数传递给 useState 以获得昂贵的值
- `rerender-simple-expression-in-memo`- 避免简单原语的备忘录
- `rerender-split-combined-hooks`- 具有独立依赖关系的拆分钩子
- `rerender-move-effect-to-event`- 将交互逻辑放入事件处理程序中
- `rerender-transitions`- 使用 startTransition 进行非紧急更新
- `rerender-use-deferred-value`- 推迟昂贵的渲染以保持输入响应
- `rerender-use-ref-transient-values`- 使用 refs 作为瞬态频繁值
- `rerender-no-inline-components`- 不要在组件内定义组件

### 6.渲染性能（中）

- `rendering-animate-svg-wrapper`- 动画 div 包装器，而不是 SVG 元素
- `rendering-content-visibility`- 对长列表使用内容可见性
- `rendering-hoist-jsx`- 提取静态 JSX 外部组件
- `rendering-svg-precision`- 降低SVG坐标精度
- `rendering-hydration-no-flicker`- 对仅限客户端的数据使用内联脚本
- `rendering-hydration-suppress-warning`- 抑制预期的不匹配
- `rendering-activity`- 使用 Activity 组件进行显示/隐藏
- `rendering-conditional-render`- 使用三元，而不是 && 作为条件
- `rendering-usetransition-loading`- 更喜欢使用 useTransition 来加载状态
- `rendering-resource-hints`- 使用 React DOM 资源提示进行预加载
- `rendering-script-defer-async`- 在脚本标签上使用 defer 或 async

### 7. JavaScript 性能（低-中）

- `js-batch-dom-css`- 通过类或 cssText 对 CSS 更改进行分组
- `js-index-maps`- 构建地图以进行重复查找
- `js-cache-property-access`- 在循环中缓存对象属性
- `js-cache-function-results`- 模块级Map中的缓存函数结果
- `js-cache-storage`- 缓存 localStorage/sessionStorage 读取
- `js-combine-iterations`- 将多个过滤器/映射合并到一个循环中
- `js-length-check-first`- 在昂贵的比较之前检查数组长度
- `js-early-exit`- 尽早从函数中返回
- `js-hoist-regexp`- 在循环外提升正则表达式创建
- `js-min-max-loop`- 使用循环来获取最小/最大而不是排序
- `js-set-map-lookups`- 使用 Set/Map 进行 O(1) 查找
- `js-tosorted-immutable`- 使用 toSorted() 实现不变性
- `js-flatmap-filter`- 使用 flatMap 一次性进行映射和过滤
- `js-request-idle-callback`- 将非关键工作推迟到浏览器空闲时间

### 8. 高级模式（低）

- `advanced-event-handler-refs`- 将事件处理程序存储在 refs 中
- `advanced-init-once`- 每次应用程序加载时初始化应用程序一次
- `advanced-use-latest`- useLatest 用于稳定的回调引用

## 如何使用

阅读各个规则文件以获取详细说明和代码示例：

```
rules/async-parallel.md
rules/bundle-barrel-imports.md
```

每个规则文件包含：
- 简要解释为什么它很重要
- 错误的代码示例及解释
- 正确的代码示例及解释
- 其他上下文和参考资料

## 完整编译文件

有关扩展所有规则的完整指南：`AGENTS.md`
