---
name: "openai-docs"
description: "当用户询问如何基于 OpenAI 产品或 API 进行开发，并且需要带引用的最新官方文档、需要帮助为某个场景选择最新模型，或需要模型升级与提示词升级指导时使用。优先使用 OpenAI 文档 MCP 工具，仅将内置参考资料作为辅助上下文；若必须回退到网页检索，也只能使用 OpenAI 官方域名。"
---


# OpenAI 文档

通过 `developers.openai.com` MCP 服务器，提供权威且最新的 OpenAI 开发文档指导。凡是与 OpenAI 相关的问题，都应优先使用开发者文档 MCP 工具，而不是 `web.run`。这个 skill 同时负责模型选择、API 模型迁移，以及提示词升级指导。只有在 MCP 已安装但仍无法返回有效结果时，才回退到网页搜索。

## API Key 预检

如果用户的请求涉及构建、运行、配置、调试或实现基于 API 的应用、脚本、CLI、生成器或工具，且存在 `openai-platform-api-key`，请先使用它完成凭证前置检查。这个凭证关卡解决后，再回到这里查当前文档。

对于纯文档问答、引用、模型/API 指导、概念解释，以及不需要实际构建或运行 API 应用的示例，可以直接使用本 skill。

## ????

- 使用 `mcp__openaiDeveloperDocs__search_openai_docs` 查找最相关的文档页面。
- 使用 `mcp__openaiDeveloperDocs__fetch_openai_doc` 拉取精确章节，并准确引用或转述。
- 只有在没有明确查询词、需要浏览或发现页面时，才使用 `mcp__openaiDeveloperDocs__list_openai_docs`。
- 对于模型选择、“latest model” 或默认模型问题，先获取 `https://developers.openai.com/api/docs/guides/latest-model.md`。如果不可用，再读取 `references/latest-model.md`。
- 对于模型升级或提示词升级，仅当目标是 latest/current/default 或目标未明确时，才运行 `node scripts/resolve-latest-model-info.js`；否则保留用户明确指定的目标。
- 保留用户明确指定的目标请求：如果用户点名目标模型，比如“迁移到 GPT-5.4”，就沿用该目标，不要悄悄改成更新模型。若有更新的官方建议，只作为可选补充说明。
- 如果需要当前远端指引，直接抓取返回的迁移指南和 prompting 指南 URL。若直接抓取失败，再使用 MCP / 搜索回退；如果仍失败，则使用内置参考资料，并明确说明这是回退方案。

## OpenAI 产品快照

1. Apps SDK：通过提供 Web 组件 UI 与 MCP server 来构建 ChatGPT 应用，使你的工具可被 ChatGPT 调用。
2. Responses API：统一接口，适合有状态、多模态、可调用工具的 agent 工作流。
3. Chat Completions API：根据一组消息生成模型回复，这些消息共同构成一段对话。
4. Codex：OpenAI 的编码代理，可进行软件编写、理解、审查和调试。
5. gpt-oss：开放权重的 OpenAI 推理模型（gpt-oss-120b 和 gpt-oss-20b），以 Apache 2.0 协议发布。
6. Realtime API：用于构建低延迟、多模态体验，包括自然的语音到语音对话。
7. Agents SDK：构建 agent 应用的工具包，支持模型使用工具与上下文、代理交接、流式输出与完整 trace。

## ?? MCP ????

如果 MCP 工具失败，或没有任何 OpenAI 文档资源可用：

1. 先自行运行安装命令：`codex mcp add openaiDeveloperDocs --url https://developers.openai.com/mcp`
2. 如果因为权限或沙箱导致失败，立刻用提权方式重试同一命令，并附上一句简短的提权理由。不要先让用户自己运行。
3. 只有当提权重试仍然失败时，才让用户自行执行安装命令。
4. 提醒用户重启 Codex。
5. 重启后再次运行文档搜索/抓取。

## ???

1. 先判断请求属于哪一类：普通文档查询、模型选择、模型字符串升级、提示词升级，还是更广义的 API / provider 迁移。
2. 对于模型选择或升级请求，当用户要求 latest/current/default 指引时，优先用当前远端文档，而不是内置参考。
   - 获取 `https://developers.openai.com/api/docs/guides/latest-model.md`。
   - 找出最新模型 ID，以及显式给出的迁移或 prompting 指南链接。
   - 优先使用 latest-model 页面给出的显式链接，而不是自行猜测 URL。
   - 对于用户明确命名的模型目标，保留该目标，不要静默改到最新模型。若有更新的远端建议，只作为可选信息。
   - 对于 latest/current/default 这类动态升级请求，运行 `node scripts/resolve-latest-model-info.js`，然后尽可能直接抓取返回的两个指南 URL。
   - 若直接抓取指南失败，使用开发者文档 MCP 工具或 OpenAI 官方域名搜索来定位同一内容。
   - 若远端文档不可用，则使用内置回退参考，并明确说明。
3. 对于模型升级，保持改动尽可能窄：只在安全时更新当前使用的 OpenAI API 模型默认值及其直接相关的 prompts。
4. 不要改动历史文档、示例、评测基线、fixture、provider 对比、provider 注册表、价格表、alias 默认值、低成本回退路径，或含义不明确的旧模型用法，除非用户明确要求升级它们。
5. 不要把 SDK、工具链、IDE、插件、shell、认证或 provider 环境迁移夹带在模型与 prompt 升级里一起做。
6. 如果升级会牵涉到 API 表面变更、schema 重连、tool handler 变更，或超出简单模型字符串替换与 prompt 编辑的实现工作，就把它报告为 blocked 或需要确认。
7. 对于普通文档查询，用精准 query 搜索文档，抓取最相关页面和所需的精确章节，并带简明引用给出答案。

## ????

只读取必要内容：

- `https://developers.openai.com/api/docs/guides/latest-model.md` -> 当前模型选择，以及“best/latest/current model”类问题。
- `references/latest-model.md` -> 模型选择与“best/latest/current model”问题的内置回退参考。
- `references/upgrade-guide.md` -> 模型升级与升级规划请求的内置回退参考。
- `references/prompting-guide.md` -> prompt 重写与 prompt 行为升级的内置回退参考。

## ????

- 将 OpenAI 文档视为事实来源，避免猜测。
- 保持迁移改动范围小，尽量不改变原有行为。
- 优先考虑只升级 prompt。
- 不要杜撰价格、可用性、参数、API 变更或 breaking changes。
- 引用尽量短，符合引用限制；优先转述并附上来源。
- 如果多个页面说法不一致，要指出差异并同时引用。
- 如果官方文档与代码库现状冲突，要明确说明冲突，并在做大范围修改前停下。
- 如果文档没有覆盖用户需求，也要直接说，并给出后续建议。

## ????

- 只要问题与 OpenAI 相关，就始终先使用 MCP 文档工具，再考虑任何网页搜索。
- 如果 MCP 已安装但没有返回有效结果，才允许回退到网页搜索。
- 回退到网页搜索时，必须限制在 OpenAI 官方域名（`developers.openai.com`、`platform.openai.com`）内，并在回答里附上来源。
