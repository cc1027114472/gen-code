---
name: "imagegen"
description: "当任务适合使用 AI 生成的位图视觉素材时使用，例如照片、插画、纹理、精灵图、Mockup 或透明背景抠图。适用于 Codex 需要创建全新图片、改造已有图片，或基于参考图生成视觉变体，且最终产物应是位图素材而不是仓库原生代码或矢量文件。若任务更适合编辑现有 SVG/矢量/代码原生资源、延续既有图标或 Logo 体系，或直接用 HTML/CSS/canvas 构建视觉，则不要使用。"
---

# 图片生成 Skill

为当前项目生成或编辑图片（例如网站素材、游戏素材、UI mockup、产品 mockup、线框图、logo 设计、写实图片或信息图）。

## ???????

这个 skill 只有两个顶层模式：

- **Default built-in tool mode (preferred)**：优先使用内建 `image_gen` 工具处理常规图片生成、编辑与简单透明背景请求。不需要 `OPENAI_API_KEY`。
- **Fallback CLI mode**：使用 `scripts/image_gen.py` CLI。仅在用户明确要求 CLI/API/model 路径，或在用户明确确认要使用 `gpt-image-1.5` 的原生透明背景回退方案后使用。需要 `OPENAI_API_KEY`。

在 CLI 回退模式里，CLI 提供三个子命令：

- `generate`
- `edit`
- `generate-batch`

规则：
- 默认优先使用内建 `image_gen` 工具来处理普通图片生成与编辑请求。
- 不要因为普通的质量、尺寸或输出路径控制就切换到 CLI。
- 如果用户明确要求透明图片或透明背景，也先走内建 `image_gen`：提示生成一个纯色可抠除的色键背景，然后用 `$CODEX_HOME/skills/.system/imagegen/scripts/remove_chroma_key.py` 在本地去背。
- 绝不要静默地从内建 `image_gen` 或 CLI `gpt-image-2` 切换到 CLI `gpt-image-1.5`。这属于模型/路径降级，必须先征得用户同意，除非用户已经明确要求 `gpt-image-1.5`、`scripts/image_gen.py` 或 CLI fallback。
- 如果透明背景请求过于复杂，不适合可靠地用色键去背，或用户明确要求原生透明输出，或本地去背验证失败，需要说明：真正的透明背景需要 CLI `gpt-image-1.5 --background transparent --output-format png`，因为 `gpt-image-2` 不支持 `background=transparent`。只有用户确认后，才能执行 CLI 回退。
- 单独出现 `batch` 这个词不意味着必须走 CLI。如果用户只是想批量生成多个素材，但没有明确要求 CLI/API/model 控制，仍然保持内建路径，每个素材或变体发起一次内建调用。
- 如果内建工具失败或不可用，要告诉用户还有 CLI 回退方案，而且它需要 `OPENAI_API_KEY`。只有在用户明确要求后才继续。
- 如果用户明确要求 CLI 模式，就使用内置的 `scripts/image_gen.py` 工作流。不要临时写一套 SDK 脚本替代它。
- 永远不要修改 `scripts/image_gen.py`。如果缺少能力，先问用户再说。

## ????
- 生成全新图片（概念图、产品图、封面图、网站 Hero 图）
- 用一张或多张参考图生成新图片，参考其风格、构图或氛围
- 编辑已有图片（局部重绘、光照/天气变化、背景替换、物体移除、合成、透明背景）
- 为同一任务产出多张素材或多个变体

## ?????
- 扩展或匹配仓库中已有的 SVG/矢量图标集、Logo 体系或插画库
- 制作更适合直接用 SVG、HTML/CSS 或 canvas 实现的简单图形、图表、线框稿或图标
- 对已有原生可编辑格式的小型本地素材做微调
- 用户明确要可确定性的代码原生产物，而不是 AI 生成位图时

## ???

要同时判断两个问题：

1. **Intent**：这是要生成新图，还是编辑现有图片？
2. **Execution strategy**：这是单个素材，还是多个素材/多个变体？

?????
- 如果用户想在保留原图大部分内容的前提下修改现有图片，把它视为 **edit**。
- 如果用户提供图片只是作为风格、构图、氛围或主体参考，把它视为 **generate**。
- 如果用户没有提供图片，也视为 **generate**。

?????
- 在内建路径下，如果要多个素材或多个变体，就按素材/变体分别调用 `image_gen`。
- 在 CLI 回退路径下，只有用户明确选择 CLI 且确实需要多 prompt/多素材时，才用 `generate-batch`。
- 对于多个彼此不同的素材，不要把 `n` 当作替代。`n` 只适合一个 prompt 的多个变体；不同素材要拆成不同调用。

默认假设用户是要生成新图，除非他们明确是在改图。

## ???

1. 先决定顶层模式：默认用内建模式；只有在用户明确要求或明确确认透明背景 CLI 回退后，才进入 CLI。
2. 决定意图：`generate` 还是 `edit`。
3. 决定输出只是预览，还是会被当前项目真正使用。
4. 决定执行策略：单图、重复调用内建工具，还是 CLI `generate-batch`。
5. 一次性收集输入：prompt、必须逐字保留的文本、限制/avoid 项，以及输入图片。
6. 对每一张输入图片，都明确它的角色：
   - 参考图
   - 被编辑目标图
   - 用于插入/风格迁移/合成的辅助图
7. 如果编辑目标图只存在于本地文件系统、且你仍打算走内建路径，先用 `view_image` 把它加载到上下文中。
8. 如果用户要的是照片、插画、sprite、产品图、banner 或其它明确的位图资产，优先使用 `image_gen`，不要随便替换成 SVG/HTML/CSS 占位方案。反过来，如果是要匹配现有 repo-native SVG/矢量/UI 图形，则优先直接编辑那些原生素材。
9. 根据 prompt 具体程度决定增强方式：
   - 如果用户的 prompt 已经很具体，只做结构化整理，不额外加创意要求。
   - 如果 prompt 比较泛，可以做适度增强，但必须真能提升结果质量。
10. 默认使用内建 `image_gen`。
11. 对透明背景请求，按下文的透明图流程处理：先用纯色背景生成，再复制到工作区或 `tmp/imagegen/`，运行内置去背脚本，并验证 alpha 效果。如果这个流程不适合或失败，再问用户是否切换到 CLI `gpt-image-1.5`。
12. 检查输出是否满足主体、风格、构图、文字准确性，以及 invariants / avoid 要求。
13. 迭代时每次只针对一个明确问题做修改，再重新检查。
14. 仅用于预览的图，可以直接内联展示；底层文件可以保留在默认 `$CODEX_HOME/generated_images/...`。
15. 项目正式使用的图，必须移动或复制进工作区，并更新引用。不要让项目引用的最终素材只留在 `$CODEX_HOME/generated_images/...`。
16. 对批量/多素材请求，除非用户明确说只要预览，否则每一项最终交付都要保存到工作区。
17. 如果用户明确选择或确认 CLI fallback，再去读取 CLI 专用文档，处理 model、quality、size、`input_fidelity`、mask、输出格式、输出路径与网络设置。
18. 最终必须报告：保存路径、最终 prompt（或 prompt 集）、以及使用的是内建工具还是 CLI 回退模式。

## ????????

透明背景请求也先走内建 `image_gen`。因为内建工具没有真正的透明背景参数，所以默认做法是生成一个纯色可去除的色键背景图，再在本地转为 alpha。

默认流程：
1. 用内建 `image_gen` 生成目标主体，并放在纯色 chroma-key 背景上。
2. 选择不太可能出现在主体里的背景色：默认 `#00ff00`；如果主体是绿色，则改用 `#ff00ff`；蓝色主体时避免 `#0000ff`。
3. 生成后，把选中的源图从 `$CODEX_HOME/generated_images/...` 移到或复制到工作区或 `tmp/imagegen/`。
4. 运行内置去背脚本：
   ```bash
   python "${CODEX_HOME:-$HOME/.codex}/skills/.system/imagegen/scripts/remove_chroma_key.py" \
     --input <source> \
     --out <final.png> \
     --auto-key border \
     --soft-matte \
     --transparent-threshold 12 \
     --opaque-threshold 220 \
     --despill
   ```
5. 验证输出是否具有 alpha 通道、角落是否透明、主体是否完整、边缘是否没有明显的色键残边。如果还有轻微边缘问题，可重试一次 `--edge-contract 1`；只有在边缘明显锯齿且主体不是高反光材质时，才考虑加 `--edge-feather 0.25`。
6. 如果是项目正式素材，最终的透明 PNG/WebP 必须保存到项目里。

透明图 prompt 建议这样写：

```text
Create the requested subject on a perfectly flat solid #00ff00 chroma-key background for background removal.
The background must be one uniform color with no shadows, gradients, texture, reflections, floor plane, or lighting variation.
Keep the subject fully separated from the background with crisp edges and generous padding.
Do not use #00ff00 anywhere in the subject.
No cast shadow, no contact shadow, no reflection, no watermark, and no text unless explicitly requested.
```

不要自动切换到 CLI `gpt-image-1.5 --background transparent --output-format png`。只有以下情况才先询问用户：
- 用户明确要求原生透明
- 本地去背验证失败
- 图像本身过于复杂，例如头发、毛发、羽毛、烟雾、玻璃、液体、半透明材质、反射物体、柔和阴影、真实产品落地阴影，或主体颜色和所有实用色键都冲突

可以用类似下面这段简短确认：

```text
This likely needs true native transparency. The default built-in path uses a chroma-key background plus local removal, but true transparency requires the CLI fallback with gpt-image-1.5 because gpt-image-2 does not support background=transparent. It also requires OPENAI_API_KEY. Should I proceed with that CLI fallback?
```

## Prompt 增强

把用户的自然语言需求整理成结构化、可执行、适合生产的图片规格说明。可以让目标更清晰，但不要盲目添油加醋。

### ?????

- 如果用户 prompt 已经非常具体，就保留其具体性，只做归整与结构化。
- 如果 prompt 比较泛，可以做适度增强，但必须是为了显著提高结果质量。

允许的增强：
- 构图与取景提示
- 打磨程度与用途提示
- 实用的版式建议
- 对场景做适度具体化，前提是能服务于用户原始需求

不允许的增强：
- 凭空加角色或道具
- 擅自加入品牌、标语、配色或叙事设定
- 无依据地指定左右摆位等细节

## 用例分类（固定 slug）

请把请求归类到以下固定 slug 中，并在 prompt 与参考中保持一致。

????
- `photorealistic-natural`
- `product-mockup`
- `ui-mockup`
- `infographic-diagram`
- `scientific-educational`
- `ads-marketing`
- `productivity-visual`
- `logo-brand`
- `illustration-story`
- `stylized-concept`
- `historical-scene`

Edit:
- `text-localization`
- `identity-preserve`
- `precise-object-edit`
- `lighting-weather`
- `background-extraction`
- `style-transfer`
- `compositing`
- `sketch-to-render`

## Shared prompt schema

对两种顶层模式都使用如下的结构化 prompt 框架：

```text
Use case: <taxonomy slug>
Asset type: <where the asset will be used>
Primary request: <user's main prompt>
Input images: <Image 1: role; Image 2: role> (optional)
Scene/backdrop: <environment>
Subject: <main subject>
Style/medium: <photo/illustration/3D/etc>
Composition/framing: <wide/close/top-down; placement>
Lighting/mood: <lighting + mood>
Color palette: <palette notes>
Materials/textures: <surface details>
Text (verbatim): "<exact text>"
Constraints: <must keep/must avoid>
Avoid: <negative constraints>
```

说明：
- `Asset type` 与 `Input images` 只是 prompt 脚手架，不是 CLI 参数。
- `Scene/backdrop` 指画面中的视觉环境，不等于 CLI 回退路径里的 `background` 参数。
- 像 `Quality:`、`Input fidelity:`、mask、输出格式和输出路径这类执行参数，只属于 CLI 路径。

## Prompting best practices

- prompt 的组织顺序建议为：场景/背景 -> 主体 -> 细节 -> 约束
- 写清用途（广告图、UI mock、信息图等），帮助模型进入正确模式
- 做写实时，用相机/构图语言
- 用户没明确要矢量时，不要轻易用 SVG/矢量替代位图
- 对所有需要准确出现的文字，必须逐字引用
- 多图输入时，用索引区分每张图的用途
- 对编辑任务，每次迭代都重复 invariants，减少漂移
- 每次迭代只改一个明确问题

## Fallback CLI mode only

以下内容只适用于 CLI 回退模式：

- `tmp/imagegen/` 用于中间文件，例如 JSONL batch；完成后删除
- 最终交付建议写到 `output/imagegen/`
- 用 `--out` 或 `--out-dir` 控制输出路径

依赖：

```bash
uv pip install openai
uv pip install pillow
```

环境：
- CLI 实时调用需要设置 `OPENAI_API_KEY`
- 使用内建 `image_gen` 时不要向用户索要 API Key
- 不要让用户把完整 key 粘贴到聊天里，应让他们在本机环境变量中设置好

## Reference map

- `references/prompting.md`：两种模式共享的 prompt 原则
- `references/sample-prompts.md`：两种模式共享的可复制 prompt 模板
- `references/cli.md`：CLI 用法
- `references/image-api.md`：CLI/API 参数说明
- `references/codex-network.md`：CLI 模式下的网络/沙箱排障
- `scripts/image_gen.py`：CLI 实现。除非用户明确选择 CLI 或明确确认透明背景 CLI 回退，否则不要加载或使用它
- `$CODEX_HOME/skills/.system/imagegen/scripts/remove_chroma_key.py`：内建透明背景流程的本地后处理脚本
