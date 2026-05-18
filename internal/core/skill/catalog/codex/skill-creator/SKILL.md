---
name: skill-creator
description: 用于创建高质量 skill 的指南。当用户希望创建一个新 skill（或更新现有 skill），以通过专业知识、工作流或工具集成来扩展 Codex 能力时，应使用此 skill。
metadata:
  short-description: 创建或更新一个 skill
---

# Skill 创建器

这个 skill 用于指导如何创建高质量、可复用的 skill。

## 关于 Skill

Skill 是模块化、自包含的目录，用来通过提供专业知识、工作流和工具扩展 Codex 的能力。可以把它理解成某个具体领域或任务的“上手指南”，它能把通用型 Codex 转变成具备专门流程知识的专业代理，而这些程序性知识不可能完全靠模型自身天然掌握。

### Skill 提供什么

1. 专业工作流：针对特定领域的多步骤流程
2. 工具集成：处理特定文件格式或 API 的说明
3. 领域知识：公司内部知识、schema、业务规则
4. 打包资源：脚本、参考资料与素材，用于复杂或重复性任务

## ????

### ?????

上下文窗口是公共资源。Skill 要和系统提示词、对话历史、其他 skill 的元数据，以及真实用户请求一起共享上下文窗口。

**默认前提：Codex 已经很聪明。** 只补充 Codex 本来不具备的上下文。审视每一段信息时都问自己：  
“Codex 真的需要这段解释吗？”  
“这一段真的值得它消耗的 token 吗？”

优先用简洁示例，而不是冗长解释。

### ????????

要根据任务的脆弱性和变化度来决定指令的具体程度：

**高自由度（纯文本指令）**：适合多种解法都合理、决策依赖上下文、更多依靠启发式判断的任务。

**中等自由度（伪代码或可配置脚本）**：适合存在推荐模式、允许一定变化、且行为会因配置而不同的任务。

**低自由度（具体脚本、很少参数）**：适合操作易错、流程脆弱、一致性非常重要，或必须严格按顺序执行的任务。

可以把 Codex 想象成在探索路径：如果是一座两侧都是悬崖的窄桥，就需要明确护栏（低自由度）；如果是一块开阔地，就可以给它更多路线选择（高自由度）。

### ???????

在迭代过程中，你可以使用子代理来验证 skill 是否能在真实任务上正常工作，或验证你怀疑的问题是否真实存在。这在你想独立评估一次 skill 的行为、输出或失败模式时很有用。当然，前提是当前环境允许启动子代理。

如果用子代理做验证，请把它当成评测面，而不是剧透答案。目标是了解这个 skill 是否具备泛化能力，而不是让另一个代理靠泄露的上下文重构出正确答案。

优先传递原始产物，例如示例 prompt、输出、diff、日志或 trace。只给完成验证所必需的最小任务上下文。除非验证本身必须依赖这些信息，否则不要把你预期的答案、你怀疑的 bug、你打算的修复方案或你之前的结论传给验证代理。

### Skill 的组成结构

每个 skill 至少包含一个必需的 `SKILL.md` 文件，并可选带一些打包资源：

```
skill-name/
├── SKILL.md (required)
│   ├── YAML frontmatter metadata (required)
│   │   ├── name: (required)
│   │   └── description: (required)
│   └── Markdown instructions (required)
├── agents/ (recommended)
│   └── openai.yaml - UI metadata for skill lists and chips
└── Bundled Resources (optional)
    ├── scripts/          - Executable code (Python/Bash/etc.)
    ├── references/       - Documentation intended to be loaded into context as needed
    └── assets/           - Files used in output (templates, icons, fonts, etc.)
```

#### `SKILL.md`????

每个 `SKILL.md` 由两部分组成：

- **Frontmatter**（YAML）：包含 `name` 和 `description` 字段。这是 Codex 判断何时启用此 skill 的唯一元数据来源，因此对 skill 是什么、在什么场景下该使用它，必须写得清楚且完整。
- **Body**（Markdown）：具体使用说明。只有在 skill 被触发后才会被加载。

#### `agents` ???????

- 面向 UI 的 skill 列表与 chip 元数据
- 在生成前先阅读 `references/openai_yaml.md`，并遵循其中的约束
- 通过阅读 skill 内容来生成面向人的 `display_name`、`short_description` 与 `default_prompt`
- 通过 `scripts/generate_openai_yaml.py` 或 `scripts/init_skill.py`，以 `--interface key=value` 的形式稳定生成这些值
- 更新已有 skill 时，要检查 `agents/openai.yaml` 是否仍与 `SKILL.md` 一致；若已过期则重新生成
- 只有在用户明确提供时，才包含其他可选字段（例如图标、品牌色）
- 具体字段定义与示例参见 `references/openai_yaml.md`

#### ????????

##### 脚本（`scripts/`）

可执行代码（Python/Bash 等），适合那些要求可重复、可确定执行，或经常会被反复重写的任务。

- **什么时候该加**：当相同代码会被频繁重复写，或任务对稳定性要求高时
- **示例**：处理 PDF 旋转的 `scripts/rotate_pdf.py`
- **好处**：更省 token、更确定，而且很多时候无需把脚本内容加载进上下文就能直接执行
- **注意**：在某些场景里，Codex 仍可能需要读取脚本本身，以便打补丁或适配环境差异

##### 参考资料（`references/`）

这类文件用于承载按需加载的文档与参考信息，帮助 Codex 理解业务上下文与处理方法。

- **什么时候该加**：当 skill 工作时需要查阅额外文档
- **示例**：`references/finance.md`、`references/mnda.md`、`references/policies.md`、`references/api_docs.md`
- **适合内容**：数据库 schema、API 文档、领域知识、公司制度、详细流程说明
- **好处**：能让 `SKILL.md` 保持轻量，只在需要时读取详细资料
- **最佳实践**：如果文件很大（>10k words），在 `SKILL.md` 中补上 grep/search 建议
- **避免重复**：信息应该存在于 `SKILL.md` 或 reference 文件中的一个地方，而不是两边都写。详细资料、schema 与长示例更适合放在 reference 中，把核心流程指令保留在 `SKILL.md`

##### 资源文件（`assets/`）

这些文件一般不需要加载进上下文，而是作为最终输出的一部分被直接使用。

- **什么时候该加**：当 skill 需要在输出中复用某些文件
- **示例**：`assets/logo.png`、`assets/slides.pptx`、`assets/frontend-template/`、`assets/font.ttf`
- **适合内容**：模板、图片、图标、脚手架代码、字体、样本文档
- **好处**：把输出用资源与说明文档分离开，让 Codex 可以使用这些文件而无需把它们读进上下文

### Skill 中不要放什么

Skill 里只应包含真正支撑其功能的必要文件。**不要** 创建额外的辅助文档，例如：

- `README.md`
- `INSTALLATION_GUIDE.md`
- `QUICK_REFERENCE.md`
- `CHANGELOG.md`
- 等等

Skill 只应包含 AI 代理完成任务真正需要的信息。不应包含创建过程记录、用户导向文档、额外的 setup 说明、测试手册等。这些只会增加噪音与混乱。

### ?????????

Skill 使用三层加载机制，来更高效地管理上下文：

1. **Metadata（name + description）**：始终在上下文中（约 100 字）
2. **SKILL.md body**：skill 触发时才加载（建议 <5k words）
3. **Bundled resources**：按需读取（理论上无限，因为脚本不需要全量读入上下文）

#### ???????

尽量把 `SKILL.md` 主体控制在 500 行以内，只保留必要内容。接近这个规模时，应拆分文件，并且**必须**在 `SKILL.md` 中明确指向这些拆分出去的文件，并解释何时该读它们。

**核心原则：** 如果一个 skill 支持多种变体、框架或选项，那么 `SKILL.md` 里只保留核心流程与如何选型，把各变体特有的细节、示例与配置放到独立 reference 文件里。

**?? 1????? + ????**

```markdown
# PDF Processing

## Quick start

Extract text with pdfplumber:
[code example]

## Advanced features

- **Form filling**: See [FORMS.md](FORMS.md) for complete guide
- **API reference**: See [REFERENCE.md](REFERENCE.md) for all methods
- **Examples**: See [EXAMPLES.md](EXAMPLES.md) for common patterns
```

这样 Codex 只有在需要时才会读取 `FORMS.md`、`REFERENCE.md` 或 `EXAMPLES.md`。

**?? 2??????**

对于支持多个业务域的 skill，可以按业务域拆分内容，避免加载无关上下文：

```
bigquery-skill/
├── SKILL.md (overview and navigation)
└── reference/
    ├── finance.md
    ├── sales.md
    ├── product.md
    └── marketing.md
```

如果用户问销售指标，Codex 只需要读取 `sales.md`。

同理，支持多个框架或云厂商时，也按变体拆分：

```
cloud-deploy/
├── SKILL.md (workflow + provider selection)
└── references/
    ├── aws.md
    ├── gcp.md
    └── azure.md
```

用户选择 AWS 时，就只读取 `aws.md`。

**Pattern 3: Conditional details**

基础内容写在主文件里，复杂情况链接出去：

```markdown
# DOCX Processing

## Creating documents

Use docx-js for new documents. See [DOCX-JS.md](DOCX-JS.md).

## Editing documents

For simple edits, modify the XML directly.

**For tracked changes**: See [REDLINING.md](REDLINING.md)
**For OOXML details**: See [OOXML.md](OOXML.md)
```

只有用户需要 tracked changes 或 OOXML 细节时，Codex 才去读对应文档。

**Important guidelines:**

- **避免深层嵌套引用**：reference 最好只比 `SKILL.md` 深一层，且都应由 `SKILL.md` 直接链接
- **长 reference 文件要有目录**：超过 100 行的 reference 文件建议在顶部加目录，方便快速预览范围

## Skill Creation Process

创建 skill 的标准步骤：

1. 用具体例子理解这个 skill
2. 规划可复用内容（scripts、references、assets）
3. 初始化 skill（运行 `init_skill.py`）
4. 编辑 skill（编写资源并撰写 `SKILL.md`）
5. 校验 skill（运行 `quick_validate.py`）
6. 基于真实使用结果持续迭代，复杂 skill 还要做 forward-test

如无特殊原因，请按这个顺序来。

### Skill Naming

- 名称只使用小写字母、数字和连字符；把用户提供的标题规范化为 kebab-case（如 `"Plan Mode"` -> `plan-mode`）
- 生成的名称应少于 64 个字符
- 优先用简短、偏动作导向的短语
- 当按工具命名能提高触发准确率时，可以做命名空间区分（如 `gh-address-comments`、`linear-address-issue`）
- skill 目录名必须与 skill 名称完全一致

### Step 1: Understanding the Skill with Concrete Examples

只有在 skill 的使用场景已经非常清楚时，才可以跳过这一步。即使是更新现有 skill，这一步通常仍然有价值。

为了做出高质量 skill，需要先明确它在真实世界中会怎么被使用。这个理解可以来自用户直接给出的例子，也可以来自你生成的例子，再让用户确认。

例如在做 image-editor skill 时，可以问：

- “这个 image-editor skill 需要支持哪些能力？编辑、旋转，还有别的吗？”
- “你能举几个真实使用场景吗？”
- “我能想到像 ‘帮我去掉这张图的红眼’ 或 ‘把图片旋转一下’ 这种请求。还有其他常见说法吗？”
- “用户会说什么样的话来触发这个 skill？”
- “这个 skill 希望创建在哪？如果你没特别要求，我会放到 `$CODEX_HOME/skills`（若 `CODEX_HOME` 未设置，则为 `~/.codex/skills`），这样 Codex 能自动发现。”

不要一次问太多问题，避免压垮用户。优先问最关键的问题，然后再逐步追问。

当你已经清楚这个 skill 应该支持哪些能力时，就可以结束这一步。

### Step 2: Planning the Reusable Skill Contents

接下来，把这些具体案例抽象成可复用资源。对每个例子都思考：

1. 如果从零执行这个任务，需要做什么？
2. 如果以后要反复执行这些任务，哪些脚本、参考资料和素材值得沉淀？

例如：

- 对 `pdf-editor` skill 来说，“帮我旋转这个 PDF” 这类需求说明：  
  1. 每次都手写旋转逻辑很浪费  
  2. 可以把它沉淀成 `scripts/rotate_pdf.py`

- 对 `frontend-webapp-builder` skill 来说，“帮我做一个 todo app / 步数仪表盘” 这类需求说明：  
  1. 每次都要搭一遍相似的前端骨架  
  2. 可以把脚手架放进 `assets/hello-world/`

- 对 `big-query` skill 来说，“今天有多少用户登录了？” 这类需求说明：  
  1. 每次查询都要重新摸清表结构关系  
  2. 可以把 schema 整理成 `references/schema.md`

目标是把反复会用到的东西沉淀成清单：scripts、references 和 assets。

### Step 3: Initializing the Skill

如果是全新 skill，现在就应该实际创建它了。

只有在你正在维护的 skill 已经存在时，才跳过这一步。

在运行 `init_skill.py` 之前，要先问用户希望把 skill 建在哪里。如果用户没指定，默认放到 `$CODEX_HOME/skills`；当 `CODEX_HOME` 未设置时，回退到 `~/.codex/skills`，这样能被自动发现。

创建全新 skill 时，**总是** 应该运行 `init_skill.py`。它会自动生成一个结构正确的模板 skill，大幅降低出错概率。

用法：

```bash
scripts/init_skill.py <skill-name> --path <output-directory> [--resources scripts,references,assets] [--examples]
```

示例：

```bash
scripts/init_skill.py my-skill --path "${CODEX_HOME:-$HOME/.codex}/skills"
scripts/init_skill.py my-skill --path "${CODEX_HOME:-$HOME/.codex}/skills" --resources scripts,references
scripts/init_skill.py my-skill --path ~/work/skills --resources scripts --examples
```

这个脚本会：

- 在指定路径创建 skill 目录
- 生成带有正确 frontmatter 与 TODO 占位的 `SKILL.md` 模板
- 使用传入的 `--interface key=value` 自动生成 `agents/openai.yaml`
- 根据 `--resources` 创建资源目录
- 若设置了 `--examples`，附带生成示例文件

初始化之后，再按需定制 `SKILL.md` 并补充资源。如果用了 `--examples`，请把占位示例替换掉或删除掉。

`display_name`、`short_description` 与 `default_prompt` 应通过阅读 skill 内容来生成，再通过 `--interface key=value` 传给 `init_skill.py`；或者之后用下面这个命令重新生成：

```bash
scripts/generate_openai_yaml.py <path/to/skill-folder> --interface key=value
```

只有在用户明确提供时，才生成额外的可选 interface 字段。完整字段说明参见 `references/openai_yaml.md`。

### Step 4: Edit the Skill

编辑 skill 时要记住：这个 skill 是给“另一个 Codex”使用的。你应当写入那些对 Codex 有帮助、但它自己不一定天然知道的信息，例如程序性知识、领域约束、项目口径和可复用资源。

在做出较大修改后，或者当 skill 本身很 tricky 时，应使用子代理在真实任务或真实产物上 forward-test。做 forward-test 时，传给子代理的是被验证的任务本身，而不是你对问题的诊断。

#### Start with Reusable Skill Contents

实现时优先把前面识别出的可复用资源补上：`scripts/`、`references/`、`assets/`。这一步有时需要用户提供材料，比如品牌素材、模板、内部文档等。

新增脚本必须通过真正运行来测试，确保没有 bug，且输出符合预期。如果脚本很多而且相似，至少测试有代表性的样本。

如果使用了 `--examples`，把那些不需要的占位文件删掉。不要创建不会真正使用的资源目录。

#### Update SKILL.md

**Writing Guidelines:** 一律使用祈使式/不定式风格来写指令。

##### Frontmatter

YAML frontmatter 只写 `name` 与 `description`：

- `name`：skill 名称
- `description`：这是 skill 的首要触发机制，因此非常重要
  - 同时写清“这个 skill 是做什么的”以及“什么时候该用它”
  - 所有“何时使用”的信息都应该放在 `description` 中，而不是正文里
  - 例如一个 `docx` skill 的描述可以是：  
    “Comprehensive document creation, editing, and analysis with support for tracked changes, comments, formatting preservation, and text extraction. Use when Codex needs to work with professional documents (.docx files) for: (1) Creating new documents, (2) Modifying or editing content, (3) Working with tracked changes, (4) Adding comments, or any other document tasks”

不要在 YAML frontmatter 中加入其他字段。

##### Body

正文用于写这个 skill 的使用说明，以及它打包资源的使用方法。

### Step 5: Validate the Skill

当 skill 开发完成后，先运行校验脚本，尽早发现基础问题：

```bash
scripts/quick_validate.py <path/to/skill-folder>
```

该脚本会检查 YAML frontmatter 格式、必填字段和命名规则。如果失败，就按提示修复后再运行一次。

### Step 6: Iterate

在测试之后，你可能会发现这个 skill 复杂到需要 forward-test，或者用户会提出改进需求。

用户测试通常会在他们刚使用完这个 skill 后立刻发生，因为那时上下文最完整。

**Forward-testing 与迭代流程：**

1. 用 skill 做真实任务
2. 观察它在哪些地方挣扎、低效或出错
3. 识别 `SKILL.md` 或打包资源需要怎样修改
4. 实施修改并再次测试
5. 合理且必要时继续做 forward-test

## Forward-testing

做 forward-test 时，启动子代理，用最少上下文去压力测试 skill。
子代理**不应该知道**自己是在“测试一个 skill”。它应被当成一个正常接收用户任务的代理。

对子代理的 prompt 应类似：
`Use $skill-x at /path/to/skill-x to solve problem y`

而不是：
`Review the skill at /path/to/skill-x; pretend a user asks you to...`

Forward-testing 的决策规则：
- 宁可多做 forward-test，也不要少做
- 但如果你认为 forward-test 可能：
  - 花费很长时间
  - 需要额外用户授权
  - 修改线上生产系统

  那就先给用户看你的计划 prompt，并请求两件事：
  1. 是否同意
  2. 是否需要修改 prompt

做 forward-test 时还要注意：
- 用新的线程做独立评估
- 传入 skill 和真实用户可能会说的请求方式
- 传原始产物，不传你的结论
- 不要泄露预期答案或预期修复
- 每轮迭代后都重新从源产物构建上下文
- 认真审阅子代理的输出、推理过程与生成产物
- 不要让旧产物残留在磁盘上干扰后续评估；必要时清理掉

如果 forward-test 只有在子代理看到了泄露上下文时才成功，那就说明 skill 或测试方式本身还不够扎实，先收紧它们，再谈是否可信。
