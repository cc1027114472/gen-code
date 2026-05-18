---
name: plugin-creator
description: 用于为 Codex 创建并脚手架化插件目录，包括必需的 `.codex-plugin/plugin.json`、可选的插件目录/文件，以及可在发布或测试前再补全的基础占位内容。当 Codex 需要创建新的本地插件、补齐可选插件结构，或生成/更新仓库根目录 `.agents/plugins/marketplace.json` 中的插件排序与可用性元数据时使用。
---

# Plugin Creator

## Quick Start

1. 运行脚手架脚本：

```bash
  # 插件名会被规范化为小写 kebab-case，且长度必须 <= 64 个字符。
  # 生成的目录名与 plugin.json 中的 name 始终保持一致。
# 请在仓库根目录执行（或把 .agents/... 替换为这个 SKILL 的绝对路径）。
# 默认会创建到 <repo_root>/plugins/<plugin-name>。
python3 .agents/skills/plugin-creator/scripts/create_basic_plugin.py <plugin-name>
```

2. 打开 `<plugin-path>/.codex-plugin/plugin.json`，将其中的 `[TODO: ...]` 占位内容替换掉。

3. 如果插件需要出现在 Codex UI 的 marketplace 排序中，则生成或更新对应的仓库 marketplace 条目：

```bash
# marketplace.json 始终位于 <repo-root>/.agents/plugins/marketplace.json
python3 .agents/skills/plugin-creator/scripts/create_basic_plugin.py my-plugin --with-marketplace
```

对于 home-local 插件，把 `<home>` 视为根目录，并这样使用：

```bash
python3 .agents/skills/plugin-creator/scripts/create_basic_plugin.py my-plugin \
  --path ~/plugins \
  --marketplace-path ~/.agents/plugins/marketplace.json \
  --with-marketplace
```

4. 按需生成或补齐可选的配套目录：

```bash
python3 .agents/skills/plugin-creator/scripts/create_basic_plugin.py my-plugin --path <parent-plugin-directory> \
  --with-skills --with-hooks --with-scripts --with-assets --with-mcp --with-apps --with-marketplace
```

`<parent-plugin-directory>` 指的是插件目录 `<plugin-name>` 将被创建到的父目录，例如 `~/code/plugins`。

## What this skill creates

- 如果用户没有明确插件位置，且要生成 marketplace 条目，先询问他们是要 repo-local 插件还是 home-local 插件。
- 创建插件根目录：`/<parent-plugin-directory>/<plugin-name>/`。
- 始终创建 `/<parent-plugin-directory>/<plugin-name>/.codex-plugin/plugin.json`。
- 在 manifest 中生成完整的 schema 结构、占位值，以及完整的 `interface` 段。
- 当设置了 `--with-marketplace` 时，创建或更新 `<repo-root>/.agents/plugins/marketplace.json`。
  - 如果 marketplace 文件还不存在，则先写入顶层 `name` 与 `interface.displayName` 占位字段，再加入第一个插件条目。
- `<plugin-name>` 使用与 skill-creator 相同的命名规范进行标准化：
  - `My Plugin` -> `my-plugin`
  - `My--Plugin` -> `my-plugin`
  - 下划线、空格与标点都会转换为 `-`
  - 结果始终是小写、以连字符分隔，并自动合并连续的连字符
- 支持按需创建以下可选内容：
  - `skills/`
  - `hooks/`
  - `scripts/`
  - `assets/`
  - `.mcp.json`
  - `.app.json`

## Marketplace workflow

- `marketplace.json` 始终位于 `<repo-root>/.agents/plugins/marketplace.json`。
- 对于 home-local 插件，也采用相同约定，只是把 `<home>` 作为根：
  `~/.agents/plugins/marketplace.json`，并配合 `./plugins/<plugin-name>`。
- marketplace 根元数据支持顶层 `name`，以及可选的 `interface.displayName`。
- 将 `plugins[]` 数组中的顺序视为 Codex 中的显示顺序。除非用户明确要求重排，否则新增条目一律追加到末尾。
- `displayName` 属于 marketplace 的 `interface` 对象，不属于单个 `plugins[]` 条目。
- 每一个生成出来的 marketplace 条目都必须包含：
  - `policy.installation`
  - `policy.authentication`
  - `category`
- 新条目的默认值为：
  - `policy.installation: "AVAILABLE"`
  - `policy.authentication: "ON_INSTALL"`
- 只有当用户明确指定其他允许值时，才覆盖默认值。
- `policy.installation` 的允许值：
  - `NOT_AVAILABLE`
  - `AVAILABLE`
  - `INSTALLED_BY_DEFAULT`
- `policy.authentication` 的允许值：
  - `ON_INSTALL`
  - `ON_USE`
- 将 `policy.products` 视为显式覆盖项。除非用户明确要求产品级限制，否则不要添加。
- 生成的 marketplace 条目结构如下：

```json
{
  "name": "plugin-name",
  "source": {
    "source": "local",
    "path": "./plugins/plugin-name"
  },
  "policy": {
    "installation": "AVAILABLE",
    "authentication": "ON_INSTALL"
  },
  "category": "Productivity"
}
```

- 只有在你明确打算替换同名旧 marketplace 条目时，才使用 `--force`。
- 如果 `<repo-root>/.agents/plugins/marketplace.json` 尚不存在，则创建包含顶层 `"name"`、带 `"displayName"` 的 `"interface"` 对象，以及 `plugins` 数组的完整结构，然后再添加新条目。

- 对于全新的 marketplace 文件，其根对象应如下所示：

```json
{
  "name": "[TODO: marketplace-name]",
  "interface": {
    "displayName": "[TODO: Marketplace Display Name]"
  },
  "plugins": [
    {
      "name": "plugin-name",
      "source": {
        "source": "local",
        "path": "./plugins/plugin-name"
      },
      "policy": {
        "installation": "AVAILABLE",
        "authentication": "ON_INSTALL"
      },
      "category": "Productivity"
    }
  ]
}
```

## Required behavior

- 外层目录名与 `plugin.json` 中的 `"name"` 必须始终使用同一个已标准化的插件名。
- 不要移除必需结构；`.codex-plugin/plugin.json` 必须保留。
- 在有人明确填写前，manifest 中的值保持为占位内容。
- 如果要在一个已经存在的插件路径里创建文件，只有在确实要覆盖时才使用 `--force`。
- 保留已有 marketplace 中的 `interface.displayName`。
- 生成 marketplace 条目时，即使值是默认值，也必须写出 `policy.installation`、`policy.authentication` 与 `category`。
- 只有当用户明确要求时，才加入 `policy.products`。
- marketplace 中的 `source.path` 必须保持为相对仓库根目录的 `./plugins/<plugin-name>`。

## Reference to exact spec sample

如果需要 manifest 与 marketplace 条目的权威 JSON 示例，请使用：

- `references/plugin-json-spec.md`

## Validation

编辑完 `SKILL.md` 后，运行：

```bash
python3 <path-to-skill-creator>/scripts/quick_validate.py .agents/skills/plugin-creator
```
