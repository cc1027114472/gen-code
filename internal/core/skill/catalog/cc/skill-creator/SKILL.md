---
name: skill-creator
description: Create new skills, modify and improve existing skills, and measure skill performance. Use when users want to create a skill from scratch, edit, or optimize an existing skill, run evals to test a skill, benchmark skill performance with variance analysis, or optimize a skill's description for better triggering accuracy.
---

# 技能创造者

创造新技能并迭代改进它们的技能。

从高层次来看，创建技能的过程是这样的：

- 决定你想要该技能做什么以及它应该如何做
- 写一份技能草稿
- 创建一些测试提示并对其运行 claude-with-access-to-the-skill
- 帮助用户定性和定量评估结果
  - 当运行在后台进行时，如果没有任何定量评估，请起草一些定量评估（如果有一些评估，您可以按原样使用，或者如果您认为需要更改它们，则可以进行修改）。然后向用户解释它们（或者如果它们已经存在，则解释已经存在的）
  - 使用`eval-viewer/generate_review.py`脚本向用户显示结果供他们查看，并让他们查看定量指标
- 根据用户对结果评估的反馈重写技能（以及定量基准是否存在任何明显的缺陷）
- 重复直到您满意为止
- 扩大测试集并在更大范围内重试

使用此技能时，您的工作是找出用户在此过程中的位置，然后介入并帮助他们完成这些阶段。例如，也许他们会说“我想为 X 创造一项技能”。您可以帮助缩小他们的含义，写草稿，编写测试用例，弄清楚他们想要如何评估，运行所有提示，然后重复。

另一方面，也许他们已经有了该技能的草稿。在这种情况下，您可以直接进入循环的评估/迭代部分。

当然，您应该始终保持灵活性，如果用户喜欢“我不需要运行一堆评估，只需与我一起交流”，您就可以这样做。

然后在技能完成后（但同样，顺序是灵活的），您还可以运行技能描述改进器，我们有一个完整的单独脚本，以优化技能的触发。

凉爽的？凉爽的。

## 与用户沟通

技能创建器很容易被熟悉编码术语的人们使用。如果你还没有听说过（你怎么可能听说过，最近才开始），现在有一种趋势，Claude 的力量正在激励水管工打开他们的终端，父母和祖父母去谷歌搜索“如何安装 npm”。另一方面，大多数用户可能相当精通计算机。

因此，请注意上下文提示，以了解如何表达您的沟通！在默认情况下，只是为了给您一些想法：

- “评估”和“基准”是边界线，但还可以
- 对于“JSON”和“断言”，您希望看到用户的严肃提示，表明他们在使用它们之前知道这些东西是什么，而不需要解释它们

如果您有疑问，可以简要解释术语，如果您不确定用户是否能理解，请随意用简短的定义来澄清术语。

---

## 创造技能

### 捕捉意图

首先了解用户的意图。当前对话可能已经包含用户想要捕获的工作流程（例如，他们说“将其转化为技能”）。如果是这样，请首先从对话历史记录中提取答案 - 使用的工具、步骤顺序、用户所做的更正、观察到的输入/输出格式。用户可能需要填补空白，并应在继续下一步之前进行确认。

1. 这项技能应该让克劳德做什么？
2. 这个技能应该什么时候触发呢？ （什么用户短语/上下文）
3. 预期的输出格式是什么？
4. 我们是否应该设置测试用例来验证该技能是否有效？具有客观可验证输出（文件转换、数据提取、代码生成、固定工作流程步骤）的技能受益于测试用例。具有主观输出的技能（写作风格、艺术）通常不需要它们。根据技能类型建议适当的默认值，但让用户决定。

### 采访与研究

主动询问有关边缘情况、输入/输出格式、示例文件、成功标准和依赖性的问题。等到这部分解决之后再写测试提示。

检查可用的 MCP - 如果对研究有用（搜索文档、查找类似技能、查找最佳实践），则通过子代理并行研究（如果可用），否则内联研究。准备好上下文以减轻用户的负担。

### 编写技能.md

根据用户访谈，填写以下组成部分：

- **name**：技能标识符
- **description**：什么时候触发，它会做什么。这是主要的触发机制 - 包括该技能的作用以及何时使用该技能的特定上下文。所有“何时使用”信息都放在这里，而不是在正文中。注意：目前克劳德有“触发不足”技能的倾向——当它们有用时不使用它们。为了解决这个问题，请让技能描述有点“咄咄逼人”。例如，您可以写“如何构建一个简单的快速仪表板来显示内部人类数据”，而不是“如何构建一个简单的快速仪表板来显示内部人类数据。当用户提到仪表板、数据可视化、内部指标或想要显示任何类型的公司数据时，确保使用此技能，即使他们没有明确要求‘仪表板’。”
- **compatibility**：所需的工具、依赖项（可选，很少需要）
- **the rest of the skill :)**

### 技能写作指南

#### 技能剖析

```
skill-name/
├── SKILL.md (required)
│   ├── YAML frontmatter (name, description required)
│   └── Markdown instructions
└── Bundled Resources (optional)
    ├── scripts/    - Executable code for deterministic/repetitive tasks
    ├── references/ - Docs loaded into context as needed
    └── assets/     - Files used in output (templates, icons, fonts)
```

#### 渐进式披露

技能采用三级加载系统：
1. **Metadata**（名称 + 描述）- 始终在上下文中（约 100 个字）
2. **SKILL.md body**- 技能触发时在上下文中（<500 行理想）
3. **Bundled resources**- 根据需要（无限制，脚本无需加载即可执行）

这些字数是近似值，如果需要，您可以随意增加。

**Key patterns:**
- 将 SKILL.md 控制在 500 行以内；如果您接近此限制，请添加额外的层次结构以及有关使用该技能的模型接下来应该跟进的位置的明确指示。
- 清楚地从 SKILL.md 中参考文件，并指导何时阅读它们
- 对于大型参考文件（>300 行），请包含目录

**Domain organization**：当一项技能支持多个领域/框架时，按变体组织：
```
cloud-deploy/
├── SKILL.md (workflow + selection)
└── references/
    ├── aws.md
    ├── gcp.md
    └── azure.md
```
克劳德只读了相关的参考文件。

#### 缺乏惊喜原则

这是不言而喻的，但技能不得包含恶意软件、漏洞代码或任何可能危及系统安全的内容。如果描述了技能的内容，那么其意图不应让用户感到惊讶。请勿同意创建误导性技能或旨在促进未经授权的访问、数据泄露或其他恶意活动的技能的请求。不过，像“XYZ 角色扮演”之类的东西还是可以的。

#### 写作模式

更喜欢在说明中使用命令形式。

**Defining output formats**- 你可以这样做：
```markdown
## Report structure
ALWAYS use this exact template:
# [Title]
## Executive summary
## Key findings
## Recommendations
```

**Examples pattern**- 包含示例很有用。您可以像这样设置它们的格式（但是如果“输入”和“输出”在示例中，您可能需要稍微偏离一下）：
```markdown
## Commit message format
**Example 1:**
Input: Added user authentication with JWT tokens
Output: feat(auth): implement JWT-based authentication
```

### 写作风格

尝试向模型解释为什么事情比严厉、发霉的“必须”更重要。使用心理理论并尝试使技能变得通用，而不是过于狭窄于具体示例。首先写草稿，然后用新的眼光审视它并改进它。

### 测试用例

写完技能草稿后，提出 2-3 个真实的测试提示——真正的用户实际上会说的话。与用户分享：[您不必使用这种确切的语言]“这里有一些我想尝试的测试用例。这些看起来正确吗，或者您想添加更多吗？”然后运行它们。

将测试用例保存到`evals/evals.json`。先不要写断言——只写提示。您将在运行进行过程中在下一步中起草断言。

```json
{
  "skill_name": "example-skill",
  "evals": [
    {
      "id": 1,
      "prompt": "User's task prompt",
      "expected_output": "Description of expected result",
      "files": []
    }
  ]
}
```

看`references/schemas.md`对于完整的模式（包括`assertions`字段，您稍后将添加）。

## 运行和评估测试用例

这一部分是一个连续的序列——不要中途停止。请勿使用`/skill-test`或任何其他测试技能。

将结果放入`<skill-name>-workspace/`作为技能目录的同级目录。在工作区中，通过迭代组织结果（`iteration-1/`, `iteration-2/`等），其中每个测试用例都有一个目录（`eval-0/`, `eval-1/`， ETC。）。不要预先创建所有这些 - 只需创建目录即可。

### 第 1 步：在同一回合中生成所有跑动（带技能和底线）

对于每个测试用例，在同一回合中生成两个子代理 - 一个具有技能，一个没有技能。这很重要：不要先生成 with-skill 运行，然后再返回基线。立即启动所有内容，以便所有内容大约在同一时间完成。

**With-skill run:**

```
Execute this task:
- Skill path: <path-to-skill>
- Task: <eval prompt>
- Input files: <eval files if any, or "none">
- Save outputs to: <workspace>/iteration-<N>/eval-<ID>/with_skill/outputs/
- Outputs to save: <what the user cares about — e.g., "the .docx file", "the final CSV">
```

**Baseline run**（相同的提示，但基线取决于上下文）：
- **Creating a new skill**: 一点技巧都没有。同样的提示，没有技能路径，保存到`without_skill/outputs/`.
- **Improving an existing skill**: 旧版本。编辑之前，先对技能进行快照（`cp -r <skill-path> <workspace>/skill-snapshot/`)，然后将基线子代理指向快照。保存到`old_skill/outputs/`.

写一个`eval_metadata.json`对于每个测试用例（断言现在可以为空）。根据每个评估的测试内容为每个评估指定一个描述性名称，而不仅仅是“eval-0”。目录也使用此名称。如果此迭代使用新的或修改的 eval 提示，请为每个新的 eval 目录创建这些文件 - 不要假设它们是从以前的迭代中继承下来的。

```json
{
  "eval_id": 0,
  "eval_name": "descriptive-name-here",
  "prompt": "The user's task prompt",
  "assertions": []
}
```

### 第 2 步：在运行过程中，草拟断言

不要只是等待运行完成——您可以高效地利用这段时间。为每个测试用例起草定量断言并向用户解释。如果断言已经存在于`evals/evals.json`，审查它们并解释它们检查的内容。

好的断言是客观可验证的，并且具有描述性名称——它们应该在基准查看器中清晰地阅读，这样人们看到结果就可以立即了解每个断言检查的内容。主观技能（写作风格、设计质量）可以更好地进行定性评估——不要将断言强加于需要人类判断的事情上。

更新`eval_metadata.json`文件和`evals/evals.json`与一旦起草的断言。还要向用户解释他们将在查看器中看到什么 - 定性输出和定量基准。

### 第 3 步：运行完成后，捕获计时数据

当每个子代理任务完成时，您会收到一条通知，其中包含`total_tokens`和`duration_ms`。立即将此数据保存到`timing.json`在运行目录中：

```json
{
  "total_tokens": 84852,
  "duration_ms": 23332,
  "total_duration_seconds": 23.3
}
```

这是捕获此数据的唯一机会 - 它来自任务通知并且不会保留在其他地方。在每个通知到达时对其进行处理，而不是尝试对它们进行批处理。

### 第 4 步：评分、聚合并启动查看器

所有运行完成后：

1. **Grade each run**— 生成一个分级机子代理（或内联分级），其内容为`agents/grader.md`并根据输出评估每个断言。将结果保存到`grading.json`在每个运行目录中。 grading.json 期望数组必须使用字段`text`, `passed`， 和`evidence`（不是`name`/`met`/`details`或其他变体）——查看器取决于这些确切的字段名称。对于可以通过编程方式检查的断言，编写并运行脚本而不是盯着它——脚本更快、更可靠，并且可以在迭代中重用。

2. **Aggregate into benchmark**— 从技能创建者目录运行聚合脚本：
   ```bash
   python -m scripts.aggregate_benchmark <workspace>/iteration-N --skill-name <name>
   ```
   这会产生`benchmark.json`和`benchmark.md`每个配置的 pass_rate、时间和令牌，以及平均值 ± stddev 和 delta。如果手动生成 benchmark.json，请参阅`references/schemas.md`以获得观众期望的确切模式。
将每个 with_skill 版本放在其基线对应版本之前。

3. **Do an analyst pass**— 阅读基准数据和汇总统计数据可能隐藏的表面模式。看`agents/analyzer.md`（“分析基准结果”部分）了解要寻找的内容 - 例如无论技能如何（无歧视）总是通过的断言、高方差评估（可能不稳定）以及时间/令牌权衡。

4. **Launch the viewer**具有定性输出和定量数据：
   ```bash
   nohup python <skill-creator-path>/eval-viewer/generate_review.py \
     <workspace>/iteration-N \
     --skill-name "my-skill" \
     --benchmark <workspace>/iteration-N/benchmark.json \
     > /dev/null 2>&1 &
   VIEWER_PID=$!
   ```
   对于迭代 2+，还传递`--previous-workspace <workspace>/iteration-<N-1>`.

   **Cowork / headless environments:**如果`webbrowser.open()`不可用或环境无显示，请使用`--static <output_path>`编写独立的 HTML 文件而不是启动服务器。反馈将作为`feedback.json`当用户单击“提交所有评论”时文件。下载后复制`feedback.json`进入工作区目录以供下一次迭代使用。

注意：请使用generate_review.py创建查看器；无需编写自定义 HTML。

5. **Tell the user**例如：“我已在浏览器中打开结果。有两个选项卡 - “输出”可让您单击每个测试用例并留下反馈，“基准”显示定量比较。完成后，请返回此处并告诉我。”

### 用户在查看器中看到的内容

“输出”选项卡一次显示一个测试用例：
- **Prompt**: 被赋予的任务
- **Output**：技能生成的文件，尽可能内联渲染
- **Previous Output**（迭代 2+）：折叠部分显示最后一次迭代的输出
- **Formal Grades**（如果进行评分）：显示断言通过/失败的折叠部分
- **Feedback**：输入时自动保存的文本框
- **Previous Feedback**（迭代 2+）：他们上次的评论，显示在文本框下方

“基准”选项卡显示统计信息摘要：每个配置的通过率、时间和令牌使用情况，以及每次评估的细分和分析师观察。

通过上一个/下一个按钮或箭头键进行导航。完成后，他们点击“提交所有评论”，将所有反馈保存到`feedback.json`.

### 第 5 步：阅读反馈

当用户告诉您他们已经完成时，请阅读`feedback.json`:

```json
{
  "reviews": [
    {"run_id": "eval-0-with_skill", "feedback": "the chart is missing axis labels", "timestamp": "..."},
    {"run_id": "eval-1-with_skill", "feedback": "", "timestamp": "..."},
    {"run_id": "eval-2-with_skill", "feedback": "perfect, love this", "timestamp": "..."}
  ],
  "status": "complete"
}
```

空反馈意味着用户认为没问题。将改进重点放在用户有具体投诉的测试用例上。

完成后杀死查看服务器：

```bash
kill $VIEWER_PID 2>/dev/null
```

---

## 提高技能

这是循环的核心。您已经运行了测试用例，用户已经审查了结果，现在您需要根据他们的反馈来改进该技能。

### 如何思考改进

1. **Generalize from the feedback.**这里发生的大事是，我们正在尝试创建可以在许多不同的提示中使用一百万次（也许是字面上的，也许更多谁知道）的技能。在这里，您和用户只需要一遍又一遍地迭代几个示例，因为这有助于加快速度。用户对这些示例了如指掌，并且可以快速评估新的输出。但是，如果您和用户共同开发的技能仅适用于这些示例，那么它就毫无用处。如果存在一些顽固的问题，您可以尝试扩展并使用不同的隐喻，或推荐不同的工作模式，而不是进行繁琐的过度调整或强制限制必须。尝试起来相对便宜，也许你会得到一些很棒的东西。

2. **Keep the prompt lean.**删除那些不发挥作用的东西。确保阅读脚本，而不仅仅是最终输出 - 如果看起来该技能使模型浪费大量时间做一些没有成效的事情，您可以尝试删除技能中导致模型这样做的部分，看看会发生什么。

3. **Explain the why.**努力解释一下**why**在你要求模型做的所有事情的背后。今天的法学硕士是*聪明的*。他们有良好的心理理论，如果给予良好的驾驭，他们可以超越死记硬背的指示，真正让事情发生。即使用户的反馈很简洁或令人沮丧，也要尝试真正理解该任务以及用户为什么要写他们所写的内容以及他们实际写的内容，然后将这种理解传递到说明中。如果你发现自己写的 ALWAYS 或 NEVER 全部大写，或者使用超级严格的结构，那就是一个黄旗 - 如果可能的话，重新构建并解释推理，以便模型理解为什么你要求的东西很重要。这是一种更人性化、更强大、更有效的方法。

4. **Look for repeated work across test cases.**阅读测试运行的记录，并注意子代理是否都独立编写了类似的帮助程序脚本或对某些内容采取了相同的多步骤方法。如果所有 3 个测试用例都导致子代理编写了`create_docx.py`或一个`build_chart.py`，这是一个强烈的信号，该技能应该捆绑该脚本。写一次，放进去`scripts/`，并告诉技能使用它。这可以避免以后的每次调用都重新发明轮子。

这项任务非常重要（我们正努力在这里每年创造数十亿美元的经济价值！），您的思考时间不是障碍；而是您的思考时间。慢慢来，认真考虑一下事情。我建议写一份修订草案，然后重新审视并进行改进。真正尽最大努力进入用户的头脑并了解他们想要什么和需要什么。

### 迭代循环

技能提升后：

1. 将您的改进应用到技能中
2. 将所有测试用例重新运行到新的`iteration-<N+1>/`目录，包括基线运行。如果你正在创造一项新技能，基线始终是`without_skill`（无技能）——在迭代中保持不变。如果您正在提高现有技能，请使用您对有意义的基线的判断：用户使用的原始版本或之前的迭代。
3. 启动审阅者`--previous-workspace`指向上一次迭代
4. 等待用户审核并告诉您他们已完成
5. 阅读新的反馈，再次改进，重复

继续下去，直到：
- 用户表示很高兴
- 反馈都是空的（一切看起来都不错）
- 你没有取得有意义的进展

---

## 进阶：盲比

对于您想要在技能的两个版本之间进行更严格比较的情况（例如，用户询问“新版本实际上更好吗？”），可以使用盲比较系统。读`agents/comparator.md`和`agents/analyzer.md`了解详情。基本思想是：将两个输出提供给独立代理，而不告诉它哪个是哪个，并让它判断质量。然后分析获胜者获胜的原因。

这是可选的，需要子代理，并且大多数用户不需要它。人工审核循环通常就足够了。

---

## 描述优化

SKILL.md frontmatter 中的描述字段是决定 Claude 是否调用技能的主要机制。创建或改进技能后，提出优化描述以获得更好的触发准确性。

### 第 1 步：生成触发器 eval 查询

创建 20 个评估查询 - 应该触发和不应该触发的混合。保存为 JSON：

```json
[
  {"query": "the user prompt", "should_trigger": true},
  {"query": "another prompt", "should_trigger": false}
]
```

查询必须是现实的，并且是 Claude Code 或 Claude.ai 用户实际会输入的内容。不是抽象的请求，而是具体、具体且具有大量细节的请求。例如，文件路径、有关用户工作或情况的个人上下文、列名称和值、公司名称、URL。一点背景故事。有些可能是小写或包含缩写或拼写错误或随意的言论。混合使用不同的长度，并关注边缘情况，而不是让它们变得清晰（用户将有机会签署它们）。

坏的：`"Format this data"`, `"Extract text from PDF"`, `"Create a chart"`

好的：`"ok so my boss just sent me this xlsx file (its in my downloads, called something like 'Q4 sales final FINAL v2.xlsx') and she wants me to add a column that shows the profit margin as a percentage. The revenue is in column C and costs are in column D i think"`

对于**should-trigger**查询 (8-10)，考虑覆盖范围。你需要相同意图的不同措辞——一些正式，一些随意。包括用户未明确命名技能或文件类型但明确需要它的情况。加入一些不常见的用例以及该技能与其他技能竞争但应该获胜的情况。

对于**should-not-trigger**查询（8-10），最有价值的是未遂事件——与技能共享关键字或概念但实际上需要不同内容的查询。考虑相邻的领域、不明确的措辞（其中简单的关键字匹配会触发但不应该触发），以及查询涉及技能所做的事情但在另一个工具更合适的上下文中的情况。

要避免的关键是：不要使不应该触发的查询明显不相关。 “编写斐波那契函数”作为 PDF 技能的负面测试太简单了——它不测试任何东西。负面案例应该确实很棘手。

### 第 2 步：与用户一起审核

使用 HTML 模板将评估集呈现给用户进行审阅：

1. 阅读模板`assets/eval_review.html`
2. 替换占位符：
   - `__EVAL_DATA_PLACEHOLDER__`→ eval 项的 JSON 数组（周围没有引号 - 这是一个 JS 变量赋值）
   - `__SKILL_NAME_PLACEHOLDER__`→ 技能名称
   - `__SKILL_DESCRIPTION_PLACEHOLDER__`→ 该技能的当前描述
3. 写入临时文件（例如，`/tmp/eval_review_<skill-name>.html`）并打开它：`open /tmp/eval_review_<skill-name>.html`
4. 用户可以编辑查询、切换应该触发、添加/删除条目，然后单击“导出评估集”
5. 文件下载到`~/Downloads/eval_set.json`— 检查下载文件夹中是否有最新版本，以防有多个版本（例如，`eval_set (1).json`)

这一步很重要——错误的评估查询会导致错误的描述。

### 第 3 步：运行优化循环

告诉用户：“这需要一些时间 - 我将在后台运行优化循环并定期检查它。”

将 eval 集保存到工作区，然后在后台运行：

```bash
python -m scripts.run_loop \
  --eval-set <path-to-trigger-eval.json> \
  --skill-path <path-to-skill> \
  --model <model-id-powering-this-session> \
  --max-iterations 5 \
  --verbose
```

使用系统提示符（为当前会话提供动力的提示符）中的模型 ID，以便触发测试与用户实际体验相匹配。

当它运行时，定期跟踪输出，向用户提供有关其正在进行的迭代以及分数的更新信息。

这会自动处理完整的优化循环。它将评估集分为 60% 的训练和 40% 的保留测试，评估当前的描述（每个查询运行 3 次以获得可靠的触发率），然后调用 Claude 根据失败的内容提出改进建议。它会重新评估训练和测试中的每个新描述，迭代最多 5 次。完成后，它会在浏览器中打开一个 HTML 报告，显示每次迭代的结果，并返回 JSON`best_description`——通过测试分数而不是训练分数来选择，以避免过度拟合。

### 技能触发原理

了解触发机制有助于设计更好的评估查询。技能出现在克劳德的身上`available_skills`列出他们的名字+描述，克劳德根据该描述决定是否咨询技能。需要了解的重要一点是，Claude 只会针对自己无法轻松处理的任务咨询技能 - 即使描述完全匹配，简单的一步式查询（例如“阅读此 PDF”）也可能不会触发技能，因为 Claude 可以使用基本工具直接处理它们。当描述匹配时，复杂、多步骤或专门的查询可靠地触发技能。

这意味着您的评估查询应该足够实质性，以便克劳德实际上可以从咨询技能中受益。像“读取文件 X”这样的简单查询是糟糕的测试用例 - 无论描述质量如何，它们都不会触发技能。

### 第 4 步：应用结果

拿`best_description`从 JSON 输出中更新技能的 SKILL.md frontmatter。向用户显示之前/之后并报告分数。

---

### 包装并展示（仅当`present_files`工具可用）

检查您是否有权访问`present_files`工具。如果不这样做，请跳过此步骤。如果这样做，请打包该技能并将 .skill 文件呈现给用户：

```bash
python -m scripts.package_skill <path/to/skill-folder>
```

打包后，引导用户查看结果`.skill`文件路径，以便他们可以安装它。

---

## Claude.ai 特定说明

在 Claude.ai 中，核心工作流程是相同的（草稿 → 测试 → 审查 → 改进 → 重复），但由于 Claude.ai 没有子代理，因此一些机制发生了变化。以下是要适应的内容：

**Running test cases**：没有子代理意味着没有并行执行。对于每个测试用例，请阅读技能的 SKILL.md，然后按照其说明自行完成测试提示。一次做一个。这不像独立的子代理那么严格（您编写了技能并且也在运行它，因此您拥有完整的上下文），但这是一个有用的健全性检查 - 并且人工审核步骤可以弥补。跳过基线运行 - 只需使用技能按要求完成任务即可。

**Reviewing results**：如果您无法打开浏览器（例如，Claude.ai 的虚拟机没有显示，或者您位于远程服务器上），请完全跳过浏览器审核程序。相反，直接在对话中呈现结果。对于每个测试用例，显示提示和输出。如果输出是用户需要查看的文件（例如 .docx 或 .xlsx），请将其保存到文件系统并告诉他们它在哪里，以便他们可以下载和检查它。请求内联反馈：“这看起来怎么样？你有什么想要改变的吗？”

**Benchmarking**：跳过定量基准测试 - 它依赖于基线比较，如果没有子代理，基线比较就没有意义。关注用户的定性反馈。

**The iteration loop**：和以前一样 - 提高技能，重新运行测试用例，寻求反馈 - 只是中间没有浏览器审核者。如果您有文件系统上的迭代目录，您仍然可以将结果组织到其中。

**Description optimization**：本节要求`claude`CLI 工具（特别是`claude -p`）仅在克劳德代码中可用。如果您使用 Claude.ai，请跳过它。

**Blind comparison**：需要子代理。跳过它。

**Packaging**： 这`package_skill.py`脚本可以在任何使用 Python 和文件系统的地方工作。在 Claude.ai 上，您可以运行它，用户可以下载结果`.skill`文件。

**Updating an existing skill**：用户可能会要求您更新现有技能，而不是创建新技能。在这种情况下：
- **Preserve the original name.**记下技能的目录名称和`name`frontmatter 字段——不加改动地使用它们。例如，如果安装的技能是`research-helper`， 输出`research-helper.skill`（不是`research-helper-v2`).
- **Copy to a writeable location before editing.**安装的技能路径可能是只读的。复制到`/tmp/skill-name/`，在那里编辑，然后从副本中打包。
- **If packaging manually, stage in `/tmp/` first**，然后复制到输出目录——直接写入可能会因权限问题而失败。

---

## 协同办公特定说明

如果您在 Cowork，需要了解的主要事项是：

- 您有子代理，因此主要工作流程（并行生成测试用例、运行基线、评分等）都可以正常工作。 （但是，如果遇到严重的超时问题，可以串行而不是并行运行测试提示。）
- 您没有浏览器或显示器，因此在生成 eval 查看器时，请使用`--static <output_path>`编写独立的 HTML 文件而不是启动服务器。然后提供一个链接，用户可以单击该链接在浏览器中打开 HTML。
- 无论出于何种原因，Cowork 设置似乎都阻止 Claude 在运行测试后生成 eval 查看器，因此重申一下：无论您是在 Cowork 还是在 Claude Code 中，运行测试后，您应该始终生成 eval 查看器，以便人们在自己修改技能并尝试进行更正之前查看示例，使用`generate_review.py`（不是写你自己的精品html代码）。提前抱歉，但我将在这里全部大写：*在*自己评估输入之前生成评估查看器。您希望尽快将它们带到人类面前！
- 反馈的工作方式不同：由于没有正在运行的服务器，查看者的“提交所有评论”按钮将下载`feedback.json`作为一个文件。然后您可以从那里阅读它（您可能必须先请求访问权限）。
- 包装工程——`package_skill.py`只需要 Python 和文件系统。
- 描述优化（`run_loop.py` / `run_eval.py`）应该可以在 Cowork 中正常工作，因为它使用`claude -p`通过子流程，而不是浏览器，但请保存它，直到您完全完成技能的制作并且用户同意它处于良好状态。
- **Updating an existing skill**：用户可能会要求您更新现有技能，而不是创建新技能。请遵循上面 claude.ai 部分中的更新指南。

---

## 参考文件

Agents/ 目录包含专用子代理的说明。当您需要生成相关子代理时请阅读它们。

- `agents/grader.md`— 如何根据输出评估断言
- `agents/comparator.md`— 如何在两个输出之间进行盲 A/B 比较
- `agents/analyzer.md`— 如何分析为什么一个版本击败另一个版本

References/ 目录有附加文档：
- `references/schemas.md`— evals.json、grading.json 等的 JSON 结构。

---

再次重复一次核心循环以强调：

- 弄清楚技能是什么
- 起草或编辑技能
- 根据测试提示运行 claude-with-access-to-the-skill
- 与用户一起评估输出：
  - 创建 benchmark.json 并运行`eval-viewer/generate_review.py`帮助用户查看它们
  - 运行定量评估
- 重复直到您和用户都满意为止
- 将最终的技能打包返回给用户。

如果您有这样的事情，请将步骤添加到您的待办事项列表中，以确保您不会忘记。如果您在 Cowork，请特别输入“创建 evals JSON 并运行`eval-viewer/generate_review.py`这样人们就可以在您的 TodoList 中查看测试用例”以确保它发生。

祝你好运！
