---
name: skill-installer
description: 用于把 Codex skill 安装到 `$CODEX_HOME/skills`，来源可以是精选列表，也可以是 GitHub 仓库路径。当用户想查看可安装的 skill、安装某个精选 skill，或从其他仓库安装 skill（包括私有仓库）时使用。
metadata:
  short-description: 从 openai/skills 或其他仓库安装精选 skills
---

# Skill 安装器

帮助安装 skills。默认来源是 `https://github.com/openai/skills/tree/main/skills/.curated`，但用户也可以提供其他位置。实验性 skill 位于 `https://github.com/openai/skills/tree/main/skills/.experimental`，安装方式相同。

根据任务使用这些辅助脚本：
- 当用户问“有哪些可用 skill”，或者调用这个 skill 却没明确说要做什么时，列出 skills。默认列出 `.curated`；如果用户想看实验性 skill，则传 `--path skills/.experimental`。
- 当用户提供 skill 名称时，从 curated 列表安装。
- 当用户提供 GitHub 仓库或路径（包括私有仓库）时，从对应仓库安装。

使用这些辅助脚本来安装 skill。

## ????

列出 skills 时，可按下面这种方式输出，具体以用户上下文为准。如果用户问的是实验性 skill，则从 `.experimental` 列表读取，并在文案里注明来源：
"""
来自 {repo} 的 skills：
1. skill-1（示例）
2. skill-2（已安装）
3. ...
你想安装哪几个？
"""

安装完成后，告诉用户："Restart Codex to pick up new skills."

## ??

这些脚本都需要网络；如果运行在沙箱里，执行时需要请求提权。

- `scripts/list-skills.py`（打印 skills 列表，并标注哪些已安装）
- `scripts/list-skills.py --format json`
- 示例（列出 experimental）：`scripts/list-skills.py --path skills/.experimental`
- `scripts/install-skill-from-github.py --repo <owner>/<repo> --path <path/to/skill> [<path/to/skill> ...]`
- `scripts/install-skill-from-github.py --url https://github.com/<owner>/<repo>/tree/<ref>/<path>`
- 示例（安装 experimental skill）：`scripts/install-skill-from-github.py --repo openai/skills --path skills/.experimental/<skill-name>`

## ?????

- 默认优先对公共 GitHub 仓库使用直接下载。
- 如果下载因认证或权限报错失败，则回退到 git sparse checkout。
- 如果目标 skill 目录已存在，则中止安装。
- 安装目标是 `$CODEX_HOME/skills/<skill-name>`（默认即 `~/.codex/skills`）。
- 多个 `--path` 会在一次执行中安装多个 skill；每个 skill 默认使用其路径 basename 作为名字，除非显式传了 `--name`。
- 可用选项：`--ref <ref>`（默认 `main`）、`--dest <path>`、`--method auto|download|git`。

## ??

- curated 列表通过 GitHub API 从 `https://github.com/openai/skills/tree/main/skills/.curated` 拉取。若不可用，应说明错误并退出。
- 私有 GitHub 仓库可通过现有 git 凭证访问，也可使用可选的 `GITHUB_TOKEN` / `GH_TOKEN` 进行下载。
- git 回退会先尝试 HTTPS，再尝试 SSH。
- `https://github.com/openai/skills/tree/main/skills/.system` 下的 skills 本来就是预装的，因此通常不需要帮助用户安装这些。如果用户问到，直接说明即可；若他们坚持，也可以下载后覆盖。
- 已安装标记来自 `$CODEX_HOME/skills`。
