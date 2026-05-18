---
name: vite-rolldown
description: Vite 8 Rolldown bundler 与 Oxc transformer 迁移说明
---

# Rolldown 迁移（Vite 8）

Vite 8 使用 Rolldown 这一统一的 Rust bundler，替代了 esbuild + Rollup 的组合。

## 有哪些变化

| 之前（Vite 7） | 之后（Vite 8） |
|-----------------|----------------|
| esbuild（开发时转换） | Oxc Transformer |
| esbuild（依赖预打包） | Rolldown |
| Rollup（生产构建） | Rolldown |
| `rollupOptions` | `rolldownOptions` |
| `esbuild` option | `oxc` option |

## 性能影响

- 生产构建速度比 Rollup 快 10-30 倍
- 开发性能可匹配 esbuild
- 开发与构建阶段的行为更加统一

## 配置迁移

### rollupOptions 到 rolldownOptions 的迁移

```ts
// Before (Vite 7)
export default defineConfig({
  build: {
    rollupOptions: {
      external: ['vue'],
      output: { globals: { vue: 'Vue' } },
    },
  },
})

// After (Vite 8)
export default defineConfig({
  build: {
    rolldownOptions: {
      external: ['vue'],
      output: { globals: { vue: 'Vue' } },
    },
  },
})
```

### esbuild 到 oxc 的迁移

```ts
// Before (Vite 7)
export default defineConfig({
  esbuild: {
    jsxFactory: 'h',
    jsxFragment: 'Fragment',
  },
})

// After (Vite 8)
export default defineConfig({
  oxc: {
    jsx: {
      runtime: 'classic',
      pragma: 'h',
      pragmaFrag: 'Fragment',
    },
  },
})
```

### JSX 配置

```ts
export default defineConfig({
  oxc: {
    jsx: {
      runtime: 'automatic',  // or 'classic'
      importSource: 'react', // for automatic runtime
    },
    jsxInject: `import React from 'react'`,  // auto-inject
  },
})
```

### 自定义转换目标

```ts
export default defineConfig({
  oxc: {
    include: ['**/*.ts', '**/*.tsx'],
    exclude: ['node_modules/**'],
  },
})
```

## 插件兼容性

大多数 Vite 插件无需修改即可继续使用。Rolldown 支持 Rollup 的插件 API。

如果某个插件只在构建阶段工作：

```ts
{
  ...rollupPlugin(),
  enforce: 'post',
  apply: 'build',
}
```

## 新能力

Rolldown 解锁了此前难以实现的能力：

- Full bundle mode（实验性）
- 模块级持久化缓存
- 更灵活的 chunk 拆分
- Module Federation 支持

## 渐进迁移

对于大型项目，可先通过 `rolldown-vite` 进行迁移验证：

```bash
# Step 1: Test with rolldown-vite
pnpm add -D rolldown-vite

# Replace vite import in config
import { defineConfig } from 'rolldown-vite'

# Step 2: Once stable, upgrade to Vite 8
pnpm add -D vite@8
```

## 在框架中覆盖 Vite 版本

当框架依赖较旧版本的 Vite 时：

```json
{
  "pnpm": {
    "overrides": {
      "vite": "8.0.0"
    }
  }
}
```

<!--
来源参考：
- https://vite.dev/blog/announcing-vite8-beta
- https://vite.dev/blog/announcing-vite7
- https://vite.dev/config/shared-options#oxc
-->
