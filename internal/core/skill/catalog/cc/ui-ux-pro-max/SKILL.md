---
name: ui-ux-pro-max
description: "用于 Web 与移动端界面的 UI/UX 设计情报库。包含多种风格、配色、字体搭配、产品类型、UX 规则、图表建议与 React Native 相关实现指引。适用于规划、构建、设计、实现、评审、修复、优化和检查界面代码。"
---

# UI/UX Pro Max：设计情报库

这是一个面向 Web 与移动端应用的综合设计指南。它提供可搜索的数据集、优先级规则、设计系统生成流程、按领域深挖的查询方式，以及面向 React Native 的实现建议，帮助你在做界面相关工作时保持系统性与一致性。

## 何时使用

当任务涉及**界面结构、视觉设计决策、交互模式或用户体验质量控制**时，应使用此 Skill。

### 必须使用

在以下场景中必须调用本 Skill：

- 设计新页面，例如落地页、Dashboard、管理后台、SaaS 页面、移动端页面
- 创建或重构 UI 组件，例如按钮、弹窗、表单、表格、图表等
- 选择配色方案、字体系统、间距规范或布局系统
- 评审现有 UI 代码的体验质量、可访问性或视觉一致性
- 实现导航结构、动画或响应式行为
- 做产品级设计决策，例如风格方向、信息层级或品牌表达
- 提升界面的清晰度、完成度、可用性或感知质量

### 推荐使用

在以下场景中建议使用：

- UI 看起来“不够专业”，但原因不清楚
- 正在处理可用性或体验反馈
- 上线前需要做界面质量优化
- 需要对齐跨平台设计体验
- 正在建设设计系统或可复用组件库

### 可以跳过

以下场景通常不需要本 Skill：

- 纯后端逻辑开发
- 只涉及 API 或数据库设计
- 与界面无关的性能优化
- 基础设施或 DevOps 工作
- 非视觉脚本或自动化任务

**判断标准：** 只要任务会改变一个功能“看起来如何、动起来如何、以及用户如何与之交互”，就应该使用本 Skill。

## 规则优先级

给人和代理的通用准则：先按优先级 1→10 决定先关注哪一类规则；如需细节，再用 `--domain <domain>` 查询。脚本不会直接读取下表，但输出逻辑应遵守同样的排序。

| 优先级 | 类别 | 影响等级 | domain | 必查项 | 反模式 |
| --- | --- | --- | --- | --- | --- |
| 1 | 可访问性 | CRITICAL | `ux` | 对比度、Alt 文本、键盘导航、标签语义 | 去掉 focus、无标签图标按钮 |
| 2 | 触控与交互 | CRITICAL | `ux` | 触控面积、目标间距、加载反馈 | 只靠 hover、零反馈状态切换 |
| 3 | 性能 | HIGH | `ux` | 图片格式、懒加载、预留布局空间 | 布局抖动、CLS |
| 4 | 风格选择 | HIGH | `style` / `product` | 风格与产品匹配、一致性、矢量图标 | 随机混搭风格、用 emoji 代替结构图标 |
| 5 | 布局与响应式 | HIGH | `ux` | 移动优先、断点、无横向滚动 | 固定宽度、禁缩放 |
| 6 | 排版与色彩 | MEDIUM | `typography` / `color` | 正文字号、行高、语义色 | 太小的正文字号、组件里散落原始十六进制色值 |
| 7 | 动画 | MEDIUM | `ux` | 150–300ms、动效有意义、空间连续性 | 纯装饰动画、动画宽高、无 reduced-motion |
| 8 | 表单与反馈 | MEDIUM | `ux` | 可见标签、近场错误、辅助说明 | placeholder 代替 label、错误只写在顶部 |
| 9 | 导航模式 | HIGH | `ux` | 可预测返回、底部导航不超载、可 deep link | 导航混乱、返回行为破坏状态 |
| 10 | 图表与数据 | LOW | `chart` | 图例、提示、可访问配色 | 只靠颜色传达意义 |

## 快速参考

### 1. 可访问性（CRITICAL）

- `color-contrast`：普通文本最少 4.5:1，大字号最少 3:1
- `focus-states`：交互元素必须保留可见 focus ring
- `alt-text`：有意义的图片需要描述性 alt 文本
- `aria-labels`：纯图标按钮必须有 `aria-label` 或原生无障碍标签
- `keyboard-nav`：Tab 顺序应与视觉顺序一致，并支持完整键盘操作
- `form-labels`：表单字段应有真实 label，而不是只靠 placeholder
- `skip-links`：Web 长页面应提供跳到主内容的入口
- `heading-hierarchy`：标题层级必须连续，不要跳级
- `color-not-only`：不能只靠颜色表达状态，必须搭配图标或文字
- `dynamic-type`：考虑系统字体放大后的可读性与不截断
- `reduced-motion`：用户要求减少动效时，应降级或关闭动画
- `voiceover-sr`：确保屏幕阅读器阅读顺序与文案意义都正确
- `escape-routes`：弹窗、多步流必须有清晰的取消或返回路径
- `keyboard-shortcuts`：不要破坏系统快捷键；拖拽也要有键盘替代路径

### 2. 触控与交互（CRITICAL）

- `touch-target-size`：触控目标最小 44×44pt / 48×48dp
- `touch-spacing`：触控目标之间至少保留 8px 间距
- `hover-vs-tap`：核心交互不能只依赖 hover
- `loading-buttons`：异步按钮应禁用重复点击并显示加载反馈
- `error-feedback`：错误提示要靠近问题点
- `cursor-pointer`：Web 上可点击元素应有 `cursor: pointer`
- `gesture-conflicts`：避免让主内容区手势与系统手势冲突
- `tap-delay`：Web 上可考虑 `touch-action: manipulation`
- `standard-gestures`：遵守平台默认手势约定
- `system-gestures`：不要拦截系统级返回、控制中心等边缘手势
- `press-feedback`：按下状态要有明确视觉反馈
- `haptic-feedback`：重要确认动作可加轻量触觉反馈，但不要滥用

### 3. 性能（HIGH）

- `image-format`：优先 WebP / AVIF，移动端注意不同平台解码策略
- `lazy-loading`：长内容列表与非首屏媒体要懒加载
- `reserve-space`：图片、骨架屏、广告位要预留空间，避免 CLS
- `virtualize-lists`：长列表优先虚拟化
- `debounce-throttle`：输入搜索、滚动监听与 resize 事件要做节流或防抖
- `animation-cheap`：优先动画 opacity / transform，减少 layout thrash
- `main-thread-budget`：避免把重计算堆到主线程
- `bundle-awareness`：样式系统、图标包和图表库要关注体积

### 4. 风格选择（HIGH）

- `style-match`：风格要与产品类型和用户期待匹配
- `consistency`：同一层级界面使用统一的视觉语言
- `icon-discipline`：结构图标使用矢量图标，不用 emoji 代替
- `brand-fit`：品牌表达应服务于信息层级，而不是压过可用性
- `effects-restraint`：玻璃、阴影、渐变等效果要克制使用

### 5. 布局与响应式（HIGH）

- `mobile-first`：优先从小屏规划信息层级
- `viewport-meta`：Web 需要正确的 viewport 配置
- `no-horizontal-scroll`：默认不允许内容在正常场景下横向滚动
- `breakpoint-consistency`：断点体系应统一
- `safe-areas`：移动端要考虑刘海、底部手势区和状态栏
- `sticky-awareness`：固定栏不能遮挡滚动内容

### 6. 排版与色彩（MEDIUM）

- `line-height`：正文建议 1.5–1.75
- `line-length`：单行文本长度尽量控制在 65–75 字符范围
- `font-pairing`：标题与正文的字体个性应协调
- `font-scale`：使用稳定的字号体系，例如 12 / 14 / 16 / 18 / 24 / 32
- `contrast-readability`：浅底用足够深的文字颜色
- `text-styles-system`：尽量依附平台或设计系统的 type roles
- `weight-hierarchy`：字重应服务层级，不要滥用粗体
- `color-semantic`：组件里优先用语义色 token，而不是散落 raw hex
- `color-dark-mode`：暗色模式不是简单反相，要单独配色与验对比
- `color-accessible-pairs`：前景与背景必须通过对比度校验
- `color-not-decorative-only`：功能色要搭配图标或文字说明
- `truncation-strategy`：优先换行，其次省略号，再提供查看全文方式
- `letter-spacing`：正文不要过度压缩字距
- `number-tabular`：数据、价格、计时器优先用等宽数字
- `whitespace-balance`：留白要帮助分组与层次，而不是被动剩余

### 7. 动画（MEDIUM）

- `duration-range`：常规交互动效一般在 150–300ms
- `meaningful-motion`：动效必须表达状态变化或空间关系
- `exit-faster-than-enter`：退场可略快于进场
- `spring-physics`：弹性动画要像真实物理反馈，不要抖动过度
- `reduced-motion-safe`：减少动效模式下也应能看懂状态变化

### 8. 表单与反馈（MEDIUM）

- `inline-validation`：靠近字段给出错误
- `helper-text`：复杂输入要有前置辅助说明
- `focus-management`：错误出现后应把焦点导向正确位置
- `progressive-disclosure`：按阶段呈现复杂输入，不要一次全砸给用户
- `button-states`：禁用、加载、成功、失败状态要清楚可辨

### 9. 导航模式（HIGH）

- `bottom-nav-limit`：底部导航最多 5 个主项
- `drawer-usage`：抽屉适合次级导航，不适合承载核心主路径
- `back-behavior`：返回应恢复之前的滚动、筛选和输入状态
- `deep-linking`：关键页面应能通过 URL / deep link 直接到达
- `nav-label-icon`：导航项优先图标 + 标签组合
- `nav-state-active`：当前项必须有明确高亮
- `modal-escape`：弹层必须有显式关闭路径

### 10. 图表与数据（LOW）

- `legend-clarity`：图例、单位、轴标签必须清楚
- `tooltip-usefulness`：悬浮提示应提供真正有用的数据补充
- `accessible-colors`：颜色组合应兼顾色觉差异
- `chart-fit`：图表类型要与任务匹配，不要为了炫而选错图表

## 使用前准备

此 Skill 的主工作流依赖 `scripts/search.py`。在执行前先确认本地有 Python：

```bash
python3 --version || python --version
```

如果没有 Python，可按系统安装：

**macOS**
```bash
brew install python3
```

**Ubuntu / Debian**
```bash
sudo apt update && sudo apt install python3
```

**Windows**
```powershell
winget install Python.Python.3.12
```

## 如何使用本 Skill

当用户提出以下需求时应使用：

- “做一个 landing page / dashboard / app 页面”
- “帮我选风格、颜色、字体”
- “这个 UI 为什么不够高级 / 不够专业”
- “检查一下可访问性 / UX / 动效 / 暗黑模式”
- “给我一套设计系统”
- “这个 React Native 页面怎么优化交互与视觉”

## 推荐工作流

### Step 1：分析用户需求

先从请求里提取这些关键信息：

- **产品类型**：例如娱乐、工具、效率、SaaS、电商、内容型、服务型
- **目标用户**：C 端为主，关注年龄、使用环境与使用频率
- **风格关键词**：例如 playful、vibrant、minimal、dark mode、content-first、immersive
- **技术栈**：当前默认以 React Native 为主

### Step 2：生成设计系统（必选）

始终先用 `--design-system` 获取一整套设计建议：

```bash
python3 skills/ui-ux-pro-max/scripts/search.py "<product_type> <industry> <keywords>" --design-system [-p "Project Name"]
```

这个命令会：

1. 并行检索 `product / style / color / landing / typography`
2. 结合 `ui-reasoning.csv` 的规则给出优先推荐
3. 输出完整设计系统：模式、风格、颜色、排版、效果
4. 同时给出应避免的反模式

**示例：**

```bash
python3 skills/ui-ux-pro-max/scripts/search.py "beauty spa wellness service" --design-system -p "Serenity Spa"
```

### Step 2b：持久化设计系统（Master + Overrides）

若要把结果保存为后续会话可复用的真值，加上 `--persist`：

```bash
python3 skills/ui-ux-pro-max/scripts/search.py "<query>" --design-system --persist -p "Project Name"
```

这会生成：

- `design-system/MASTER.md`：全局设计真值
- `design-system/pages/`：页面级覆盖规则目录

若某个页面需要单独覆盖，也可以加 `--page`：

```bash
python3 skills/ui-ux-pro-max/scripts/search.py "<query>" --design-system --persist -p "Project Name" --page "dashboard"
```

层级读取规则如下：

1. 构建某个页面前，先看 `design-system/pages/<page>.md`
2. 如果页面文件存在，其规则覆盖 `MASTER.md`
3. 如果不存在，则完全使用 `MASTER.md`

上下文提示词可写成：

```text
我正在构建 [Page Name] 页面。请先读取 design-system/MASTER.md。
再检查 design-system/pages/[page-name].md 是否存在。
如果页面文件存在，优先使用其规则。
如果不存在，只使用 Master 规则。
然后再生成代码。
```

### Step 3：按需补充领域搜索

生成设计系统后，再用按领域搜索补细节：

```bash
python3 skills/ui-ux-pro-max/scripts/search.py "<keyword>" --domain <domain> [-n <max_results>]
```

常见需求与 domain 对应关系：

| 需求 | domain | 示例 |
| --- | --- | --- |
| 产品类型模式 | `product` | `--domain product "entertainment social"` |
| 更多风格选项 | `style` | `--domain style "glassmorphism dark"` |
| 配色方案 | `color` | `--domain color "entertainment vibrant"` |
| 字体搭配 | `typography` | `--domain typography "playful modern"` |
| 图表建议 | `chart` | `--domain chart "real-time dashboard"` |
| UX 最佳实践 | `ux` | `--domain ux "animation accessibility"` |
| Google Fonts 个体检索 | `google-fonts` | `--domain google-fonts "sans serif variable"` |
| Landing 结构 | `landing` | `--domain landing "hero social-proof"` |
| React / RN 实现建议 | `react` | `--domain react "rerender memo list"` |
| App 界面可访问性 | `web` | `--domain web "accessibilityLabel touch safe-areas"` |
| AI prompt / CSS 关键词 | `prompt` | `--domain prompt "minimalism"` |

### Step 4：查询技术栈实现建议

针对 React Native 的实现建议：

```bash
python3 skills/ui-ux-pro-max/scripts/search.py "<keyword>" --stack react-native
```

然后把设计系统、领域检索和技术栈建议合并起来，输出最终设计与实现方案。

## Search Reference

### 可用 domains

| domain | 用途 |
| --- | --- |
| `product` | 产品类型推荐 |
| `style` | 风格、效果与视觉方向 |
| `typography` | 字体搭配与字体选择 |
| `color` | 配色方案 |
| `landing` | 页面结构与 CTA 组织 |
| `chart` | 图表类型与展示建议 |
| `ux` | UX 最佳实践与反模式 |
| `google-fonts` | 单个字体检索 |
| `react` | React / React Native 实现建议 |
| `web` | App / 移动端界面规范 |
| `prompt` | AI prompt 与 CSS 风格关键词 |

### 可用 stacks

| stack | 聚焦点 |
| --- | --- |
| `react-native` | 组件、导航、列表与移动端交互 |

## 示例流程

### 示例：为 AI 搜索工具生成设计系统

输入拆解：

- 产品类型：工具（AI 搜索引擎）
- 目标用户：希望快速获得智能结果的 C 端用户
- 风格关键词：modern、minimal、content-first、dark mode
- 技术栈：React Native

生成设计系统：

```bash
python3 skills/ui-ux-pro-max/scripts/search.py "AI search tool modern minimal" --design-system -p "AI Search"
```

若需要再查：

```bash
python3 skills/ui-ux-pro-max/scripts/search.py "minimalism dark mode" --domain style
python3 skills/ui-ux-pro-max/scripts/search.py "search loading animation" --domain ux
python3 skills/ui-ux-pro-max/scripts/search.py "list performance navigation" --stack react-native
```

## 输出格式

`--design-system` 支持两种输出格式：

```bash
# 默认：ASCII 盒状输出，适合终端
python3 skills/ui-ux-pro-max/scripts/search.py "fintech crypto" --design-system

# Markdown：适合落文档
python3 skills/ui-ux-pro-max/scripts/search.py "fintech crypto" --design-system -f markdown
```

## 提升结果质量的建议

### Query 策略

- 使用多维关键词：产品 + 行业 + 气质 + 信息密度
- 同一需求尝试不同词组，观察推荐是否更贴近目标
- 始终先跑 `--design-system`，再用 `--domain` 下钻
- 需要 React Native 细节时，再补 `--stack react-native`

### 常见排障

| 症状 | 处理方式 |
| --- | --- |
| 风格或配色难以决策 | 换关键词重新跑 `--design-system` |
| 暗黑模式对比度不稳 | 重点复查色彩规则与暗黑模式规则 |
| 动效不自然 | 回查动效规则与时长、退出节奏 |
| 表单体验差 | 回查表单、错误提示与 focus 管理规则 |
| 导航迷惑 | 回查导航层级与返回行为 |
| 小屏布局破裂 | 回查布局与响应式规则 |
| 列表卡顿 | 回查虚拟化、节流与主线程预算规则 |

### 交付前检查

- 先跑一次 `--domain ux "animation accessibility z-index loading"` 做 UX 复核
- 至少检查 Quick Reference 中 1–3 的高优先级项
- 在小屏手机、横屏和大字号模式下验证
- 分别验证亮色和暗色模式，不要只测一种主题
- 确认所有触控目标 ≥ 44pt，且不会被安全区遮挡

## 专业 UI 的常见硬规则

以下问题最容易让 UI 看起来“不专业”。默认范围主要是 App UI（iOS / Android / React Native / Flutter），不是传统桌面 Web 交互。

### 图标与视觉元素

| 规则 | 标准 | 避免 | 价值 |
| --- | --- | --- | --- |
| 结构图标不用 Emoji | 使用矢量图标库 | 用 🎨🚀⚙️ 做导航或系统控制 | 保证跨平台一致性与可控性 |
| 矢量优先 | 使用 SVG 或平台矢量图标 | 模糊的 PNG 图标 | 保持清晰、可缩放与可主题化 |
| 交互状态稳定 | 用颜色、透明度、阴影做反馈 | 让周边内容抖动的 layout-shift 动效 | 保证质感与稳定性 |
| 官方品牌 Logo | 使用官方资源并遵循间距规范 | 猜 Logo、随意改色改比例 | 避免品牌误用 |
| 图标尺寸一致 | 用 token 定义尺寸 | 20 / 24 / 28pt 随机混用 | 保持节奏 |
| 描边一致 | 同层级保持统一描边粗细 | 粗细风格乱混 | 提升统一性 |
| Filled / Outline 纪律 | 同层级尽量一种风格 | 同级混用填充与描边 | 保持语义清晰 |

### 内容与层级

- 主 CTA 必须一眼可见
- 次级动作应退后，不要跟主 CTA 抢层级
- 不要让装饰性视觉压过信息本身
- 信息密度高时优先分组、留白与标题层级，而不是单纯缩小字号

### 暗黑模式

- 颜色要重新校准，不是简单反相
- 亮色与暗色都要独立验对比
- 边框、分割线和禁用态都要在暗色里保持可辨性
- Scrim / 蒙层强度要足以保证前景可读

## 最终检查清单

### 视觉

- [ ] 主层级、次层级、辅助层级清楚
- [ ] 风格与产品类型匹配
- [ ] 图标、阴影、圆角、描边保持一致
- [ ] 没有多余的视觉噪音

### 交互

- [ ] 所有关键操作都有即时反馈
- [ ] 加载、错误、空状态都有明确信号
- [ ] 返回行为可预测
- [ ] 弹窗、Sheet、抽屉都有清晰退出路径

### 布局

- [ ] 安全区已处理
- [ ] 固定栏不会遮挡滚动内容
- [ ] 小屏、大屏、横屏都验证过
- [ ] 无非预期横向滚动

### 可访问性

- [ ] 主要文本对比度达标
- [ ] 重要图标 / 图片有无障碍标签
- [ ] 表单有标签、提示和近场错误
- [ ] 颜色不是唯一状态信号
- [ ] 支持 reduced-motion 与大字号

### 性能

- [ ] 长列表已考虑虚拟化
- [ ] 重动画优先使用 opacity / transform
- [ ] 输入、滚动、resize 等事件有节流或防抖
- [ ] 资源和视觉效果没有造成明显卡顿

如果你已经完成以上检查，再把本 Skill 产出的设计系统、领域搜索结果和实现建议综合起来，用于真正的界面设计与代码实现。
