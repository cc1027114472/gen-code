---
name: react-vite-expert
description: Complete React + Vite expertise for building optimized, scalable applications. Covers project architecture, folder structure, component patterns, performance optimization, TypeScript best practices, testing, and build configuration. Use when building React apps with Vite, organizing projects, optimizing performance, or implementing best practices. Includes code generators, bundle analyzer, and production-ready templates.
---

# React + Vite 专家

## 概述

转变为 React + Vite 专家，深入了解现代 React 开发模式、最佳项目组织、性能优化技术和生产就绪配置。这项技能提供了使用 Vite 作为构建工具构建快速、可维护和可扩展的 React 应用程序所需的一切。

## 核心能力

### 1. 项目架构与组织

指导用户构建 React 应用程序，以实现最大的可维护性和可扩展性。

**参考：`references/project_architecture.md`**

本综合指南涵盖：
- **文件夹结构模式**：基于特征、原子设计、领域驱动
- **文件组织**：托管策略、命名约定
- **导入策略**：路径别名、桶式导出、tree-shaking
- **状态管理组织**：本地与全局，状态放在哪里
- **扩展指南**：如何随着应用程序的增长而发展结构

**何时咨询：**
- 用户问“我应该如何组织我的 React 项目？”
- 开始一个新项目
- 重构现有项目结构
- 应用程序变得难以导航
- 需要建立团队公约

**关键决策树：**
1. **基于功能与基于组件**：阅读“最佳文件夹结构”部分
2. **状态管理策略**：阅读“状态管理策略”部分
3. **导入组织**：阅读“导入策略”部分

### 2. 代码生成和脚手架

使用生产就绪模板自动创建组件、挂钩和功能。

**可用脚本：**

**`scripts/create_component.py`**
生成包含所有必需文件的完整组件：
- 组件文件 (.tsx)
- TypeScript 类型 (.types.ts)
- CSS 模块 (.module.css)
- 测试（.test.tsx）
- 故事书故事 (.stories.tsx) [可选]
- 干净导入的索引文件

```bash
# Create a basic component
python scripts/create_component.py Button --type component

# Create a page component with lazy loading
python scripts/create_component.py Dashboard --type page

# Create component with children prop
python scripts/create_component.py Card --children

# Create component with Storybook story
python scripts/create_component.py Button --story

# Without tests
python scripts/create_component.py SimpleComponent --no-tests
```

**何时使用：**
- 创建任何新组件
- 设置新功能模块
- 需要一致的组件结构
- 想要加快发展

**`scripts/create_hook.py`**
使用常见模式的模板生成自定义挂钩：
- 状态管理挂钩
- 效果挂钩
- 数据获取钩子
- 本地存储挂钩
- 防抖钩子
- 间隔挂钩

```bash
# Create custom hook
python scripts/create_hook.py useAuth --type custom

# Create data fetching hook
python scripts/create_hook.py useUserData --type fetch

# Create localStorage hook
python scripts/create_hook.py useSettings --type localStorage

# Create debounce hook
python scripts/create_hook.py useSearchDebounce --type debounce
```

**何时使用：**
- 提取可重用逻辑
- 创建自定义状态管理
- 需要常用的钩子样式
- 想要自动挂钩测试

### 3、性能优化

优化 React 应用程序以获得最大性能和最小包大小。

**参考：`references/performance_optimization.md`**

本指南涵盖：
- **React 渲染优化**：React.memo()、useMemo()、useCallback()
- **代码分割**：React.lazy()、基于路由的分割、组件分割
- **虚拟化**：使用react-window进行长列表优化
- **去抖动和节流**：输入优化、滚动处理
- **Vite 构建优化**：块分割、缩小、压缩
- **图像优化**：WebP/AVIF、延迟加载、响应式图像
- **网络优化**：API请求优化、预取
- **CSS 性能**：CSS 模块与 CSS-in-JS，关键 CSS
- **网络生命体征跟踪**：测量 LCP、FID、CLS

**何时咨询：**
- 应用程序感觉缓慢或滞后
- 大捆尺寸
- 初始加载时间长
- 用户询问优化
- 准备生产部署
- 绩效审计揭示问题

**快速性能检查表：**
1. 跑步`python scripts/analyze_bundle.py`识别大的依赖关系
2. 查看`references/performance_optimization.md`用于优化策略
3. 对路由应用代码分割：`React.lazy(() => import('./Page'))`
4. 记住昂贵的组件：`React.memo(Component)`
5. 使用`useMemo()`用于昂贵的计算
6. 实现长列表的虚拟化（react-window）
7. 优化图像（WebP、延迟加载）
8. 查看 Vite 配置`assets/vite.config.optimized.ts`

**`scripts/analyze_bundle.py`**
分析构建输出并提供优化建议：

```bash
# Run bundle analysis
python scripts/analyze_bundle.py
```

**分析内容：**
- Package.json 依赖项（标识大型库）
- 导入模式（建议更好地导入以进行树摇动）
- 构建输出（包大小、块分布）
- 提供具体的优化建议

**何时运行：**
- 生产部署前
- 添加新的依赖项后
- 当捆绑包大小意外增加时
- 每月定期审核
- 性能优化会议

### 4. 生产就绪配置

部署优化的 Vite 配置和项目设置。

**可用资产：**

**`assets/vite.config.optimized.ts`**
全面优化的 Vite 配置：
- **路径别名**：干净的导入（@/components、@/hooks 等）
- **手动块分割**：供应商、基于功能的块以实现更好的缓存
- **缩小**：在生产中删除 console.log 的 Terser
- **捆绑分析器**：可视化捆绑组成
- **资产优化**：图像处理、字体加载
- **开发代理**：API代理配置
- **源映射**：条件源映射生成
- **CSS 代码分割**：自动 CSS 分块

**何时使用：**
- 开始新项目
- 优化现有构建
- 设置生产管道
- 需要更好的缓存策略
- 想要分析捆绑包

**使用方法：**
1. 复制`assets/vite.config.optimized.ts`到项目根目录
2. 安装依赖项：`npm install -D rollup-plugin-visualizer`
3. 根据您的功能自定义手动块
4. 使用分析器运行构建：`npm run build:analyze`

**`assets/tsconfig.optimized.json`**
TypeScript 配置：
- **启用严格模式**：在编译时捕获更多错误
- **路径别名**：匹配Vite配置
- **最佳编译器选项**：适用于 Vite 和现代 React
- **未使用的代码检测**：noUnusedLocals、noUnusedParameters
- **类型安全**：noImplicitReturns、noUncheckedIndexedAccess

**何时使用：**
- 开始新的 TypeScript 项目
- 想要更严格的类型检查
- 需要路径别名
- 提高类型安全性

**`assets/package.json.example`**
使用以下命令完成 package.json：
- **所有推荐脚本**：dev、build、test、lint、format
- **基本依赖**：React、React DOM、Router
- **开发依赖项**：TypeScript、ESLint、Prettier、Vitest
- **推荐的可选依赖项**：按用例分类
- **Husky 和 ​​lint-staged 设置**：预提交挂钩
- **CI/CD 脚本**：用于自动化管道

**何时使用：**
- 开始新项目
- 需要剧本推荐
- 设置 CI/CD
- 想要 git hooks
- 需要封装参考

**`assets/project-structure-example.md`**
完整的项目结构：
- **完整目录树**：基于功能的架构
- **关键文件示例**：App.tsx、路由器、提供商、API 设置
- **配置示例**：vitest、eslint、prettier
- **测试设置**：测试实用程序和模拟
- **扩展指南**：如何扩展结构

**何时使用：**
- 从头开始新项目
- 需要结构参考
- 重构现有项目
- 组织教学团队
- 创建项目模板

### 5. React 最佳实践和模式

实施现代 React 模式并避免常见陷阱。

**参考：`references/best_practices.md`**

本指南涵盖：
- **组件模式**：复合组件、渲染道具、HOC、自定义挂钩
- **TypeScript 最佳实践**：键入组件、挂钩、事件、通用组件
- **错误处理**：错误边界、异步错误处理
- **表单处理**：受控组件、验证、表单库
- **测试**：组件测试、钩子测试、模拟
- **常见反模式**：要避免什么以及为什么
- **辅助功能**：a11y 最佳实践、ARIA、键盘导航

**何时咨询：**
- 实现复杂的组件模式
- 需要 TypeScript 指导
- 设置错误处理
- 创建表单
- 编写测试
- 用户问“最好的方法是什么......？”
- 代码审查请求
- 教学 React 模式

**模式决策指南：**
- **复合组件**：用于灵活、可组合的 UI（选项卡、手风琴）
- **自定义 Hooks**：提取和重用逻辑（useAuth、useDebounce）
- **Context + Hook**：跨树共享状态（主题、Auth）
- **Render Props**：与渲染控件共享代码（很少见，大部分被钩子代替）
- **HOC**：添加横切关注点（很少见，大部分被钩子取代）

### 6. TypeScript 卓越

使用正确的 TypeScript 模式编写类型安全的 React 代码。

**关键 TypeScript 模式`references/best_practices.md`:**
- 组件属性类型（接口与类型）
- 事件处理程序输入
- 参考输入
- 通用组件类型
- 钩子打字
- 类型保护和缩小
- 实用程序类型

**当用户询问 TypeScript 时：**
1. 阅读相关部分`references/best_practices.md`
2. 提供类型安全的示例
3. 解释模式背后的“原因”
4. 显示错误和正确的方法

**常见打字稿问题：**
- “我如何输入这个组件？” → 组件 Props 键入部分
- “如何输入事件处理程序？” → 挂钩打字部分
- “如何制作通用组件？” → 通用组件部分
- “如何输入参考号？” → 挂钩打字部分

### 7. 测试策略

对 React 应用程序实施全面的测试。

**测试模​​式`references/best_practices.md`:**
- 使用 React 测试库进行组件测试
- 自定义钩子测试
- 测试实用程序和设置
- 模拟策略
- 集成测试

**测试理念：**
- 测试用户行为，而不是实施
- 测试用户看到的内容和执行的操作
- 模拟外部依赖项
- 使用描述性测试名称
- 安排-行动-断言模式

**当用户需要测试帮助时：**
1. 检查组件生成器是否创建了测试：`scripts/create_component.py`
2. 参考测试部分`references/best_practices.md`
3. 显示测试设置`assets/project-structure-example.md`
4. 为其用例提供具体的测试示例

### 8. 状态管理指南

选择并实施正确的状态管理解决方案。

**状态管理决策树（来自`references/project_architecture.md`):**

```
Is it server data (from API)?
└─ Yes → TanStack Query (React Query)

Is it local to a component?
└─ Yes → useState

Is it shared between 2-3 components?
└─ Yes → Lift state up (props)

Is it global but simple (theme, auth)?
└─ Yes → Context + useState

Is it global and complex?
├─ Small/medium app → Zustand
└─ Large app with complex async → Redux Toolkit
```

**何时咨询`references/project_architecture.md`:**
- 选择状态管理解决方案
- 需要每种方法的代码示例
- 了解权衡
- 迁移状态管理
- 重新渲染的性能问题

## 工作流程示例

### 示例 1：“帮助我利用最佳实践启动一个新的 React 项目”

1. **了解要求**：
   - 询问：项目规模、特点、国家需求、团队规模
   - 确定：使用哪些模式、结构复杂性

2. **提供结构**：
   - 展示`assets/project-structure-example.md`
   - 解释基于功能的架构与更简单的架构
   - 根据项目规模推荐

3. **设置配置**：
   - 复制`assets/vite.config.optimized.ts`
   - 复制`assets/tsconfig.optimized.json`
   - 参考`assets/package.json.example`对于脚本

4. **生成初始组件**：
   ```bash
   # Create basic UI components
   python scripts/create_component.py Button --type component --story
   python scripts/create_component.py Input --type component

   # Create pages
   python scripts/create_component.py HomePage --type page

   # Create hooks
   python scripts/create_hook.py useAuth --type custom
   ```

5. **解释后续步骤**：
   - 设置 git hooks (husky)
   - 配置 ESLint 和 Prettier
   - 设置测试
   - 创建初始路线

### 示例 2：“我的 React 应用程序很慢，我该如何优化它？”

1. **分析当前状态**：
   ```bash
   # Run bundle analyzer
   python scripts/analyze_bundle.py
   ```

2. **查看分析输出**：
   - 识别大的依赖关系
   - 检查是否有重复项
   - 检查导入模式

3. **查阅优化指南**：
   - 读`references/performance_optimization.md`
   - 根据分析重点关注相关部分

4. **应用优化**（按影响顺序）：
   - **代码分割**：实现路由的延迟加载
   - **删除大的依赖项**：建议更轻的替代方案
   - **Memoization**：将 React.memo() 添加到昂贵的组件中
   - **虚拟化**：如果渲染长列表
   - **图像优化**：实现延迟加载，WebP格式
   - **构建优化**：应用`assets/vite.config.optimized.ts`

5. **衡量改进**：
   - 在之前和之后运行构建
   - 比较捆绑包大小
   - 测试网络生命力

### 示例 3：“我应该如何组织不断增长的 React 项目？”

1. **评估当前大小**：
   - 问：有多少个组件？有多少功能？
   - 确定：当前痛点

2. **参考架构指南**：
   - 读`references/project_architecture.md`
   - 部分：“最佳文件夹结构”

3. **推荐结构**：
   - **小型（<10 个组件）**：扁平结构
   - **中 (10-50)**：功能文件夹 + 共享组件
   - **大型（50+）**：基于完整功能的架构

4. **显示具体示例**：
   - 显示相关部分`assets/project-structure-example.md`
   - 解释每个文件夹的用途

5. **提供迁移路径**：
   - 不要一次重构所有内容
   - 从新结构新功能入手
   - 逐步迁移旧代码

### 示例 4：“我需要创建许多类似的组件”

1. **使用组件生成器**：
   ```bash
   # Generate multiple components at once
   python scripts/create_component.py UserCard --type component
   python scripts/create_component.py ProductCard --type component
   python scripts/create_component.py OrderCard --type component
   ```

2. **解释结构**：
   - 显示生成的文件
   - 解释每个文件的用途
   - 根据需要定制

3. **创建共享模式**：
   - 将公共属性提取到共享类型
   - 创建基础 Card 组件
   - 使用构图模式

4. **参考模式指南**：
   - 显示复合组件模式`references/best_practices.md`
   - 展示组件组成

### 示例 5：“帮助我为我的 React 应用程序设置测试”

1. **参考测试设置**：
   - 展示`assets/project-structure-example.md`
   - 部分：“src/test/”文件夹结构

2. **设置测试实用程序**：
   - 从示例中复制测试设置
   - 配置 vitest.config.ts
   - 创建测试实用程序（与提供者一起渲染）

3. **生成带有测试的组件**：
   ```bash
   # Components come with tests by default
   python scripts/create_component.py Button
   ```

4. **解释测试模式**：
   - 参考`references/best_practices.md`
   - 部分：“测试最佳实践”
   - 显示组件和钩子测试示例

5. **设置 CI/CD**：
   - 添加测试脚本来自`assets/package.json.example`
   - 配置预提交挂钩
   - 设置 GitHub 操作

## 使用此技能的最佳实践

### 全面
- 不只是回答问题——提供完整的解决方案
- 显示文件结构、配置和示例
- 解释建议背后的“原因”

### 使用所有资源
- **脚本**：生成代码以保持一致性
- **参考文献**：深入探讨概念
- **资产**：生产就绪的配置和示例

### 遵循此命令
1. **理解**：提出澄清问题
2. **参考**：查阅相关文档
3. **生成**：适用时使用脚本
4. **解释**：教授模式/概念
5. **提供**：给出完整的工作示例

### 优先考虑性能
- 主动提出优化建议
- 定期运行捆绑分析器
- 推荐默认延迟加载
- 使用优化配置

### 教授最佳实践
- 展示错误的方式与正确的方式
- 解释权衡
- 参考 TypeScript 严格模式
- 鼓励测试

### 保持井井有条
- 尽早推荐基于特征的结构
- 从一开始就使用路径别名
- 建立命名约定
- 规划规模

## 参考文档

### 参考文献/project_architecture.md
**阅读时间：**
- 构建新项目
- 组织现有项目
- 选择状态管理
- 设置导入
- 用户问“我应该如何组织......？”

**关键部分：**
- 最佳文件夹结构（2种模式）
- 命名约定
- 组件组织模式
- 状态管理策略
- 进口策略
- 决策矩阵

### 参考文献/performance_optimization.md
**阅读时间：**
- 应用程序运行缓慢
- 大捆尺寸
- 优化生产
- 用户询问性能
- 部署前

**关键部分：**
- React 渲染优化（memo、useMemo、useCallback）
- 代码分割
- 虚拟化
- Vite构建优化
- 图像优化
- 网络性能
- CSS 性能
- 网络生命体征追踪

### 参考文献/best_practices.md
**阅读时间：**
- 实施模式
- TypeScript 问题
- 错误处理设置
- 表单实施
- 测试问题
- 代码审查

**关键部分：**
- 组件模式（5 种模式）
- TypeScript 最佳实践
- 错误处理模式
- 表格处理
- 测试最佳实践
- 常见的反模式
- 无障碍

## 快速参考

### 常用命令
```bash
# Generate component
python scripts/create_component.py ComponentName --type component

# Generate page
python scripts/create_component.py PageName --type page

# Generate hook
python scripts/create_hook.py useHookName --type custom

# Analyze bundle
python scripts/analyze_bundle.py
```

### 常见问题
- “我如何构建我的项目？” →`references/project_architecture.md`
- “如何优化性能？” →`references/performance_optimization.md`+ 运行`analyze_bundle.py`
- “我应该使用什么模式？” →`references/best_practices.md`
- “如何配置Vite？” →`assets/vite.config.optimized.ts`
- “我的 package.json 应该是什么样子？” →`assets/package.json.example`

### 文件结构优先级
1. 基于功能（大型应用程序） - 请参阅`references/project_architecture.md`
2. 基于组件（中型应用程序） - 查看更简单的结构
3. 扁平（小型应用程序）- 最小的组织

### 性能优先
1. 代码分割（路由优先）
2. 删除大的依赖项
3. 延迟加载（图像、组件）
4. 记忆（昂贵的组件）
5. 虚拟化（长列表）

### 状态管理优先级
1. 服务器数据 → React 查询
2. 本地状态 → useState
3. 共享简单状态 → 上下文
4. 全局复杂状态 → Zustand 或 Redux Toolkit

## 何时不使用此技能

- **非 React 框架**：Next.js 有自己的模式（使用 Next.js 技巧）
- **React Native**：移动设备有不同的模式
- **类组件**：重点是现代功能组件
- **非 Vite 构建工具**：Webpack/Parcel 有不同的配置
- **后端开发**：这是以前端为中心的

对于这些主题，提供一般的 React 指南，但承认其局限性。

## 成功指标

你的 React + Vite 项目应该实现：
- ✅ 捆绑包大小 < 200KB（初始，gzip 压缩）
- ✅ 灯塔评分 > 90
- ✅ 所有测试均通过
- ✅ 没有 ESLint 错误
- ✅ 一致的文件结构
- ✅ 类型安全（TypeScript 严格模式）
- ✅ 快速构建时间（生产< 30 秒）
- ✅ 快速 HMR（< 100 毫秒）
