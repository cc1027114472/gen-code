# React 最佳实践

**版本 1.0.0**
Vercel Engineering
2026 年 1 月

> **笔记：**
> 本文件主要供代理与 LLM 在维护、
> 生成或重构 React 和 Next.js 代码库时遵循。
> 人类也可以参考，但这里的指导主要针对自动化
> 与 AI 辅助工作流的一致性进行优化。

---

## 摘要

这是面向 React 与 Next.js 应用的综合性能优化指南，专为 AI 代理与 LLM 设计。它包含 8 个类别、40 多条规则，按影响从关键项（消除瀑布、减小包体积）到增量项（高级模式）排序。每条规则都包含详细说明、对比错误与正确实现的真实示例，以及用于指导自动重构和代码生成的影响指标。

---

## 目录

1. [Eliminating Waterfalls](#1-eliminating-waterfalls) - **关键**
   - 1.1 [Defer Await Until Needed](#11-defer-await-until-needed)
   - 1.2 [Dependency-Based Parallelization](#12-dependency-based-parallelization)
   - 1.3 [Prevent Waterfall Chains in API Routes](#13-prevent-waterfall-chains-in-api-routes)
   - 1.4 [Promise.all() for Independent Operations](#14-promiseall-for-independent-operations)
   - 1.5 [Strategic Suspense Boundaries](#15-strategic-suspense-boundaries)
2. [Bundle Size Optimization](#2-bundle-size-optimization) - **关键**
   - 2.1 [Avoid Barrel File Imports](#21-avoid-barrel-file-imports)
   - 2.2 [Conditional Module Loading](#22-conditional-module-loading)
   - 2.3 [Defer Non-Critical Third-Party Libraries](#23-defer-non-critical-third-party-libraries)
   - 2.4 [Dynamic Imports for Heavy Components](#24-dynamic-imports-for-heavy-components)
   - 2.5 [Preload Based on User Intent](#25-preload-based-on-user-intent)
3. [Server-Side Performance](#3-server-side-performance) - **高**
   - 3.1 [Authenticate Server Actions Like API Routes](#31-authenticate-server-actions-like-api-routes)
   - 3.2 [Avoid Duplicate Serialization in RSC Props](#32-avoid-duplicate-serialization-in-rsc-props)
   - 3.3 [Cross-Request LRU Caching](#33-cross-request-lru-caching)
   - 3.4 [Hoist Static I/O to Module Level](#34-hoist-static-io-to-module-level)
   - 3.5 [Minimize Serialization at RSC Boundaries](#35-minimize-serialization-at-rsc-boundaries)
   - 3.6 [Parallel Data Fetching with Component Composition](#36-parallel-data-fetching-with-component-composition)
   - 3.7 [Parallel Nested Data Fetching](#37-parallel-nested-data-fetching)
   - 3.8 [Per-Request Deduplication with React.cache()](#38-per-request-deduplication-with-reactcache)
   - 3.9 [Use after() for Non-Blocking Operations](#39-use-after-for-non-blocking-operations)
4. [Client-Side Data Fetching](#4-client-side-data-fetching) - **中高**
   - 4.1 [Deduplicate Global Event Listeners](#41-deduplicate-global-event-listeners)
   - 4.2 [Use Passive Event Listeners for Scrolling Performance](#42-use-passive-event-listeners-for-scrolling-performance)
   - 4.3 [Use SWR for Automatic Deduplication](#43-use-swr-for-automatic-deduplication)
   - 4.4 [Version and Minimize localStorage Data](#44-version-and-minimize-localstorage-data)
5. [Re-render Optimization](#5-re-render-optimization) - **中**
   - 5.1 [Calculate Derived State During Rendering](#51-calculate-derived-state-during-rendering)
   - 5.2 [Defer State Reads to Usage Point](#52-defer-state-reads-to-usage-point)
   - 5.3 [Do not wrap a simple expression with a primitive result type in useMemo](#53-do-not-wrap-a-simple-expression-with-a-primitive-result-type-in-usememo)
   - 5.4 [Don't Define Components Inside Components](#54-dont-define-components-inside-components)
   - 5.5 [Extract Default Non-primitive Parameter Value from Memoized Component to Constant](#55-extract-default-non-primitive-parameter-value-from-memoized-component-to-constant)
   - 5.6 [Extract to Memoized Components](#56-extract-to-memoized-components)
   - 5.7 [Narrow Effect Dependencies](#57-narrow-effect-dependencies)
   - 5.8 [Put Interaction Logic in Event Handlers](#58-put-interaction-logic-in-event-handlers)
   - 5.9 [Split Combined Hook Computations](#59-split-combined-hook-computations)
   - 5.10 [Subscribe to Derived State](#510-subscribe-to-derived-state)
   - 5.11 [Use Functional setState Updates](#511-use-functional-setstate-updates)
   - 5.12 [Use Lazy State Initialization](#512-use-lazy-state-initialization)
   - 5.13 [Use Transitions for Non-Urgent Updates](#513-use-transitions-for-non-urgent-updates)
   - 5.14 [Use useDeferredValue for Expensive Derived Renders](#514-use-usedeferredvalue-for-expensive-derived-renders)
   - 5.15 [Use useRef for Transient Values](#515-use-useref-for-transient-values)
6. [Rendering Performance](#6-rendering-performance) - **中**
   - 6.1 [Animate SVG Wrapper Instead of SVG Element](#61-animate-svg-wrapper-instead-of-svg-element)
   - 6.2 [CSS content-visibility for Long Lists](#62-css-content-visibility-for-long-lists)
   - 6.3 [Hoist Static JSX Elements](#63-hoist-static-jsx-elements)
   - 6.4 [Optimize SVG Precision](#64-optimize-svg-precision)
   - 6.5 [Prevent Hydration Mismatch Without Flickering](#65-prevent-hydration-mismatch-without-flickering)
   - 6.6 [Suppress Expected Hydration Mismatches](#66-suppress-expected-hydration-mismatches)
   - 6.7 [Use Activity Component for Show/Hide](#67-use-activity-component-for-showhide)
   - 6.8 [Use defer or async on Script Tags](#68-use-defer-or-async-on-script-tags)
   - 6.9 [Use Explicit Conditional Rendering](#69-use-explicit-conditional-rendering)
   - 6.10 [Use React DOM Resource Hints](#610-use-react-dom-resource-hints)
   - 6.11 [Use useTransition Over Manual Loading States](#611-use-usetransition-over-manual-loading-states)
7. [JavaScript Performance](#7-javascript-performance) - **低-中**
   - 7.1 [Avoid Layout Thrashing](#71-avoid-layout-thrashing)
   - 7.2 [Build Index Maps for Repeated Lookups](#72-build-index-maps-for-repeated-lookups)
   - 7.3 [Cache Property Access in Loops](#73-cache-property-access-in-loops)
   - 7.4 [Cache Repeated Function Calls](#74-cache-repeated-function-calls)
   - 7.5 [Cache Storage API Calls](#75-cache-storage-api-calls)
   - 7.6 [Combine Multiple Array Iterations](#76-combine-multiple-array-iterations)
   - 7.7 [Defer Non-Critical Work with requestIdleCallback](#77-defer-non-critical-work-with-requestidlecallback)
   - 7.8 [Early Length Check for Array Comparisons](#78-early-length-check-for-array-comparisons)
   - 7.9 [Early Return from Functions](#79-early-return-from-functions)
   - 7.10 [Hoist RegExp Creation](#710-hoist-regexp-creation)
   - 7.11 [Use flatMap to Map and Filter in One Pass](#711-use-flatmap-to-map-and-filter-in-one-pass)
   - 7.12 [Use Loop for Min/Max Instead of Sort](#712-use-loop-for-minmax-instead-of-sort)
   - 7.13 [Use Set/Map for O(1) Lookups](#713-use-setmap-for-o1-lookups)
   - 7.14 [Use toSorted() Instead of sort() for Immutability](#714-use-tosorted-instead-of-sort-for-immutability)
8. [Advanced Patterns](#8-advanced-patterns)- **低的**
   - 8.1 [Initialize App Once, Not Per Mount](#81-initialize-app-once-not-per-mount)
   - 8.2 [Store Event Handlers in Refs](#82-store-event-handlers-in-refs)
   - 8.3 [useEffectEvent for Stable Callback Refs](#83-useeffectevent-for-stable-callback-refs)

---

## 1.消除瀑布

**影响：严重**

瀑布是第一大性能杀手。每个顺序等待都会增加完整的网络延迟。消除它们会产生最大的收益。

### 1.1 推迟等待直到需要

**影响：高（避免阻塞未使用的代码路径）**

移动`await`将操作放入实际使用它们的分支中，以避免阻塞不需要它们的代码路径。

**不正确：阻止两个分支**

```typescript
async function handleRequest(userId: string, skipProcessing: boolean) {
  const userData = await fetchUserData(userId)
  
  if (skipProcessing) {
    // Returns immediately but still waited for userData
    return { skipped: true }
  }
  
  // Only this branch uses userData
  return processUserData(userData)
}
```

**正确：仅在需要时阻止**

```typescript
async function handleRequest(userId: string, skipProcessing: boolean) {
  if (skipProcessing) {
    // Returns immediately without waiting
    return { skipped: true }
  }
  
  // Fetch only when needed
  const userData = await fetchUserData(userId)
  return processUserData(userData)
}
```

**另一个例子：早期回报优化**

```typescript
// Incorrect: always fetches permissions
async function updateResource(resourceId: string, userId: string) {
  const permissions = await fetchPermissions(userId)
  const resource = await getResource(resourceId)
  
  if (!resource) {
    return { error: 'Not found' }
  }
  
  if (!permissions.canEdit) {
    return { error: 'Forbidden' }
  }
  
  return await updateResourceData(resource, permissions)
}

// Correct: fetches only when needed
async function updateResource(resourceId: string, userId: string) {
  const resource = await getResource(resourceId)
  
  if (!resource) {
    return { error: 'Not found' }
  }
  
  const permissions = await fetchPermissions(userId)
  
  if (!permissions.canEdit) {
    return { error: 'Forbidden' }
  }
  
  return await updateResourceData(resource, permissions)
}
```

当经常采用跳过的分支或延迟操作成本高昂时，这种优化尤其有价值。

### 1.2 基于依赖的并行化

**影响：严重（2-10 倍改进）**

对于具有部分依赖关系的操作，请使用`better-all`最大化并行性。它会尽早自动启动每项任务。

**不正确：配置文件不必要地等待配置**

```typescript
const [user, config] = await Promise.all([
  fetchUser(),
  fetchConfig()
])
const profile = await fetchProfile(user.id)
```

**正确：配置和配置文件并行运行**

```typescript
import { all } from 'better-all'

const { user, config, profile } = await all({
  async user() { return fetchUser() },
  async config() { return fetchConfig() },
  async profile() {
    return fetchProfile((await this.$.user).id)
  }
})
```

**没有额外依赖的替代方案：**

```typescript
const userPromise = fetchUser()
const profilePromise = userPromise.then(user => fetchProfile(user.id))

const [user, config, profile] = await Promise.all([
  userPromise,
  fetchConfig(),
  profilePromise
])
```

我们也可以先创造所有的承诺，然后做`Promise.all()`在最后。

参考：[https://github.com/shuding/better-all](https://github.com/shuding/better-all)

### 1.3 防止API路由中出现瀑布链

**影响：严重（2-10 倍改进）**

在 API 路由和服务器操作中，立即启动独立操作，即使您还没有等待它们。

**不正确：配置等待身份验证，数据等待两者**

```typescript
export async function GET(request: Request) {
  const session = await auth()
  const config = await fetchConfig()
  const data = await fetchData(session.user.id)
  return Response.json({ data, config })
}
```

**正确：身份验证和配置立即启动**

```typescript
export async function GET(request: Request) {
  const sessionPromise = auth()
  const configPromise = fetchConfig()
  const session = await sessionPromise
  const [config, data] = await Promise.all([
    configPromise,
    fetchData(session.user.id)
  ])
  return Response.json({ data, config })
}
```

对于具有更复杂依赖链的操作，使用`better-all`自动最大化并行性（请参阅基于依赖的并行化）。

### 1.4 Promise.all() 用于独立操作

**影响：严重（2-10 倍改进）**

当异步操作没有相互依赖性时，使用以下方法同时执行它们`Promise.all()`.

**错误：顺序执行，3次往返**

```typescript
const user = await fetchUser()
const posts = await fetchPosts()
const comments = await fetchComments()
```

**正确：并行执行，1 次往返**

```typescript
const [user, posts, comments] = await Promise.all([
  fetchUser(),
  fetchPosts(),
  fetchComments()
])
```

### 1.5 战略悬念边界

**影响：高（初始绘制速度更快）**

不要在返回 JSX 之前等待异步组件中的数据，而是使用 Suspense 边界在数据加载时更快地显示包装器 UI。

**不正确：包装器被数据获取阻止**

```tsx
async function Page() {
  const data = await fetchData() // Blocks entire page
  
  return (
    <div>
      <div>Sidebar</div>
      <div>Header</div>
      <div>
        <DataDisplay data={data} />
      </div>
      <div>Footer</div>
    </div>
  )
}
```

即使只有中间部分需要数据，整个布局也会等待数据。

**正确：包装器立即显示，数据流在**

```tsx
function Page() {
  return (
    <div>
      <div>Sidebar</div>
      <div>Header</div>
      <div>
        <Suspense fallback={<Skeleton />}>
          <DataDisplay />
        </Suspense>
      </div>
      <div>Footer</div>
    </div>
  )
}

async function DataDisplay() {
  const data = await fetchData() // Only blocks this component
  return <div>{data.content}</div>
}
```

侧边栏、页眉和页脚立即呈现。只有DataDisplay等待数据。

**替代方案：跨组件共享承诺**

```tsx
function Page() {
  // Start fetch immediately, but don't await
  const dataPromise = fetchData()
  
  return (
    <div>
      <div>Sidebar</div>
      <div>Header</div>
      <Suspense fallback={<Skeleton />}>
        <DataDisplay dataPromise={dataPromise} />
        <DataSummary dataPromise={dataPromise} />
      </Suspense>
      <div>Footer</div>
    </div>
  )
}

function DataDisplay({ dataPromise }: { dataPromise: Promise<Data> }) {
  const data = use(dataPromise) // Unwraps the promise
  return <div>{data.content}</div>
}

function DataSummary({ dataPromise }: { dataPromise: Promise<Data> }) {
  const data = use(dataPromise) // Reuses the same promise
  return <div>{data.summary}</div>
}
```

两个组件共享相同的承诺，因此只发生一次获取。当两个组件一起等待时，布局立即呈现。

**何时不使用此模式：**

- 布局决策所需的关键数据（影响定位）

- 首屏 SEO 关键内容

- 小而快速的查询，其中悬念开销不值得

- 当你想避免布局移位（加载→内容跳转）时

**权衡：**更快的初始绘制与潜在的布局转变。根据您的用户体验优先级进行选择。

---

## 2. 捆绑包大小优化

**影响：严重**

减少初始包大小可以缩短交互和最大内容绘制的时间。

### 2.1 避免桶文件导入

**影响：严重（200-800 毫秒导入成本，构建缓慢）**

直接从源文件而不是桶文件导入，以避免加载数千个未使用的模块。 **桶文件**是重新导出多个模块的入口点（例如，`index.js`确实如此`export * from './module'`).

流行的图标和组件库在其条目文件中最多可以有 10,000 个重新导出**。对于许多 React 包来说，**仅导入它们就需要 200-800 毫秒**，影响开发速度和生产冷启动。

**为什么 tree-shaking 没有帮助：** 当库被标记为外部（未捆绑）时，捆绑器无法优化它。如果将其捆绑以启用树摇动，则分析整个模块图的构建速度会显着变慢。

**不正确：导入整个库**

```tsx
import { Check, X, Menu } from 'lucide-react'
// Loads 1,583 modules, takes ~2.8s extra in dev
// Runtime cost: 200-800ms on every cold start

import { Button, TextField } from '@mui/material'
// Loads 2,225 modules, takes ~4.2s extra in dev
```

**正确 - Next.js 13.5+（推荐）：**

```tsx
// Keep the standard imports - Next.js transforms them to direct imports
import { Check, X, Menu } from 'lucide-react'
// Full TypeScript support, no manual path wrangling
```

这是推荐的方法，因为它保留了 TypeScript 类型安全性和编辑器自动完成功能，同时仍然消除了桶导入成本。

**正确 - 直接导入（非 Next.js 项目）：**

```tsx
import Button from '@mui/material/Button'
import TextField from '@mui/material/TextField'
// Loads only what you use
```

> **TypeScript 警告：** 一些库（特别是`lucide-react`) 不发货`.d.ts`文件的深度导入路径。导入自`lucide-react/dist/esm/icons/check`解析为隐式`any`类型，导致错误`strict`或者`noImplicitAny`。更喜欢`optimizePackageImports`如果可用，或在使用直接导入之前验证其子路径的库导出类型。

这些优化使开发启动速度提高了 15-70%，构建速度提高了 28%，冷启动速度提高了 40%，并且 HMR 速度显着加快。

通常受影响的图书馆：`lucide-react`, `@mui/material`, `@mui/icons-material`, `@tabler/icons-react`, `react-icons`, `@headlessui/react`, `@radix-ui/react-*`, `lodash`, `ramda`, `date-fns`, `rxjs`, `react-use`.

参考：[https://vercel.com/blog/how-we-optimized-package-imports-in-next-js](https://vercel.com/blog/how-we-optimized-package-imports-in-next-js)

### 2.2 条件模块加载

**影响：高（仅在需要时加载大数据）**

仅在激活功能时加载大数据或模块。

**示例：延迟加载动画帧**

```tsx
function AnimationPlayer({ enabled, setEnabled }: { enabled: boolean; setEnabled: React.Dispatch<React.SetStateAction<boolean>> }) {
  const [frames, setFrames] = useState<Frame[] | null>(null)

  useEffect(() => {
    if (enabled && !frames && typeof window !== 'undefined') {
      import('./animation-frames.js')
        .then(mod => setFrames(mod.frames))
        .catch(() => setEnabled(false))
    }
  }, [enabled, frames, setEnabled])

  if (!frames) return <Skeleton />
  return <Canvas frames={frames} />
}
```

这`typeof window !== 'undefined'`检查可防止将此模块捆绑到 SSR，从而优化服务器捆绑包大小和构建速度。

### 2.3 推迟非关键第三方库

**影响：中（水合后负载）**

分析、日志记录和错误跟踪不会阻止用户交互。水合后加载它们。

**不正确：阻止初始捆绑**

```tsx
import { Analytics } from '@vercel/analytics/react'

export default function RootLayout({ children }) {
  return (
    <html>
      <body>
        {children}
        <Analytics />
      </body>
    </html>
  )
}
```

**正确：水合后负荷**

```tsx
import dynamic from 'next/dynamic'

const Analytics = dynamic(
  () => import('@vercel/analytics/react').then(m => m.Analytics),
  { ssr: false }
)

export default function RootLayout({ children }) {
  return (
    <html>
      <body>
        {children}
        <Analytics />
      </body>
    </html>
  )
}
```

### 2.4 重型部件的动态导入

**影响：严重（直接影响 TTI 和 LCP）**

使用`next/dynamic`延迟加载初始渲染时不需要的大型组件。

**不正确：Monaco 捆绑包的主块约为 300KB**

```tsx
import { MonacoEditor } from './monaco-editor'

function CodePanel({ code }: { code: string }) {
  return <MonacoEditor value={code} />
}
```

**正确：摩纳哥按需加载**

```tsx
import dynamic from 'next/dynamic'

const MonacoEditor = dynamic(
  () => import('./monaco-editor').then(m => m.MonacoEditor),
  { ssr: false }
)

function CodePanel({ code }: { code: string }) {
  return <MonacoEditor value={code} />
}
```

### 2.5 基于用户意图的预加载

**影响：中（减少感知延迟）**

在需要之前预加载大量捆绑包以减少感知延迟。

**示例：悬停/焦点时预加载**

```tsx
function EditorButton({ onClick }: { onClick: () => void }) {
  const preload = () => {
    if (typeof window !== 'undefined') {
      void import('./monaco-editor')
    }
  }

  return (
    <button
      onMouseEnter={preload}
      onFocus={preload}
      onClick={onClick}
    >
      Open Editor
    </button>
  )
}
```

**示例：启用功能标志时预加载**

```tsx
function FlagsProvider({ children, flags }: Props) {
  useEffect(() => {
    if (flags.editorEnabled && typeof window !== 'undefined') {
      void import('./monaco-editor').then(mod => mod.init())
    }
  }, [flags.editorEnabled])

  return <FlagsContext.Provider value={flags}>
    {children}
  </FlagsContext.Provider>
}
```

这`typeof window !== 'undefined'`检查可防止捆绑 SSR 的预加载模块，从而优化服务器捆绑包大小和构建速度。

---

## 3. 服务器端性能

**影响：高**

优化服务器端渲染和数据获取可以消除服务器端瀑布并缩短响应时间。

### 3.1 验证 API 路由等服务器操作

**影响：严重（防止未经授权的访问服务器突变）**

服务器操作（功能与`"use server"`）作为公共端点公开，就像 API 路由一样。始终在每个服务器操作**内部**验证身份验证和授权 - 不要仅仅依赖中间件、布局防护或页面级检查，因为服务器操作可以直接调用。

Next.js 文档明确指出：“以与面向公众的 API 端点相同的安全考虑来对待服务器操作，并验证是否允许用户执行变更。”

**不正确：没有身份验证检查**

```typescript
'use server'

export async function deleteUser(userId: string) {
  // Anyone can call this! No auth check
  await db.user.delete({ where: { id: userId } })
  return { success: true }
}
```

**正确：操作内的身份验证**

```typescript
'use server'

import { verifySession } from '@/lib/auth'
import { unauthorized } from '@/lib/errors'

export async function deleteUser(userId: string) {
  // Always check auth inside the action
  const session = await verifySession()
  
  if (!session) {
    throw unauthorized('Must be logged in')
  }
  
  // Check authorization too
  if (session.user.role !== 'admin' && session.user.id !== userId) {
    throw unauthorized('Cannot delete other users')
  }
  
  await db.user.delete({ where: { id: userId } })
  return { success: true }
}
```

**带输入验证：**

```typescript
'use server'

import { verifySession } from '@/lib/auth'
import { z } from 'zod'

const updateProfileSchema = z.object({
  userId: z.string().uuid(),
  name: z.string().min(1).max(100),
  email: z.string().email()
})

export async function updateProfile(data: unknown) {
  // Validate input first
  const validated = updateProfileSchema.parse(data)
  
  // Then authenticate
  const session = await verifySession()
  if (!session) {
    throw new Error('Unauthorized')
  }
  
  // Then authorize
  if (session.user.id !== validated.userId) {
    throw new Error('Can only update own profile')
  }
  
  // Finally perform the mutation
  await db.user.update({
    where: { id: validated.userId },
    data: {
      name: validated.name,
      email: validated.email
    }
  })
  
  return { success: true }
}
```

参考：[https://nextjs.org/docs/app/guides/authentication](https://nextjs.org/docs/app/guides/authentication)

### 3.2 避免 RSC Props 中的重复序列化

**影响：低（通过避免重复序列化减少网络负载）**

RSC→客户端序列化通过对象引用而不是值来消除重复。相同的引用=序列化一次；新参考=再次序列化。进行变换（`.toSorted()`, `.filter()`, `.map()`）在客户端，而不是服务器。

**不正确：重复数组**

```tsx
// RSC: sends 6 strings (2 arrays × 3 items)
<ClientList usernames={usernames} usernamesOrdered={usernames.toSorted()} />
```

**正确：发送 3 个字符串**

```tsx
// RSC: send once
<ClientList usernames={usernames} />

// Client: transform there
'use client'
const sorted = useMemo(() => [...usernames].sort(), [usernames])
```

**嵌套重复数据删除行为：**

```tsx
// string[] - duplicates everything
usernames={['a','b']} sorted={usernames.toSorted()} // sends 4 strings

// object[] - duplicates array structure only
users={[{id:1},{id:2}]} sorted={users.toSorted()} // sends 2 arrays + 2 unique objects (not 4)
```

重复数据删除以递归方式进行。影响因数据类型而异：

- `string[]`, `number[]`, `boolean[]`：**高影响** - 数组+所有基元完全重复

- `object[]`：**低影响** - 数组重复，但嵌套对象通过引用进行重复数据删除

**破坏重复数据删除的操作：创建新引用**

- 数组：`.toSorted()`, `.filter()`, `.map()`, `.slice()`, `[...arr]`

- 对象：`{...obj}`, `Object.assign()`, `structuredClone()`, `JSON.parse(JSON.stringify())`

**更多示例：**

```tsx
// ❌ Bad
<C users={users} active={users.filter(u => u.active)} />
<C product={product} productName={product.name} />

// ✅ Good
<C users={users} />
<C product={product} />
// Do filtering/destructuring in client
```

**例外：** 当转换成本高昂或客户端不需要原始数据时传递派生数据。

### 3.3 跨请求LRU缓存

**影响：高（跨请求缓存）**

`React.cache()`只能在一个请求内起作用。对于跨顺序请求共享的数据（用户单击按钮 A，然后单击按钮 B），请使用 LRU 缓存。

**执行：**

```typescript
import { LRUCache } from 'lru-cache'

const cache = new LRUCache<string, any>({
  max: 1000,
  ttl: 5 * 60 * 1000  // 5 minutes
})

export async function getUser(id: string) {
  const cached = cache.get(id)
  if (cached) return cached

  const user = await db.user.findUnique({ where: { id } })
  cache.set(id, user)
  return user
}

// Request 1: DB query, result cached
// Request 2: cache hit, no DB query
```

当连续的用户操作在几秒钟内到达需要相同数据的多个端点时使用。

**与维塞尔的[Fluid Compute](https://vercel.com/docs/fluid-compute):** LRU 缓存特别有效，因为多个并发请求可以共享相同的函数实例和缓存。这意味着缓存可以跨请求持续存在，而不需要像 Redis 这样的外部存储。

**在传统的无服务器中：** 每个调用都是独立运行的，因此请考虑使用 Redis 进行跨进程缓存。

参考：[https://github.com/isaacs/node-lru-cache](https://github.com/isaacs/node-lru-cache)

### 3.4 将静态 I/O 提升到模块级别

**影响：高（避免每个请求重复的文件/网络 I/O）**

在路由处理程序或服务器函数中加载静态资源（字体、徽标、图像、配置文件）时，将 I/O 操作提升到模块级别。模块级代码在模块首次导入时运行一次，而不是在每个请求时运行。这消除了每次调用时都会运行的冗余文件系统读取或网络获取。

**不正确：根据每个请求读取字体文件**

**正确：在模块初始化时加载一次**

**替代方案：使用 Node.js fs 同步文件读取**

**一般 Node.js 示例：加载配置或模板**

**何时使用此模式：**

- 加载用于 OG 图像生成的字体

- 加载静态徽标、图标或水印

- 读取运行时不会更改的配置文件

- 加载电子邮件模板或其他静态模板

- 所有请求中都相同的任何静态资源

**何时不使用此模式：**

- 资产因请求或用户而异

- 在运行时可能会更改的文件（使用 TTL 缓存代替）

- 如果保持加载，大文件会消耗太多内存

- 不应保留在内存中的敏感数据

**与维塞尔的[Fluid Compute](https://vercel.com/docs/fluid-compute):** 模块级缓存特别有效，因为多个并发请求共享同一个函数实例。静态资源在请求之间保持加载在内存中，没有冷启动惩罚。

**在传统的无服务器中：** 每次冷启动都会重新执行模块级代码，但后续的热调用会重用已加载的资源，直到实例被回收为止。

### 3.5 最小化 RSC 边界的序列化

**影响：高（减少数据传输大小）**

React 服务器/客户端边界将所有对象属性序列化为字符串，并将它们嵌入到 HTML 响应和后续 RSC 请求中。这些序列化数据直接影响页面重量和加载时间，因此**大小非常重要**。只传递客户端实际使用的字段。

**不正确：序列化所有 50 个字段**

```tsx
async function Page() {
  const user = await fetchUser()  // 50 fields
  return <Profile user={user} />
}

'use client'
function Profile({ user }: { user: User }) {
  return <div>{user.name}</div>  // uses 1 field
}
```

**正确：仅序列化 1 个字段**

```tsx
async function Page() {
  const user = await fetchUser()
  return <Profile name={user.name} />
}

'use client'
function Profile({ name }: { name: string }) {
  return <div>{name}</div>
}
```

### 3.6 组件组合的并行数据获取

**影响：严重（消除服务器端瀑布）**

React 服务器组件在树中顺序执行。通过组合重组以并行化数据获取。

**不正确：侧边栏等待页面的获取完成**

```tsx
export default async function Page() {
  const header = await fetchHeader()
  return (
    <div>
      <div>{header}</div>
      <Sidebar />
    </div>
  )
}

async function Sidebar() {
  const items = await fetchSidebarItems()
  return <nav>{items.map(renderItem)}</nav>
}
```

**正确：同时获取**

```tsx
async function Header() {
  const data = await fetchHeader()
  return <div>{data}</div>
}

async function Sidebar() {
  const items = await fetchSidebarItems()
  return <nav>{items.map(renderItem)}</nav>
}

export default function Page() {
  return (
    <div>
      <Header />
      <Sidebar />
    </div>
  )
}
```

**替代儿童道具：**

```tsx
async function Header() {
  const data = await fetchHeader()
  return <div>{data}</div>
}

async function Sidebar() {
  const items = await fetchSidebarItems()
  return <nav>{items.map(renderItem)}</nav>
}

function Layout({ children }: { children: ReactNode }) {
  return (
    <div>
      <Header />
      {children}
    </div>
  )
}

export default function Page() {
  return (
    <Layout>
      <Sidebar />
    </Layout>
  )
}
```

### 3.7 并行嵌套数据获取

**影响：严重（消除服务器端瀑布）**

当并行获取嵌套数据时，链相关的获取会在每个项目的承诺内进行，因此缓慢的项目不会阻塞其余的项目。

**不正确：单个慢速项目会阻止所有嵌套的获取**

```tsx
const chats = await Promise.all(
  chatIds.map(id => getChat(id))
)

const chatAuthors = await Promise.all(
  chats.map(chat => getUser(chat.author))
)
```

如果有一个`getChat(id)`100 条聊天记录的速度非常慢，其他 99 条聊天记录的作者即使数据已准备好也无法开始加载。

**正确：每个项目都链接自己的嵌套获取**

```tsx
const chatAuthors = await Promise.all(
  chatIds.map(id => getChat(id).then(chat => getUser(chat.author)))
)
```

每个项目独立链接`getChat` → `getUser`，因此缓慢的聊天不会阻止作者对其他人的获取。

### 3.8 使用 React.cache() 按请求重复数据删除

**影响：中（请求内重复数据删除）**

使用`React.cache()`用于服务器端请求重复数据删除。身份验证和数据库查询受益最多。

**用法：**

```typescript
import { cache } from 'react'

export const getCurrentUser = cache(async () => {
  const session = await auth()
  if (!session?.user?.id) return null
  return await db.user.findUnique({
    where: { id: session.user.id }
  })
})
```

在单个请求中，多次调用`getCurrentUser()`仅执行一次查询。

**避免内联对象作为参数：**

`React.cache()`使用浅层相等（`Object.is`) 来确定缓存命中。内联对象每次调用都会创建新的引用，从而防止缓存命中。

**不正确：总是缓存丢失**

```typescript
const getUser = cache(async (params: { uid: number }) => {
  return await db.user.findUnique({ where: { id: params.uid } })
})

// Each call creates new object, never hits cache
getUser({ uid: 1 })
getUser({ uid: 1 })  // Cache miss, runs query again
```

**正确：缓存命中**

```typescript
const params = { uid: 1 }
getUser(params)  // Query runs
getUser(params)  // Cache hit (same reference)
```

如果必须传递对象，请传递相同的引用：

**Next.js-具体说明：**

在 Next.js 中，`fetch`API 通过请求记忆自动扩展。具有相同 URL 和选项的请求会在单个请求中自动进行重复数据删除，因此您不需要`React.cache()`为了`fetch`来电。然而，`React.cache()`对于其他异步任务仍然是必不可少的：

- 数据库查询（Prisma、Drizzle 等）

- 繁重的计算

- 身份验证检查

- 文件系统操作

- 任何非获取异步工作

使用`React.cache()`在组件树中删除重复的这些操作。

参考：[https://react.dev/reference/react/cache](https://react.dev/reference/react/cache)

### 3.9 使用after()进行非阻塞操作

**影响：中（响应时间更快）**

使用 Next.js 的`after()`安排在发送响应后应执行的工作。这可以防止日志记录、分析和其他副作用阻止响应。

**不正确：阻止响应**

```tsx
import { logUserAction } from '@/app/utils'

export async function POST(request: Request) {
  // Perform mutation
  await updateDatabase(request)
  
  // Logging blocks the response
  const userAgent = request.headers.get('user-agent') || 'unknown'
  await logUserAction({ userAgent })
  
  return new Response(JSON.stringify({ status: 'success' }), {
    status: 200,
    headers: { 'Content-Type': 'application/json' }
  })
}
```

**正确：非阻塞**

```tsx
import { after } from 'next/server'
import { headers, cookies } from 'next/headers'
import { logUserAction } from '@/app/utils'

export async function POST(request: Request) {
  // Perform mutation
  await updateDatabase(request)
  
  // Log after response is sent
  after(async () => {
    const userAgent = (await headers()).get('user-agent') || 'unknown'
    const sessionCookie = (await cookies()).get('session-id')?.value || 'anonymous'
    
    logUserAction({ sessionCookie, userAgent })
  })
  
  return new Response(JSON.stringify({ status: 'success' }), {
    status: 200,
    headers: { 'Content-Type': 'application/json' }
  })
}
```

当日志记录在后台进行时，响应会立即发送。

**常见用例：**

- 分析跟踪

- 审计日志记录

- 发送通知

- 缓存失效

- 清理任务

**重要说明：**

- `after()`即使响应失败或重定向也会运行

- 适用于服务器操作、路由处理程序和服务器组件

参考：[https://nextjs.org/docs/app/api-reference/functions/after](https://nextjs.org/docs/app/api-reference/functions/after)

---

## 4. 客户端数据获取

**影响：中高**

自动重复数据删除和高效的数据获取模式减少了冗余的网络请求。

### 4.1 全局事件监听器去重

**影响：低（N 个组件的单个侦听器）**

使用`useSWRSubscription()`在组件实例之间共享全局事件侦听器。

**不正确：N 个实例 = N 个侦听器**

```tsx
function useKeyboardShortcut(key: string, callback: () => void) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.metaKey && e.key === key) {
        callback()
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [key, callback])
}
```

当使用`useKeyboardShortcut`hook 多次，每个实例都会注册一个新的监听器。

**正确：N 个实例 = 1 个侦听器**

```tsx
import useSWRSubscription from 'swr/subscription'

// Module-level Map to track callbacks per key
const keyCallbacks = new Map<string, Set<() => void>>()

function useKeyboardShortcut(key: string, callback: () => void) {
  // Register this callback in the Map
  useEffect(() => {
    if (!keyCallbacks.has(key)) {
      keyCallbacks.set(key, new Set())
    }
    keyCallbacks.get(key)!.add(callback)

    return () => {
      const set = keyCallbacks.get(key)
      if (set) {
        set.delete(callback)
        if (set.size === 0) {
          keyCallbacks.delete(key)
        }
      }
    }
  }, [key, callback])

  useSWRSubscription('global-keydown', () => {
    const handler = (e: KeyboardEvent) => {
      if (e.metaKey && keyCallbacks.has(e.key)) {
        keyCallbacks.get(e.key)!.forEach(cb => cb())
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  })
}

function Profile() {
  // Multiple shortcuts will share the same listener
  useKeyboardShortcut('p', () => { /* ... */ }) 
  useKeyboardShortcut('k', () => { /* ... */ })
  // ...
}
```

### 4.2 使用被动事件监听器来提高滚动性能

**影响：中（消除事件侦听器引起的滚动延迟）**

添加`{ passive: true }`触摸和滚轮事件侦听器以启用立即滚动。浏览器通常会等待侦听器完成以检查是否`preventDefault()`被调用，导致滚动延迟。

**错误：**

```typescript
useEffect(() => {
  const handleTouch = (e: TouchEvent) => console.log(e.touches[0].clientX)
  const handleWheel = (e: WheelEvent) => console.log(e.deltaY)
  
  document.addEventListener('touchstart', handleTouch)
  document.addEventListener('wheel', handleWheel)
  
  return () => {
    document.removeEventListener('touchstart', handleTouch)
    document.removeEventListener('wheel', handleWheel)
  }
}, [])
```

**正确的：**

```typescript
useEffect(() => {
  const handleTouch = (e: TouchEvent) => console.log(e.touches[0].clientX)
  const handleWheel = (e: WheelEvent) => console.log(e.deltaY)
  
  document.addEventListener('touchstart', handleTouch, { passive: true })
  document.addEventListener('wheel', handleWheel, { passive: true })
  
  return () => {
    document.removeEventListener('touchstart', handleTouch)
    document.removeEventListener('wheel', handleWheel)
  }
}, [])
```

**在以下情况下使用被动：**跟踪/分析、日志记录、任何不调用的侦听器`preventDefault()`.

**在以下情况下不要使用被动：**实现自定义滑动手势、自定义缩放控件或任何需要的侦听器`preventDefault()`.

### 4.3 使用SWR进行自动重复数据删除

**影响：中-高（自动重复数据删除）**

SWR 支持跨组件实例的请求重复数据删除、缓存和重新验证。

**错误：没有重复数据删除，每个实例都获取**

```tsx
function UserList() {
  const [users, setUsers] = useState([])
  useEffect(() => {
    fetch('/api/users')
      .then(r => r.json())
      .then(setUsers)
  }, [])
}
```

**正确：多个实例共享一个请求**

```tsx
import useSWR from 'swr'

function UserList() {
  const { data: users } = useSWR('/api/users', fetcher)
}
```

**对于不可变数据：**

```tsx
import { useImmutableSWR } from '@/lib/swr'

function StaticContent() {
  const { data } = useImmutableSWR('/api/config', fetcher)
}
```

**对于突变：**

```tsx
import { useSWRMutation } from 'swr/mutation'

function UpdateButton() {
  const { trigger } = useSWRMutation('/api/user', updateUser)
  return <button onClick={() => trigger()}>Update</button>
}
```

参考：[https://swr.vercel.app](https://swr.vercel.app)

### 4.4 版本和最小化本地存储数据

**影响：中（防止架构冲突，减少存储大小）**

向键添加版本前缀并仅存储需要的字段。防止架构冲突和敏感数据的意外存储。

**错误：**

```typescript
// No version, stores everything, no error handling
localStorage.setItem('userConfig', JSON.stringify(fullUserObject))
const data = localStorage.getItem('userConfig')
```

**正确的：**

```typescript
const VERSION = 'v2'

function saveConfig(config: { theme: string; language: string }) {
  try {
    localStorage.setItem(`userConfig:${VERSION}`, JSON.stringify(config))
  } catch {
    // Throws in incognito/private browsing, quota exceeded, or disabled
  }
}

function loadConfig() {
  try {
    const data = localStorage.getItem(`userConfig:${VERSION}`)
    return data ? JSON.parse(data) : null
  } catch {
    return null
  }
}

// Migration from v1 to v2
function migrate() {
  try {
    const v1 = localStorage.getItem('userConfig:v1')
    if (v1) {
      const old = JSON.parse(v1)
      saveConfig({ theme: old.darkMode ? 'dark' : 'light', language: old.lang })
      localStorage.removeItem('userConfig:v1')
    }
  } catch {}
}
```

**存储服务器响应的最少字段：**

```typescript
// User object has 20+ fields, only store what UI needs
function cachePrefs(user: FullUser) {
  try {
    localStorage.setItem('prefs:v1', JSON.stringify({
      theme: user.preferences.theme,
      notifications: user.preferences.notifications
    }))
  } catch {}
}
```

**始终包含在 try-catch 中：**`getItem()`和`setItem()`当超出配额或禁用时，引入隐身/私密浏览（Safari、Firefox）。

**优点：** 通过版本控制进行架构演变，减少存储大小，防止存储令牌/PII/内部标志。

---

## 5. 重新渲染优化

**影响：中**

减少不必要的重新渲染可以最大限度地减少计算浪费并提高 UI 响应能力。

### 5.1 计算渲染期间的导出状态

**影响：中等（避免冗余渲染和状态漂移）**

如果可以从当前的 props/state 计算出一个值，请勿将其存储在 state 中或在效果中更新它。在渲染期间导出它以避免额外的渲染和状态漂移。不要仅仅为了响应 prop 的变化而设置效果状态；更喜欢派生值或键控重置。

**错误：冗余状态和效果**

```tsx
function Form() {
  const [firstName, setFirstName] = useState('First')
  const [lastName, setLastName] = useState('Last')
  const [fullName, setFullName] = useState('')

  useEffect(() => {
    setFullName(firstName + ' ' + lastName)
  }, [firstName, lastName])

  return <p>{fullName}</p>
}
```

**正确：在渲染期间导出**

```tsx
function Form() {
  const [firstName, setFirstName] = useState('First')
  const [lastName, setLastName] = useState('Last')
  const fullName = firstName + ' ' + lastName

  return <p>{fullName}</p>
}
```

参考：[https://react.dev/learn/you-might-not-need-an-effect](https://react.dev/learn/you-might-not-need-an-effect)

### 5.2 将状态读取推迟到使用点

**影响：中（避免不必要的订阅）**

如果您只在回调中读取动态状态（searchParams、localStorage），请不要订阅它。

**不正确：订阅所有 searchParams 更改**

```tsx
function ShareButton({ chatId }: { chatId: string }) {
  const searchParams = useSearchParams()

  const handleShare = () => {
    const ref = searchParams.get('ref')
    shareChat(chatId, { ref })
  }

  return <button onClick={handleShare}>Share</button>
}
```

**正确：按需阅读，无需订阅**

```tsx
function ShareButton({ chatId }: { chatId: string }) {
  const handleShare = () => {
    const params = new URLSearchParams(window.location.search)
    const ref = params.get('ref')
    shareChat(chatId, { ref })
  }

  return <button onClick={handleShare}>Share</button>
}
```

### 5.3 不要在 useMemo 中使用原始结果类型包装简单表达式

**影响：低-中（每次渲染都浪费计算）**

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

### 5.4 不要在组件内定义组件

**影响：高（防止在每次渲染时重新安装）**

在另一个组件内定义一个组件会在每次渲染时创建一个新的组件类型。 React 每次都会看到不同的组件并完全重新安装它，从而销毁所有状态和 DOM。

开发人员这样做的一个常见原因是访问父变量而不传递 props。始终传递道具。

**不正确：在每次渲染时重新安装**

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

**正确：改为传递道具**

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

### 5.5 从记忆组件中提取默认非原始参数值到常量

**影响：中（通过使用常量作为默认值来恢复记忆）**

当记忆组件具有某些非原始可选参数（例如数组、函数或对象）的默认值时，调用没有该参数的组件会导致记忆损坏。这是因为每次重新渲染时都会创建新的值实例，并且它们不会通过严格的相等比较`memo()`.

要解决此问题，请将默认值提取到常量中。

**不正确：`onClick`每次渲染时都有不同的值**

```tsx
const UserAvatar = memo(function UserAvatar({ onClick = () => {} }: { onClick?: () => void }) {
  // ...
})

// Used without optional onClick
<UserAvatar />
```

**正确：稳定的默认值**

```tsx
const NOOP = () => {};

const UserAvatar = memo(function UserAvatar({ onClick = NOOP }: { onClick?: () => void }) {
  // ...
})

// Used without optional onClick
<UserAvatar />
```

### 5.6 提取到记忆组件

**影响：中（可提前返回）**

将昂贵的工作提取到记忆组件中，以便在计算之前尽早返回。

**不正确：即使在加载时也会计算头像**

```tsx
function Profile({ user, loading }: Props) {
  const avatar = useMemo(() => {
    const id = computeAvatarId(user)
    return <Avatar id={id} />
  }, [user])

  if (loading) return <Skeleton />
  return <div>{avatar}</div>
}
```

**正确：加载时跳过计算**

```tsx
const UserAvatar = memo(function UserAvatar({ user }: { user: User }) {
  const id = useMemo(() => computeAvatarId(user), [user])
  return <Avatar id={id} />
})

function Profile({ user, loading }: Props) {
  if (loading) return <Skeleton />
  return (
    <div>
      <UserAvatar user={user} />
    </div>
  )
}
```

**注意：** 如果您的项目有[React Compiler](https://react.dev/learn/react-compiler)启用，手动记忆`memo()`和`useMemo()`没有必要。编译器会自动优化重新渲染。

### 5.7 狭义效应依赖性

**影响：低（最小化影响重新运行）**

指定原始依赖项而不是对象，以最大限度地减少重新运行的影响。

**不正确：在任何用户字段更改时重新运行**

```tsx
useEffect(() => {
  console.log(user.id)
}, [user])
```

**正确：仅当 id 更改时重新运行**

```tsx
useEffect(() => {
  console.log(user.id)
}, [user.id])
```

**对于派生状态，计算外部效应：**

```tsx
// Incorrect: runs on width=767, 766, 765...
useEffect(() => {
  if (width < 768) {
    enableMobileMode()
  }
}, [width])

// Correct: runs only on boolean transition
const isMobile = width < 768
useEffect(() => {
  if (isMobile) {
    enableMobileMode()
  }
}, [isMobile])
```

### 5.8 将交互逻辑放入事件处理程序中

**影响：中（避免效果重新运行和重复的副作用）**

如果特定用户操作（提交、单击、拖动）触发副作用，请在该事件处理程序中运行它。不要将动作建模为状态+效果；它使效果在不相关的更改上重新运行，并且可以重复操作。

**不正确：事件建模为状态+效果**

```tsx
function Form() {
  const [submitted, setSubmitted] = useState(false)
  const theme = useContext(ThemeContext)

  useEffect(() => {
    if (submitted) {
      post('/api/register')
      showToast('Registered', theme)
    }
  }, [submitted, theme])

  return <button onClick={() => setSubmitted(true)}>Submit</button>
}
```

**正确：在处理程序中执行**

```tsx
function Form() {
  const theme = useContext(ThemeContext)

  function handleSubmit() {
    post('/api/register')
    showToast('Registered', theme)
  }

  return <button onClick={handleSubmit}>Submit</button>
}
```

参考：[https://react.dev/learn/removing-effect-dependencies#should-this-code-move-to-an-event-handler](https://react.dev/learn/removing-effect-dependencies#should-this-code-move-to-an-event-handler)

### 5.9 分割组合Hook计算

**影响：中（避免重新计算独立步骤）**

当一个钩子包含多个具有不同依赖关系的独立任务时，将它们拆分为单独的钩子。当任何依赖项发生更改时，组合挂钩会重新运行所有任务，即使某些任务不使用更改后的值。

**错误：改变`sortOrder`重新计算过滤**

```tsx
const sortedProducts = useMemo(() => {
  const filtered = products.filter((p) => p.category === category)
  const sorted = filtered.toSorted((a, b) =>
    sortOrder === "asc" ? a.price - b.price : b.price - a.price
  )
  return sorted
}, [products, category, sortOrder])
```

**正确：过滤仅在产品或类别更改时重新计算**

```tsx
const filteredProducts = useMemo(
  () => products.filter((p) => p.category === category),
  [products, category]
)

const sortedProducts = useMemo(
  () =>
    filteredProducts.toSorted((a, b) =>
      sortOrder === "asc" ? a.price - b.price : b.price - a.price
    ),
  [filteredProducts, sortOrder]
)
```

此模式也适用于`useEffect`当组合不相关的副作用时：

**不正确：当任一依赖项更改时，两种效果都会运行**

```tsx
useEffect(() => {
  analytics.trackPageView(pathname)
  document.title = `${pageTitle} | My App`
}, [pathname, pageTitle])
```

**正确：效果独立运行**

```tsx
useEffect(() => {
  analytics.trackPageView(pathname)
}, [pathname])

useEffect(() => {
  document.title = `${pageTitle} | My App`
}, [pageTitle])
```

**注意：** 如果您的项目有[React Compiler](https://react.dev/learn/react-compiler)启用后，它会自动优化依赖项跟踪，并可以为您处理其中一些情况。

### 5.10 订阅派生状态

**影响：中（降低重新渲染频率）**

订阅派生布尔状态而不是连续值以减少重新渲染频率。

**不正确：在每个像素更改时重新渲染**

```tsx
function Sidebar() {
  const width = useWindowWidth()  // updates continuously
  const isMobile = width < 768
  return <nav className={isMobile ? 'mobile' : 'desktop'} />
}
```

**正确：仅当布尔值更改时重新渲染**

```tsx
function Sidebar() {
  const isMobile = useMediaQuery('(max-width: 767px)')
  return <nav className={isMobile ? 'mobile' : 'desktop'} />
}
```

### 5.11 使用函数式 setState 更新

**影响：中（防止过时的关闭和不必要的回调重新创建）**

当根据当前状态值更新状态时，使用setState的函数式更新形式，而不是直接引用状态变量。这可以防止过时的闭包，消除不必要的依赖关系，并创建稳定的回调引用。

**不正确：需要状态作为依赖**

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

**正确：稳定的回调，没有陈旧的闭包**

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

### 5.12 使用惰性状态初始化

**影响：中（每次渲染都会浪费计算量）**

将函数传递给`useState`对于昂贵的初始值。如果没有函数形式，初始化程序将在每次渲染时运行，即使该值仅使用一次。

**不正确：在每个渲染上运行**

```tsx
function FilteredList({ items }: { items: Item[] }) {
  // buildSearchIndex() runs on EVERY render, even after initialization
  const [searchIndex, setSearchIndex] = useState(buildSearchIndex(items))
  const [query, setQuery] = useState('')
  
  // When query changes, buildSearchIndex runs again unnecessarily
  return <SearchResults index={searchIndex} query={query} />
}

function UserProfile() {
  // JSON.parse runs on every render
  const [settings, setSettings] = useState(
    JSON.parse(localStorage.getItem('settings') || '{}')
  )
  
  return <SettingsForm settings={settings} onChange={setSettings} />
}
```

**正确：仅运行一次**

```tsx
function FilteredList({ items }: { items: Item[] }) {
  // buildSearchIndex() runs ONLY on initial render
  const [searchIndex, setSearchIndex] = useState(() => buildSearchIndex(items))
  const [query, setQuery] = useState('')
  
  return <SearchResults index={searchIndex} query={query} />
}

function UserProfile() {
  // JSON.parse runs only on initial render
  const [settings, setSettings] = useState(() => {
    const stored = localStorage.getItem('settings')
    return stored ? JSON.parse(stored) : {}
  })
  
  return <SettingsForm settings={settings} onChange={setSettings} />
}
```

从 localStorage/sessionStorage 计算初始值、构建数据结构（索引、映射）、从 DOM 读取或执行大量转换时，请使用延迟初始化。

对于简单的原语 (`useState(0)`），直接引用（`useState(props.value)`)，或者廉价的文字 (`useState({})`)，函数形式是不必要的。

### 5.13 使用转换进行非紧急更新

**影响：中（保持 UI 响应能力）**

将频繁、非紧急的状态更新标记为转换以保持 UI 响应能力。

**不正确：在每次滚动时阻止 UI**

```tsx
function ScrollTracker() {
  const [scrollY, setScrollY] = useState(0)
  useEffect(() => {
    const handler = () => setScrollY(window.scrollY)
    window.addEventListener('scroll', handler, { passive: true })
    return () => window.removeEventListener('scroll', handler)
  }, [])
}
```

**正确：非阻塞更新**

```tsx
import { startTransition } from 'react'

function ScrollTracker() {
  const [scrollY, setScrollY] = useState(0)
  useEffect(() => {
    const handler = () => {
      startTransition(() => setScrollY(window.scrollY))
    }
    window.addEventListener('scroll', handler, { passive: true })
    return () => window.removeEventListener('scroll', handler)
  }, [])
}
```

### 5.14 使用 useDeferredValue 进行昂贵的派生渲染

**影响：中等（在繁重的计算过程中保持输入响应）**

当用户输入触发昂贵的计算或渲染时，使用`useDeferredValue`保持输入响应。延迟值滞后，允许 React 优先考虑输入更新并在空闲时渲染昂贵的结果。

**不正确：过滤时输入感觉滞后**

```tsx
function Search({ items }: { items: Item[] }) {
  const [query, setQuery] = useState('')
  const filtered = items.filter(item => fuzzyMatch(item, query))

  return (
    <>
      <input value={query} onChange={e => setQuery(e.target.value)} />
      <ResultsList results={filtered} />
    </>
  )
}
```

**正确：输入保持敏捷，结果在准备好时呈现**

```tsx
function Search({ items }: { items: Item[] }) {
  const [query, setQuery] = useState('')
  const deferredQuery = useDeferredValue(query)
  const filtered = useMemo(
    () => items.filter(item => fuzzyMatch(item, deferredQuery)),
    [items, deferredQuery]
  )
  const isStale = query !== deferredQuery

  return (
    <>
      <input value={query} onChange={e => setQuery(e.target.value)} />
      <div style={{ opacity: isStale ? 0.7 : 1 }}>
        <ResultsList results={filtered} />
      </div>
    </>
  )
}
```

**何时使用：**

- 过滤/搜索大型列表

- 对输入做出反应的昂贵的可视化（图表、图形）

- 任何导致明显渲染延迟的派生状态

**注意：** 将昂贵的计算包含在`useMemo`将延迟值作为依赖项，否则它仍然在每个渲染上运行。

参考：[https://react.dev/reference/react/useDeferredValue](https://react.dev/reference/react/useDeferredValue)

### 5.15 使用 useRef 作为瞬态值

**影响：中（避免频繁更新时不必要的重新渲染）**

当值频繁更改并且您不希望每次更新时都重新渲染（例如，鼠标跟踪器、间隔、瞬态标志）时，请将其存储在`useRef`而不是`useState`。保持 UI 组件状态；使用 refs 作为临时 DOM 相邻值。更新引用不会触发重新渲染。

**不正确：渲染每次更新**

```tsx
function Tracker() {
  const [lastX, setLastX] = useState(0)

  useEffect(() => {
    const onMove = (e: MouseEvent) => setLastX(e.clientX)
    window.addEventListener('mousemove', onMove)
    return () => window.removeEventListener('mousemove', onMove)
  }, [])

  return (
    <div
      style={{
        position: 'fixed',
        top: 0,
        left: lastX,
        width: 8,
        height: 8,
        background: 'black',
      }}
    />
  )
}
```

**正确：无需重新渲染跟踪**

```tsx
function Tracker() {
  const lastXRef = useRef(0)
  const dotRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const onMove = (e: MouseEvent) => {
      lastXRef.current = e.clientX
      const node = dotRef.current
      if (node) {
        node.style.transform = `translateX(${e.clientX}px)`
      }
    }
    window.addEventListener('mousemove', onMove)
    return () => window.removeEventListener('mousemove', onMove)
  }, [])

  return (
    <div
      ref={dotRef}
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        width: 8,
        height: 8,
        background: 'black',
        transform: 'translateX(0px)',
      }}
    />
  )
}
```

---

## 6. 渲染性能

**影响：中**

优化渲染过程减少了浏览器需要做的工作。

### 6.1 动画 SVG 包装而不是 SVG 元素

**影响：低（启用硬件加速）**

许多浏览器没有针对 SVG 元素上的 CSS3 动画的硬件加速。将 SVG 包裹在`<div>`并对包装器进行动画处理。

**不正确：直接为 SVG 制作动画 - 无硬件加速**

```tsx
function LoadingSpinner() {
  return (
    <svg 
      className="animate-spin"
      width="24" 
      height="24" 
      viewBox="0 0 24 24"
    >
      <circle cx="12" cy="12" r="10" stroke="currentColor" />
    </svg>
  )
}
```

**正确：动画包装 div - 硬件加速**

```tsx
function LoadingSpinner() {
  return (
    <div className="animate-spin">
      <svg 
        width="24" 
        height="24" 
        viewBox="0 0 24 24"
      >
        <circle cx="12" cy="12" r="10" stroke="currentColor" />
      </svg>
    </div>
  )
}
```

这适用于所有 CSS 变换和过渡（`transform`, `opacity`, `translate`, `scale`, `rotate`）。包装器 div 允许浏览器使用 GPU 加速来实现更流畅的动画。

### 6.2 长列表的 CSS 内容可见性

**影响：高（初始渲染速度更快）**

申请`content-visibility: auto`推迟离屏渲染。

**CSS：**

```css
.message-item {
  content-visibility: auto;
  contain-intrinsic-size: 0 80px;
}
```

**例子：**

```tsx
function MessageList({ messages }: { messages: Message[] }) {
  return (
    <div className="overflow-y-auto h-screen">
      {messages.map(msg => (
        <div key={msg.id} className="message-item">
          <Avatar user={msg.author} />
          <div>{msg.content}</div>
        </div>
      ))}
    </div>
  )
}
```

对于 1000 条消息，浏览器会跳过约 990 个屏幕外项目的布局/绘制（初始渲染速度提高 10 倍）。

### 6.3 提升静态 JSX 元素

**影响：低（避免重新创建）**

提取静态 JSX 外部组件以避免重新创建。

**不正确：每次渲染都重新创建元素**

```tsx
function LoadingSkeleton() {
  return <div className="animate-pulse h-20 bg-gray-200" />
}

function Container() {
  return (
    <div>
      {loading && <LoadingSkeleton />}
    </div>
  )
}
```

**正确：重复使用相同的元素**

```tsx
const loadingSkeleton = (
  <div className="animate-pulse h-20 bg-gray-200" />
)

function Container() {
  return (
    <div>
      {loading && loadingSkeleton}
    </div>
  )
}
```

这对于大型静态 SVG 节点尤其有用，因为在每次渲染时重新创建这些节点的成本可能很高。

**注意：** 如果您的项目有[React Compiler](https://react.dev/learn/react-compiler)启用后，编译器会自动提升静态 JSX 元素并优化组件重新渲染，从而无需手动提升。

### 6.4 优化 SVG 精度

**影响：低（减少文件大小）**

降低 SVG 坐标精度以减小文件大小。最佳精度取决于 viewBox 大小，但通常应考虑降低精度。

**错误：精度过高**

```svg
<path d="M 10.293847 20.847362 L 30.938472 40.192837" />
```

**正确：小数点后 1 位**

```svg
<path d="M 10.3 20.8 L 30.9 40.2" />
```

**使用 SVGO 实现自动化：**

```bash
npx svgo --precision=1 --multipass icon.svg
```

### 6.5 防止水合不匹配而不闪烁

**影响：中（避免视觉闪烁和水合错误）**

当渲染依赖于客户端存储（localStorage、cookie）的内容时，通过注入在 React 水合之前更新 DOM 的同步脚本来避免 SSR 损坏和水合后闪烁。

**不正确：破坏 SSR**

```tsx
function ThemeWrapper({ children }: { children: ReactNode }) {
  // localStorage is not available on server - throws error
  const theme = localStorage.getItem('theme') || 'light'
  
  return (
    <div className={theme}>
      {children}
    </div>
  )
}
```

服务端渲染会失败，因为`localStorage`未定义。

**错误：视觉闪烁**

```tsx
function ThemeWrapper({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState('light')
  
  useEffect(() => {
    // Runs after hydration - causes visible flash
    const stored = localStorage.getItem('theme')
    if (stored) {
      setTheme(stored)
    }
  }, [])
  
  return (
    <div className={theme}>
      {children}
    </div>
  )
}
```

组件首先使用默认值（`light`），然后在水合后更新，导致错误内容的可见闪烁。

**正确：无闪烁，无水合不匹配**

```tsx
function ThemeWrapper({ children }: { children: ReactNode }) {
  return (
    <>
      <div id="theme-wrapper">
        {children}
      </div>
      <script
        dangerouslySetInnerHTML={{
          __html: `
            (function() {
              try {
                var theme = localStorage.getItem('theme') || 'light';
                var el = document.getElementById('theme-wrapper');
                if (el) el.className = theme;
              } catch (e) {}
            })();
          `,
        }}
      />
    </>
  )
}
```

内联脚本在显示元素之前同步执行，确保 DOM 已经具有正确的值。无闪烁，无水合作用不匹配。

此模式对于主题切换、用户首选项、身份验证状态以及任何应立即呈现而不闪烁默认值的仅限客户端的数据特别有用。

### 6.6 抑制预期的水合不匹配

**影响：低-中（避免因已知差异而发出吵闹的水合警告）**

在 SSR 框架（例如 Next.js）中，服务器和客户端上的某些值故意不同（随机 ID、日期、区域设置/时区格式）。对于这些*预期的*不匹配，请将动态文本包装在元素中`suppressHydrationWarning`以防止发出噪音警告。不要用它来隐藏真正的错误。不要过度使用它。

**不正确：已知的不匹配警告**

```tsx
function Timestamp() {
  return <span>{new Date().toLocaleString()}</span>
}
```

**正确：仅抑制预期的不匹配**

```tsx
function Timestamp() {
  return (
    <span suppressHydrationWarning>
      {new Date().toLocaleString()}
    </span>
  )
}
```

### 6.7 使用Activity组件进行显示/隐藏

**影响：中（保留状态/DOM）**

使用 React 的`<Activity>`为频繁切换可见性的昂贵组件保留状态/DOM。

**用法：**

```tsx
import { Activity } from 'react'

function Dropdown({ isOpen }: Props) {
  return (
    <Activity mode={isOpen ? 'visible' : 'hidden'}>
      <ExpensiveMenu />
    </Activity>
  )
}
```

避免昂贵的重新渲染和状态损失。

### 6.8 在脚本标签上使用 defer 或 async

**影响：高（消除渲染阻塞）**

脚本标签不带`defer`或者`async`在脚本下载和执行时阻止 HTML 解析。这会延迟首次内容绘制和交互时间。

- **`defer`**：并行下载，HTML解析完成后执行，保持执行顺序

- **`async`**：并行下载，准备好后立即执行，不保证顺序

使用`defer`对于依赖于 DOM 或其他脚本的脚本。使用`async`用于分析等独立脚本。

**不正确：阻止渲染**

```tsx
export default function Document() {
  return (
    <html>
      <head>
        <script src="https://example.com/analytics.js" />
        <script src="/scripts/utils.js" />
      </head>
      <body>{/* content */}</body>
    </html>
  )
}
```

**正确：非阻塞**

```tsx
import Script from 'next/script'

export default function Page() {
  return (
    <>
      <Script src="https://example.com/analytics.js" strategy="afterInteractive" />
      <Script src="/scripts/utils.js" strategy="beforeInteractive" />
    </>
  )
}
```

**注意：** 在 Next.js 中，更喜欢`next/script`组件与`strategy`prop 而不是原始脚本标签：

参考：[https://developer.mozilla.org/en-US/docs/Web/HTML/Element/script#defer](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/script#defer)

### 6.9 使用显式条件渲染

**影响：低（防止渲染 0 或 NaN）**

使用显式三元运算符 (`? :`）而不是`&&`用于条件渲染，当条件可以是`0`, `NaN`，或其他呈现的虚假值。

**不正确：当计数为 0 时呈现“0”**

```tsx
function Badge({ count }: { count: number }) {
  return (
    <div>
      {count && <span className="badge">{count}</span>}
    </div>
  )
}

// When count = 0, renders: <div>0</div>
// When count = 5, renders: <div><span class="badge">5</span></div>
```

**正确：当计数为 0 时不渲染任何内容**

```tsx
function Badge({ count }: { count: number }) {
  return (
    <div>
      {count > 0 ? <span className="badge">{count}</span> : null}
    </div>
  )
}

// When count = 0, renders: <div></div>
// When count = 5, renders: <div><span class="badge">5</span></div>
```

### 6.10 使用 React DOM 资源提示

**影响：高（减少关键资源的加载时间）**

React DOM 提供 API 来提示浏览器所需的资源。这些在服务器组件中特别有用，可以在客户端收到 HTML 之前开始加载资源。

- **`prefetchDNS(href)`**：解析您希望连接的域的 DNS

- **`preconnect(href)`**：建立到服务器的连接（DNS + TCP + TLS）

- **`preload(href, options)`**：获取您即将使用的资源（样式表、字体、脚本、图像）

- **`preloadModule(href)`**：获取您即将使用的 ES 模块

- **`preinit(href, options)`**：获取并评估样式表或脚本

- **`preinitModule(href)`**：获取并评估 ES 模块

**示例：预连接到第三方 API**

```tsx
import { preconnect, prefetchDNS } from 'react-dom'

export default function App() {
  prefetchDNS('https://analytics.example.com')
  preconnect('https://api.example.com')

  return <main>{/* content */}</main>
}
```

**示例：预加载关键字体和样式**

```tsx
import { preload, preinit } from 'react-dom'

export default function RootLayout({ children }) {
  // Preload font file
  preload('/fonts/inter.woff2', { as: 'font', type: 'font/woff2', crossOrigin: 'anonymous' })

  // Fetch and apply critical stylesheet immediately
  preinit('/styles/critical.css', { as: 'style' })

  return (
    <html>
      <body>{children}</body>
    </html>
  )
}
```

**示例：代码分割路由的预加载模块**

```tsx
import { preloadModule, preinitModule } from 'react-dom'

function Navigation() {
  const preloadDashboard = () => {
    preloadModule('/dashboard.js', { as: 'script' })
  }

  return (
    <nav>
      <a href="/dashboard" onMouseEnter={preloadDashboard}>
        Dashboard
      </a>
    </nav>
  )
}
```

**何时使用每个：**

|应用程序接口 |使用案例 |

|-----|----------|

| `prefetchDNS`|您稍后将连接到的第三方域 |

| `preconnect`|您将立即获取的 API 或 CDN |

| `preload`|当前页面所需的关键资源 |

| `preloadModule`| JS 模块可能用于下一个导航 |

| `preinit`|必须尽早执行的样式表/脚本 |

| `preinitModule`|必须提前执行的 ES 模块 |

参考：[https://react.dev/reference/react-dom#resource-preloading-apis](https://react.dev/reference/react-dom#resource-preloading-apis)

### 6.11 在手动加载状态上使用 useTransition

**影响：低（减少重新渲染并提高代码清晰度）**

使用`useTransition`而不是手动`useState`对于加载状态。这提供了内置`isPending`状态并自动管理转换。

**错误：手动加载状态**

```tsx
function SearchResults() {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState([])
  const [isLoading, setIsLoading] = useState(false)

  const handleSearch = async (value: string) => {
    setIsLoading(true)
    setQuery(value)
    const data = await fetchResults(value)
    setResults(data)
    setIsLoading(false)
  }

  return (
    <>
      <input onChange={(e) => handleSearch(e.target.value)} />
      {isLoading && <Spinner />}
      <ResultsList results={results} />
    </>
  )
}
```

**正确：具有内置挂起状态的 useTransition**

```tsx
import { useTransition, useState } from 'react'

function SearchResults() {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState([])
  const [isPending, startTransition] = useTransition()

  const handleSearch = (value: string) => {
    setQuery(value) // Update input immediately
    
    startTransition(async () => {
      // Fetch and update results
      const data = await fetchResults(value)
      setResults(data)
    })
  }

  return (
    <>
      <input onChange={(e) => handleSearch(e.target.value)} />
      {isPending && <Spinner />}
      <ResultsList results={results} />
    </>
  )
}
```

**好处：**

- **自动挂起状态**：无需手动管理`setIsLoading(true/false)`

- **错误恢复**：即使转换抛出异常，挂起状态也会正确重置

- **更好的响应能力**：在更新期间保持 UI 响应能力

- **中断处理**：新的转换会自动取消待处理的转换

参考：[https://react.dev/reference/react/useTransition](https://react.dev/reference/react/useTransition)

---

## 7. JavaScript 性能

**影响：低-中**

对热路径的微观优化可以带来有意义的改进。

### 7.1 避免布局抖动

**影响：中（防止强制同步布局并减少性能瓶颈）**

避免样式写入与布局读取交错。当您读取布局属性时（例如`offsetWidth`, `getBoundingClientRect()`， 或者`getComputedStyle()`）在样式更改之间，浏览器被迫触发同步重排。

**这没问题：浏览器批量样式更改**

```typescript
function updateElementStyles(element: HTMLElement) {
  // Each line invalidates style, but browser batches the recalculation
  element.style.width = '100px'
  element.style.height = '200px'
  element.style.backgroundColor = 'blue'
  element.style.border = '1px solid black'
}
```

**不正确：交错读写会强制回流**

```typescript
function layoutThrashing(element: HTMLElement) {
  element.style.width = '100px'
  const width = element.offsetWidth  // Forces reflow
  element.style.height = '200px'
  const height = element.offsetHeight  // Forces another reflow
}
```

**正确：批量写入，然后读取一次**

```typescript
function updateElementStyles(element: HTMLElement) {
  // Batch all writes together
  element.style.width = '100px'
  element.style.height = '200px'
  element.style.backgroundColor = 'blue'
  element.style.border = '1px solid black'
  
  // Read after all writes are done (single reflow)
  const { width, height } = element.getBoundingClientRect()
}
```

**正确：批量读取，然后写入**

```typescript
function updateElementStyles(element: HTMLElement) {
  element.classList.add('highlighted-box')
  
  const { width, height } = element.getBoundingClientRect()
}
```

**更好：使用 CSS 类**

**反应示例：**

```tsx
// Incorrect: interleaving style changes with layout queries
function Box({ isHighlighted }: { isHighlighted: boolean }) {
  const ref = useRef<HTMLDivElement>(null)
  
  useEffect(() => {
    if (ref.current && isHighlighted) {
      ref.current.style.width = '100px'
      const width = ref.current.offsetWidth // Forces layout
      ref.current.style.height = '200px'
    }
  }, [isHighlighted])
  
  return <div ref={ref}>Content</div>
}

// Correct: toggle class
function Box({ isHighlighted }: { isHighlighted: boolean }) {
  return (
    <div className={isHighlighted ? 'highlighted-box' : ''}>
      Content
    </div>
  )
}
```

如果可能的话，优先选择 CSS 类而不是内联样式。 CSS 文件由浏览器缓存，类提供更好的关注点分离并且更易于维护。

看[this gist](https://gist.github.com/paulirish/5d52fb081b3570c81e3a)和[CSS Triggers](https://csstriggers.com/)有关布局强制操作的更多信息。

### 7.2 为重复查找构建索引映射

**影响：低-中（1M 操作到 2K 操作）**

多种的`.find()`同一个键的调用应该使用 Map。

**不正确（每次查找 O(n)）：**

```typescript
function processOrders(orders: Order[], users: User[]) {
  return orders.map(order => ({
    ...order,
    user: users.find(u => u.id === order.userId)
  }))
}
```

**正确（每次查找 O(1)）：**

```typescript
function processOrders(orders: Order[], users: User[]) {
  const userById = new Map(users.map(u => [u.id, u]))

  return orders.map(order => ({
    ...order,
    user: userById.get(order.userId)
  }))
}
```

构建一次映射 (O(n))，那么所有查找都是 O(1)。

对于 1000 个订单 × 1000 个用户：1M 操作 → 2K 操作。

### 7.3 循环中的缓存属性访问

**影响：低-中（减少查找）**

在热路径中缓存对象属性查找。

**错误：3次查找×N次迭代**

```typescript
for (let i = 0; i < arr.length; i++) {
  process(obj.config.settings.value)
}
```

**正确：总共 1 次查找**

```typescript
const value = obj.config.settings.value
const len = arr.length
for (let i = 0; i < len; i++) {
  process(value)
}
```

### 7.4 缓存重复的函数调用

**影响：中（避免冗余计算）**

当在渲染期间使用相同的输入重复调用相同的函数时，使用模块级 Map 来缓存函数结果。

**错误：冗余计算**

```typescript
function ProjectList({ projects }: { projects: Project[] }) {
  return (
    <div>
      {projects.map(project => {
        // slugify() called 100+ times for same project names
        const slug = slugify(project.name)
        
        return <ProjectCard key={project.id} slug={slug} />
      })}
    </div>
  )
}
```

**正确：缓存结果**

```typescript
// Module-level cache
const slugifyCache = new Map<string, string>()

function cachedSlugify(text: string): string {
  if (slugifyCache.has(text)) {
    return slugifyCache.get(text)!
  }
  const result = slugify(text)
  slugifyCache.set(text, result)
  return result
}

function ProjectList({ projects }: { projects: Project[] }) {
  return (
    <div>
      {projects.map(project => {
        // Computed only once per unique project name
        const slug = cachedSlugify(project.name)
        
        return <ProjectCard key={project.id} slug={slug} />
      })}
    </div>
  )
}
```

**单值函数的更简单模式：**

```typescript
let isLoggedInCache: boolean | null = null

function isLoggedIn(): boolean {
  if (isLoggedInCache !== null) {
    return isLoggedInCache
  }
  
  isLoggedInCache = document.cookie.includes('auth=')
  return isLoggedInCache
}

// Clear cache when auth changes
function onAuthChange() {
  isLoggedInCache = null
}
```

使用 Map （而不是钩子），这样它就可以在任何地方工作：实用程序、事件处理程序，而不仅仅是 React 组件。

参考：[https://vercel.com/blog/how-we-made-the-vercel-dashboard-twice-as-fast](https://vercel.com/blog/how-we-made-the-vercel-dashboard-twice-as-fast)

### 7.5 缓存存储API调用

**影响：低-中（减少昂贵的 I/O）**

`localStorage`, `sessionStorage`， 和`document.cookie`是同步的并且昂贵。缓存读取内存中的内容。

**不正确：每次调用时都会读取存储**

```typescript
function getTheme() {
  return localStorage.getItem('theme') ?? 'light'
}
// Called 10 times = 10 storage reads
```

**正确：地图缓存**

```typescript
const storageCache = new Map<string, string | null>()

function getLocalStorage(key: string) {
  if (!storageCache.has(key)) {
    storageCache.set(key, localStorage.getItem(key))
  }
  return storageCache.get(key)
}

function setLocalStorage(key: string, value: string) {
  localStorage.setItem(key, value)
  storageCache.set(key, value)  // keep cache in sync
}
```

使用 Map （而不是钩子），这样它就可以在任何地方工作：实用程序、事件处理程序，而不仅仅是 React 组件。

**Cookie 缓存：**

```typescript
let cookieCache: Record<string, string> | null = null

function getCookie(name: string) {
  if (!cookieCache) {
    cookieCache = Object.fromEntries(
      document.cookie.split('; ').map(c => c.split('='))
    )
  }
  return cookieCache[name]
}
```

**重要：外部更改无效**

```typescript
window.addEventListener('storage', (e) => {
  if (e.key) storageCache.delete(e.key)
})

document.addEventListener('visibilitychange', () => {
  if (document.visibilityState === 'visible') {
    storageCache.clear()
  }
})
```

如果存储可以从外部更改（另一个选项卡，服务器设置的 cookie），则使缓存无效：

### 7.6 组合多个数组迭代

**影响：低-中（减少迭代）**

多种的`.filter()`或者`.map()`调用会多次迭代数组。合并为一个循环。

**错误：3次迭代**

```typescript
const admins = users.filter(u => u.isAdmin)
const testers = users.filter(u => u.isTester)
const inactive = users.filter(u => !u.isActive)
```

**正确：1 次迭代**

```typescript
const admins: User[] = []
const testers: User[] = []
const inactive: User[] = []

for (const user of users) {
  if (user.isAdmin) admins.push(user)
  if (user.isTester) testers.push(user)
  if (!user.isActive) inactive.push(user)
}
```

### 7.7 使用 requestIdleCallback 推迟非关键工作

**影响：中（在后台任务期间保持 UI 响应）**

使用`requestIdleCallback()`在浏览器空闲期间安排非关键工作。这使得主线程可以自由用于用户交互和动画，从而减少卡顿并提高感知性能。

**不正确：在用户交互期间阻塞主线程**

```typescript
function handleSearch(query: string) {
  const results = searchItems(query)
  setResults(results)

  // These block the main thread immediately
  analytics.track('search', { query })
  saveToRecentSearches(query)
  prefetchTopResults(results.slice(0, 3))
}
```

**正确：将非关键工作推迟到空闲时间**

```typescript
function handleSearch(query: string) {
  const results = searchItems(query)
  setResults(results)

  // Defer non-critical work to idle periods
  requestIdleCallback(() => {
    analytics.track('search', { query })
  })

  requestIdleCallback(() => {
    saveToRecentSearches(query)
  })

  requestIdleCallback(() => {
    prefetchTopResults(results.slice(0, 3))
  })
}
```

**所需工作超时：**

```typescript
// Ensure analytics fires within 2 seconds even if browser stays busy
requestIdleCallback(
  () => analytics.track('page_view', { path: location.pathname }),
  { timeout: 2000 }
)
```

**分块大型任务：**

```typescript
function processLargeDataset(items: Item[]) {
  let index = 0

  function processChunk(deadline: IdleDeadline) {
    // Process items while we have idle time (aim for <50ms chunks)
    while (index < items.length && deadline.timeRemaining() > 0) {
      processItem(items[index])
      index++
    }

    // Schedule next chunk if more items remain
    if (index < items.length) {
      requestIdleCallback(processChunk)
    }
  }

  requestIdleCallback(processChunk)
}
```

**针对不支持的浏览器提供后备：**

```typescript
const scheduleIdleWork = window.requestIdleCallback ?? ((cb: () => void) => setTimeout(cb, 1))

scheduleIdleWork(() => {
  // Non-critical work
})
```

**何时使用：**

- 分析和遥测

- 将状态保存到 localStorage/IndexedDB

- 为可能的下一步操作预取资源

- 处理非紧急数据转换

- 非关键功能的延迟初始化

**何时不使用：**

- 用户发起的需要立即反馈的操作

- 用户正在等待的渲染更新

- 时间敏感的操作

### 7.8 数组比较的早期长度检查

**影响：中高（当长度不同时避免昂贵的操作）**

将数组与昂贵的操作（排序、深度相等、序列化）进行比较时，首先检查长度。如果长度不同，则数组不能相等。

在实际应用程序中，当比较在热路径（事件处理程序、渲染循环）中运行时，这种优化尤其有价值。

**不正确：总是进行昂贵的比较**

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

### 7.9 函数提前返回

**影响：低-中（避免不必要的计算）**

当确定结果时尽早返回，以跳过不必要的处理。

**不正确：即使找到答案后仍处理所有项目**

```typescript
function validateUsers(users: User[]) {
  let hasError = false
  let errorMessage = ''
  
  for (const user of users) {
    if (!user.email) {
      hasError = true
      errorMessage = 'Email required'
    }
    if (!user.name) {
      hasError = true
      errorMessage = 'Name required'
    }
    // Continues checking all users even after error found
  }
  
  return hasError ? { valid: false, error: errorMessage } : { valid: true }
}
```

**正确：第一次错误时立即返回**

```typescript
function validateUsers(users: User[]) {
  for (const user of users) {
    if (!user.email) {
      return { valid: false, error: 'Email required' }
    }
    if (!user.name) {
      return { valid: false, error: 'Name required' }
    }
  }

  return { valid: true }
}
```

### 7.10 提升正则表达式创建

**影响：低-中（避免娱乐）**

不要在渲染中创建 RegExp。提升到模块范围或使用以下命令进行记忆`useMemo()`.

**不正确：每次渲染都有新的正则表达式**

```tsx
function Highlighter({ text, query }: Props) {
  const regex = new RegExp(`(${query})`, 'gi')
  const parts = text.split(regex)
  return <>{parts.map((part, i) => ...)}</>
}
```

**正确：记忆或提升**

```tsx
const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

function Highlighter({ text, query }: Props) {
  const regex = useMemo(
    () => new RegExp(`(${escapeRegex(query)})`, 'gi'),
    [query]
  )
  const parts = text.split(regex)
  return <>{parts.map((part, i) => ...)}</>
}
```

**警告：全局正则表达式具有可变状态**

```typescript
const regex = /foo/g
regex.test('foo')  // true, lastIndex = 3
regex.test('foo')  // false, lastIndex = 0
```

全局正则表达式 (`/g`) 具有可变性`lastIndex`状态：

### 7.11 使用 flatMap 一次性进行映射和过滤

**影响：低-中（消除中间阵列）**

链接`.map().filter(Boolean)`创建一个中间数组并迭代两次。使用`.flatMap()`在一次传递中进行转换和过滤。

**不正确：2次迭代，中间数组**

```typescript
const userNames = users
  .map(user => user.isActive ? user.name : null)
  .filter(Boolean)
```

**正确：1 次迭代，无中间数组**

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

### 7.12 使用循环求最小值/最大值而不是排序

**影响：低（O(n) 而不是 O(n log n)）**

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

**替代方案：小数组的 Math.min/Math.max**

```typescript
const numbers = [5, 2, 8, 1, 9]
const min = Math.min(...numbers)
const max = Math.max(...numbers)
```

这适用于小型数组，但由于扩展运算符的限制，速度可能会较慢，或者对于非常大的数组会引发错误。最大数组长度在 Chrome 143 中约为 124000，在 Safari 18 中约为 638000；确切的数字可能有所不同 - 请参阅[the fiddle](https://jsfiddle.net/qw1jabsx/4/)。使用循环方法来提高可靠性。

### 7.13 使用集合/映射进行 O(1) 查找

**影响：低-中（O(n) 到 O(1)）**

将数组转换为 Set/Map 以进行重复的成员资格检查。

**不正确（每次检查 O(n)）：**

```typescript
const allowedIds = ['a', 'b', 'c', ...]
items.filter(item => allowedIds.includes(item.id))
```

**正确（每次检查 O(1)）：**

```typescript
const allowedIds = new Set(['a', 'b', 'c', ...])
items.filter(item => allowedIds.has(item.id))
```

### 7.14 使用 toSorted() 代替 sort() 来实现不变性

**影响：中高（防止 React 状态下的突变错误）**

`.sort()`就地改变数组，这可能会导致 React state 和 props 出现错误。使用`.toSorted()`创建一个没有突变的新排序数组。

**不正确：改变原始数组**

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

**正确：创建新数组**

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

**浏览器支持：旧版浏览器的后备**

```typescript
// Fallback for older browsers
const sorted = [...items].sort((a, b) => a.value - b.value)
```

`.toSorted()`适用于所有现代浏览器（Chrome 110+、Safari 16+、Firefox 115+、Node.js 20+）。对于较旧的环境，请使用扩展运算符：

**其他不可变数组方法：**

- `.toSorted()`- 不可变排序

- `.toReversed()`- 不可变的反向

- `.toSpliced()`- 不可变的拼接

- `.with()`- 不可变元素替换

---

## 8. 高级模式

**影响：低**

适用于需要仔细实施的特定案例的高级模式。

### 8.1 初始化应用程序一次，而不是每次安装

**影响：低-中（避免开发中重复初始化）**

不要将每次应用程序加载必须运行一次的应用程序范围初始化放入其中`useEffect([])`一个组件的。组件可以重新安装并且效果将重新运行。请在入口模块中使用模块级防护或顶级 init。

**不正确：在开发中运行两次，重新安装时重新运行**

```tsx
function Comp() {
  useEffect(() => {
    loadFromStorage()
    checkAuthToken()
  }, [])

  // ...
}
```

**正确：每次应用程序加载一次**

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

参考：[https://react.dev/learn/you-might-not-need-an-effect#initializing-the-application](https://react.dev/learn/you-might-not-need-an-effect#initializing-the-application)

### 8.2 在 Refs 中存储事件处理程序

**影响：低（稳定订阅）**

当用于不应重新订阅回调更改的效果时，将回调存储在 refs 中。

**不正确：在每次渲染时重新订阅**

```tsx
function useWindowEvent(event: string, handler: (e) => void) {
  useEffect(() => {
    window.addEventListener(event, handler)
    return () => window.removeEventListener(event, handler)
  }, [event, handler])
}
```

**正确：稳定订阅**

```tsx
import { useEffectEvent } from 'react'

function useWindowEvent(event: string, handler: (e) => void) {
  const onEvent = useEffectEvent(handler)

  useEffect(() => {
    window.addEventListener(event, onEvent)
    return () => window.removeEventListener(event, onEvent)
  }, [event])
}
```

**替代方案：使用`useEffectEvent`如果你使用的是最新的 React：**

`useEffectEvent`为相同的模式提供了更清晰的 API：它创建了一个稳定的函数引用，该引用始终调用最新版本的处理程序。

### 8.3 useEffectEvent 用于稳定回调引用

**影响：低（防止效果重新运行）**

访问回调中的最新值，而不将它们添加到依赖项数组中。防止效果重新运行，同时避免过时的关闭。

**不正确：效果在每次回调更改时重新运行**

```tsx
function SearchInput({ onSearch }: { onSearch: (q: string) => void }) {
  const [query, setQuery] = useState('')

  useEffect(() => {
    const timeout = setTimeout(() => onSearch(query), 300)
    return () => clearTimeout(timeout)
  }, [query, onSearch])
}
```

**正确：使用React的useEffectEvent**

```tsx
import { useEffectEvent } from 'react';

function SearchInput({ onSearch }: { onSearch: (q: string) => void }) {
  const [query, setQuery] = useState('')
  const onSearchEvent = useEffectEvent(onSearch)

  useEffect(() => {
    const timeout = setTimeout(() => onSearchEvent(query), 300)
    return () => clearTimeout(timeout)
  }, [query])
}
```

---

## 参考

1. [https://react.dev](https://react.dev)
2. [https://nextjs.org](https://nextjs.org)
3. [https://swr.vercel.app](https://swr.vercel.app)
4. [https://github.com/shuding/better-all](https://github.com/shuding/better-all)
5. [https://github.com/isaacs/node-lru-cache](https://github.com/isaacs/node-lru-cache)
6. [https://vercel.com/blog/how-we-optimized-package-imports-in-next-js](https://vercel.com/blog/how-we-optimized-package-imports-in-next-js)
7. [https://vercel.com/blog/how-we-made-the-vercel-dashboard-twice-as-fast](https://vercel.com/blog/how-we-made-the-vercel-dashboard-twice-as-fast)
