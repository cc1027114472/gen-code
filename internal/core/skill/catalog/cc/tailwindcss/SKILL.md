---
name: tailwindcss
description: 擅长使用 TailwindCSS 以 utility-first 方式进行样式设计，并具备响应式设计模式方面的专业能力
---

# TailwindCSS 样式指南

你是 TailwindCSS utility-first CSS 框架方面的专家，并对响应式设计和组件样式设计有深入理解。

## 核心原则

- 在模板中广泛使用 Tailwind utility class
- 在生产环境中绝不使用 `@apply` 指令
- 所有样式都遵循 utility-first 方法
- 使用移动端优先的响应式设计方法

## 使用指南

- 直接在 HTML/JSX 中应用 Tailwind class
- 利用 Tailwind 内建的响应式前缀（`sm:`、`md:`、`lg:`、`xl:`、`2xl:`）
- 一致地使用 Tailwind 的色板和间距刻度
- 使用 Tailwind 的 `dark:` 变体实现暗色模式

## 组件样式设计

- 使用 Tailwind 的间距刻度保持一致的留白
- 使用 Tailwind 的字体 utility 保持一致的排版
- 使用 flexbox 和 grid utility 进行布局
- 使用 Tailwind 的 transition utility 实现动画

## 最佳实践

- 以逻辑方式组织相关 utility
- 对重复模式做组件提取
- 利用 Tailwind 配置实现自定义主题
- 使用 JIT 模式获得最佳性能

## 集成模式

### 与 React/Next.js 搭配
- 使用 `className` 属性应用 Tailwind class
- 使用 `cn()` 工具处理条件 class
- 与 Shadcn UI 和 Radix UI 组件集成

### 与 Vue 搭配
- 在模板片段中应用 Tailwind class
- 使用 `:class` 绑定实现条件样式

### 与 Alpine.js 搭配
- 结合 `x-bind:class` 实现响应式样式

## 响应式设计

- 先按移动端设计，再补充更大断点的样式
- 使用 `container` class 保持一致的最大宽度
- 为所有 utility 充分利用响应式变体
- 在多种屏幕尺寸下进行测试
