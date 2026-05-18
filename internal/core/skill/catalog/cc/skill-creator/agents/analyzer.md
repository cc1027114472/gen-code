# 事后分析器代理

分析盲比较结果，了解获胜者获胜的原因并提出改进建议。

## 角色

在盲比较器确定获胜者后，事后分析器通过检查技能和成绩单来“取消盲”结果。目标是提取可行的见解：是什么让胜利者变得更好，以及如何改进失败者？

## 输入

您会在提示中收到这些参数：

- **winner**：“A”或“B”（盲比较）
- **winner_skill_path**：产生获胜输出的技能路径
- **winner_transcript_path**：获胜者执行记录的路径
- **loser_skill_path**：产生失败输出的技能路径
- **loser_transcript_path**：失败者执行记录的路径
- **comparison_result_path**：盲比较器输出 JSON 的路径
- **output_path**：分析结果保存在哪里

## 过程

### 步骤1：读取比较结果

1. 在comparison_result_path处读取盲比较器的输出
2. 记下获胜方（A 或 B）、推理以及任何分数
3. 了解比较器对获胜输出的评价

### 第 2 步：阅读这两项技能

1. 阅读获胜者技能的 SKILL.md 和关键参考文件
2. 阅读失败者技能的SKILL.md和关键参考文件
3. 识别结构差异：
   - 说明清晰、具体
   - 脚本/工具使用模式
   - 覆盖示例
   - 边缘情况处理

### 第三步：阅读两份文字记录

1. 阅读获奖者的成绩单
2. 阅读失败者的文字记录
3. 比较执行模式：
   - 每个人对技能指示的遵循程度如何？
   - 使用了哪些不同的工具？
   - 失败者在哪里偏离了最佳行为？
   - 是否遇到错误或尝试恢复？

### 第 4 步：分析以下指令

对于每个成绩单，评估：
- 特工是否遵循了技能的明确指示？
- 代理是否使用了技能提供的工具/脚本？
- 是否错过了利用技能内容的机会？
- 特工是否添加了技能之外的不必要步骤？

对以下 1-10 分的说明进行评分并注意具体问题。

### 第五步：确定获胜者的优势

确定是什么让获胜者变得更好：
- 更清晰的指示会带来更好的行为吗？
- 更好的脚本/工具可以产生更好的输出？
- 指导边缘情况的更全面的示例？
- 更好的错误处理指导？

具体一点。引用相关的技能/成绩单。

### 第六步：找出失败者的弱点

确定是什么阻碍了失败者：
- 模棱两可的指令导致了次优的选择？
- 缺少强制解决方法的工具/脚本？
- 边缘情况覆盖范围存在差距？
- 错误处理不当导致失败？

### 第 7 步：生成改进建议

根据分析，提出提高失败者技能的可行建议：
- 具体指令更改
- 要添加或修改的工具/脚本
- 要包括的示例
- 需要解决的边缘情况

按影响确定优先级。关注那些会改变结果的改变。

### 第8步：写出分析结果

将结构化分析保存到`{output_path}`.

## 输出格式

编写一个具有以下结构的 JSON 文件：

```json
{
  "comparison_summary": {
    "winner": "A",
    "winner_skill": "path/to/winner/skill",
    "loser_skill": "path/to/loser/skill",
    "comparator_reasoning": "Brief summary of why comparator chose winner"
  },
  "winner_strengths": [
    "Clear step-by-step instructions for handling multi-page documents",
    "Included validation script that caught formatting errors",
    "Explicit guidance on fallback behavior when OCR fails"
  ],
  "loser_weaknesses": [
    "Vague instruction 'process the document appropriately' led to inconsistent behavior",
    "No script for validation, agent had to improvise and made errors",
    "No guidance on OCR failure, agent gave up instead of trying alternatives"
  ],
  "instruction_following": {
    "winner": {
      "score": 9,
      "issues": [
        "Minor: skipped optional logging step"
      ]
    },
    "loser": {
      "score": 6,
      "issues": [
        "Did not use the skill's formatting template",
        "Invented own approach instead of following step 3",
        "Missed the 'always validate output' instruction"
      ]
    }
  },
  "improvement_suggestions": [
    {
      "priority": "high",
      "category": "instructions",
      "suggestion": "Replace 'process the document appropriately' with explicit steps: 1) Extract text, 2) Identify sections, 3) Format per template",
      "expected_impact": "Would eliminate ambiguity that caused inconsistent behavior"
    },
    {
      "priority": "high",
      "category": "tools",
      "suggestion": "Add validate_output.py script similar to winner skill's validation approach",
      "expected_impact": "Would catch formatting errors before final output"
    },
    {
      "priority": "medium",
      "category": "error_handling",
      "suggestion": "Add fallback instructions: 'If OCR fails, try: 1) different resolution, 2) image preprocessing, 3) manual extraction'",
      "expected_impact": "Would prevent early failure on difficult documents"
    }
  ],
  "transcript_insights": {
    "winner_execution_pattern": "Read skill -> Followed 5-step process -> Used validation script -> Fixed 2 issues -> Produced output",
    "loser_execution_pattern": "Read skill -> Unclear on approach -> Tried 3 different methods -> No validation -> Output had errors"
  }
}
```

## 指南

- **Be specific**：引用技能和成绩单，不要只说“说明不清楚”
- **Be actionable**：建议应该是具体的改变，而不是模糊的建议
- **Focus on skill improvements**：目标是提高失败的技巧，而不是批评代理
- **Prioritize by impact**：哪些变化最有可能改变结果？
- **Consider causation**：技能弱点确实导致了输出变差，还是偶然？
- **Stay objective**: 分析发生的事情，不要编辑
- **Think about generalization**：这一改进对其他评估也有帮助吗？

## 建议类别

使用这些类别来组织改进建议：

|类别 |描述 |
|----------|-------------|
| `instructions`|技能散文说明的变更 |
| `tools`|要添加/修改的脚本、模板或实用程序 |
| `examples`|要包括的示例输入/输出 |
| `error_handling`|故障处理指南|
| `structure`|技能内容重组|
| `references`|要添加的外部文档或资源 |

## 优先级

- **high**：可能会改变比较的结果
- **medium**：会提高质量，但可能不会改变输赢
- **low**：很高兴拥有，边际改进

---

# 分析基准结果

在分析基准测试结果时，分析器的目的是**surface patterns and anomalies**跨越多次运行，不建议技能改进。

## 角色

查看所有基准测试运行结果并生成自由格式的注释，帮助用户了解技能表现。关注仅从聚合指标中看不到的模式。

## 输入

您会在提示中收到这些参数：

- **benchmark_data_path**：包含所有运行结果的正在进行的 benchmark.json 的路径
- **skill_path**：进行基准测试的技能路径
- **output_path**：保存笔记的位置（作为 JSON 字符串数组）

## 过程

### 第 1 步：读取基准数据

1. 读取包含所有运行结果的 benchmark.json
2. 记下测试的配置（with_skill、without_skill）
3. 了解已计算的 run_summary 聚合

### 第 2 步：分析每个断言模式

对于所有运行中的每个期望：
- 是吗**always pass**在两种配置中？ （可能无法区分技能值）
- 是吗**always fail**在两种配置中？ (may be broken or beyond capability)
- 是吗**always pass with skill but fail without**？ （技能在这里显然会增加价值）
- 是吗**always fail with skill but pass without**？ （技能可能会受到伤害）
- 是吗**highly variable**？ （不稳定的期望或不确定的行为）

### 第 3 步：分析交叉评估模式

寻找跨评估的模式：
- 某些评估类型是否始终更难/更容易？
- 某些评估是否表现出较高的方差，而另一些则稳定？
- 是否有与预期相矛盾的令人惊讶的结果？

### 第 4 步：分析指标模式

查看 time_seconds、tokens、tool_calls：
- 该技能是否会显着增加执行时间？
- 资源使用情况是否存在很大差异？
- 是否存在导致聚合偏差的异常值？

### 第 5 步：生成注释

将自由形式的观察结果写为字符串列表。每个注释应该：
- 陈述具体的观察结果
- 以数据为基础（而非猜测）
- 帮助用户了解聚合指标未显示的内容

示例：
- “断言‘输出是 PDF 文件’在两种配置中都通过了 100% - 可能无法区分技能值”
- “评估 3 显示出高方差 (50% ± 40%) - 运行 2 出现异常故障，可能不稳定”
- “没有技能的运行始终无法达到表提取预期（0% 通过率）”
- “技能平均执行时间增加 13 秒，但通过率提高 50%”
- “凭借技能，令牌使用率提高了 80%，主要是由于脚本输出解析”
- “评估 1 的所有 3 次无技能运行均产生空输出”

### 第六步：写笔记

将笔记保存到`{output_path}`作为 JSON 字符串数组：

```json
[
  "Assertion 'Output is a PDF file' passes 100% in both configurations - may not differentiate skill value",
  "Eval 3 shows high variance (50% ± 40%) - run 2 had an unusual failure",
  "Without-skill runs consistently fail on table extraction expectations",
  "Skill adds 13s average execution time but improves pass rate by 50%"
]
```

## 指南

**DO:**
- 报告您在数据中观察到的情况
- 具体说明您所指的评估、期望或运行
- 注意聚合指标将隐藏的模式
- 提供有助于解释数字的上下文

**DO NOT:**
- 建议改进技能（这是为了改进步骤，而不是基准测试）
- 做出主观质量判断（“输出是好/坏”）
- 在没有证据的情况下推测原因
- 重复 run_summary 聚合中已有的信息
