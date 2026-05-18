# React 最佳实践

这是一个结构化仓库，用于创建和维护面向代理与 LLM 优化的 React 最佳实践。

## 结构

- `rules/` - 单条规则文件目录（每条规则一个文件）
  - `_sections.md` - 章节元数据（标题、影响、描述）
  - `_template.md` - 新规则模板
  - `area-description.md` - 单条规则文件示例
- `src/` - 构建脚本与工具
- `metadata.json` - 文档元数据（版本、组织、摘要）
- __`AGENTS.md`__ - 编译产物（生成）
- __`test-cases.json`__ - 用于 LLM 评估的测试用例（生成）

## 入门

1. 安装依赖：
   ```bash
   pnpm install
   ```

2. 根据规则构建 `AGENTS.md`：
   ```bash
   pnpm build
   ```

3. 校验规则文件：
   ```bash
   pnpm validate
   ```

4. 提取测试用例：
   ```bash
   pnpm extract-tests
   ```

## 创建新规则

1. 复制 `rules/_template.md` 到 `rules/area-description.md`
2. 选择合适的分类前缀：
   - `async-`：消除瀑布（第 1 节）
   - `bundle-`：包体积优化（第 2 节）
   - `server-`：服务器端性能（第 3 节）
   - `client-`：客户端数据获取（第 4 节）
   - `rerender-`：重新渲染优化（第 5 节）
   - `rendering-`：渲染性能（第 6 节）
   - `js-`：JavaScript 性能（第 7 节）
   - `advanced-`：高级模式（第 8 节）
3. 填写标题和内容
4. 确保有清晰的示例和解释
5. 运行 `pnpm build` 重新生成 `AGENTS.md` 和 `test-cases.json`

## 规则文件结构

每个规则文件应遵循以下结构：

```markdown
---
title: Rule Title Here
impact: MEDIUM
impactDescription: Optional description
tags: tag1, tag2, tag3
---

## Rule Title Here

在这里简要说明规则内容以及它的重要性。

**Incorrect (description of what's wrong):**

```typescript
// 错误的代码示例
```

**Correct (description of what's right):**

```typescript
// 正确的代码示例
```

Optional explanatory text after examples.

Reference: [Link](https://example.com)

## File Naming Convention

- Files starting with `_` are special (excluded from build)
- Rule files: `area-description.md` (e.g., `async-parallel.md`)
- Section is automatically inferred from filename prefix
- Rules are sorted alphabetically by title within each section
- IDs (e.g., 1.1, 1.2) are auto-generated during build

## Impact Levels

- `CRITICAL` - Highest priority, major performance gains
- `HIGH` - Significant performance improvements
- `MEDIUM-HIGH` - Moderate-high gains
- `MEDIUM` - Moderate performance improvements
- `LOW-MEDIUM` - Low-medium gains
- `LOW` - Incremental improvements

## Scripts

- `pnpm build` - Compile rules into AGENTS.md
- `pnpm validate` - Validate all rule files
- `pnpm extract-tests` - Extract test cases for LLM evaluation
- `pnpm dev` - Build and validate

## Contributing

When adding or modifying rules:

1. Use the correct filename prefix for your section
2. Follow the `_template.md` structure
3. Include clear bad/good examples with explanations
4. Add appropriate tags
5. Run `pnpm build` to regenerate AGENTS.md and test-cases.json
6. Rules are automatically sorted by title - no need to manage numbers!

## Acknowledgments

Originally created by [@shuding](https://x.com/shuding) at [Vercel](https://vercel.com).
