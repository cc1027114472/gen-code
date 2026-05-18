---
title: Avoid Barrel File Imports
impact: CRITICAL
impactDescription: 200-800ms import cost, slow builds
tags: bundle, imports, tree-shaking, barrel-files, performance
---

## 避免桶文件导入

直接从源文件而不是桶文件导入，以避免加载数千个未使用的模块。 **桶文件**是重新导出多个模块的入口点（例如，`index.js`确实如此`export * from './module'`).

流行的图标和组件库在其条目文件中最多可以有 10,000 个重新导出**。对于许多 React 包来说，**仅导入它们就需要 200-800 毫秒**，影响开发速度和生产冷启动。

**为什么 tree-shaking 没有帮助：** 当库被标记为外部（未捆绑）时，捆绑器无法优化它。如果将其捆绑以启用树摇动，则分析整个模块图的构建速度会显着变慢。

**不正确（导入整个库）：**

```tsx
import { Check, X, Menu } from 'lucide-react'
// Loads 1,583 modules, takes ~2.8s extra in dev
// Runtime cost: 200-800ms on every cold start

import { Button, TextField } from '@mui/material'
// Loads 2,225 modules, takes ~4.2s extra in dev
```

**正确 - Next.js 13.5+（推荐）：**

```js
// next.config.js - automatically optimizes barrel imports at build time
module.exports = {
  experimental: {
    optimizePackageImports: ['lucide-react', '@mui/material']
  }
}
```

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

参考：[How we optimized package imports in Next.js](https://vercel.com/blog/how-we-optimized-package-imports-in-next-js)
