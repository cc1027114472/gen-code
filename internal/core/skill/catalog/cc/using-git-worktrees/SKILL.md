---
name: using-git-worktrees
description: 在开始需要与当前工作区隔离的功能开发时，或在执行实现计划之前使用——创建带有智能目录选择与安全校验的隔离 git worktree
---

# 使用 Git Worktrees

## 概述

Git worktree 会创建共享同一仓库的隔离工作区，让你可以同时在多个分支上工作，而无需来回切换。

**核心原则：** 系统化目录选择 + 安全校验 = 可靠隔离。

**开始时要声明：** “我正在使用 using-git-worktrees skill 来建立一个隔离工作区。”

## 目录选择流程

遵循以下优先级顺序：

### 1. 检查已有目录

```bash
# 按优先级顺序检查
ls -d .worktrees 2>/dev/null     # 首选（隐藏目录）
ls -d worktrees 2>/dev/null      # 备选
```

**如果找到了：** 使用该目录。如果两个都存在，`.worktrees` 优先。

### 2. 检查 `CLAUDE.md`

```bash
grep -i "worktree.*director" CLAUDE.md 2>/dev/null
```

**如果指定了偏好：** 直接使用，不需要询问。

### 3. 询问用户

如果不存在目录，并且 `CLAUDE.md` 里也没有偏好：

```
没有找到 worktree 目录。你希望我把 worktree 创建在哪里？

1. .worktrees/（项目本地，隐藏）
2. ~/.config/superpowers/worktrees/<project-name>/（全局位置）

你更倾向于哪一个？
```

## 安全校验

### 针对项目本地目录（`.worktrees` 或 `worktrees`）

**在创建 worktree 之前，必须验证该目录已被忽略：**

```bash
# 检查目录是否被忽略（遵循本地、全局和系统 gitignore）
git check-ignore -q .worktrees 2>/dev/null || git check-ignore -q worktrees 2>/dev/null
```

**如果没有被忽略：**

按 Jesse 的规则 “Fix broken things immediately”：
1. 在 `.gitignore` 中加入合适的一行
2. 提交该变更
3. 继续创建 worktree

**为什么这很关键：** 它能防止你不小心把 worktree 内容提交进仓库。

### 针对全局目录（`~/.config/superpowers/worktrees`）

不需要做 `.gitignore` 校验——因为它完全在项目外部。

## 创建步骤

### 1. 检测项目名

```bash
project=$(basename "$(git rev-parse --show-toplevel)")
```

### 2. 创建 Worktree

```bash
# 确定完整路径
case $LOCATION in
  .worktrees|worktrees)
    path="$LOCATION/$BRANCH_NAME"
    ;;
  ~/.config/superpowers/worktrees/*)
    path="~/.config/superpowers/worktrees/$project/$BRANCH_NAME"
    ;;
esac

# 用新分支创建 worktree
git worktree add "$path" -b "$BRANCH_NAME"
cd "$path"
```

### 3. 运行项目初始化

自动检测并运行合适的初始化：

```bash
# Node.js
if [ -f package.json ]; then npm install; fi

# Rust
if [ -f Cargo.toml ]; then cargo build; fi

# Python
if [ -f requirements.txt ]; then pip install -r requirements.txt; fi
if [ -f pyproject.toml ]; then poetry install; fi

# Go
if [ -f go.mod ]; then go mod download; fi
```

### 4. 验证干净基线

运行测试，确保 worktree 起始状态是干净的：

```bash
# 示例——使用适合项目的命令
npm test
cargo test
pytest
go test ./...
```

**如果测试失败：** 报告失败项，并询问是继续还是先调查。

**如果测试通过：** 报告已准备就绪。

### 5. 报告位置

```
Worktree 已准备好，位置：<full-path>
测试通过（<N> 个测试，0 个失败）
可以开始实现 <feature-name>
```

## 速查表

| 情况 | 操作 |
|-----------|--------|
| 存在 `.worktrees/` | 使用它（验证已忽略） |
| 存在 `worktrees/` | 使用它（验证已忽略） |
| 两者都存在 | 使用 `.worktrees/` |
| 两者都不存在 | 检查 `CLAUDE.md` → 询问用户 |
| 目录未被忽略 | 加入 `.gitignore` + 提交 |
| 基线测试失败 | 报告失败 + 询问 |
| 没有 `package.json`/`Cargo.toml` | 跳过依赖安装 |

## 常见错误

### 跳过 ignore 校验

- **问题：** worktree 内容会被 git 跟踪，污染 `git status`
- **修复：** 在创建项目本地 worktree 之前，始终使用 `git check-ignore`

### 想当然地决定目录位置

- **问题：** 会制造不一致，并违背项目约定
- **修复：** 遵循优先级：已有目录 > `CLAUDE.md` > 询问

### 在测试失败时继续推进

- **问题：** 你无法区分新 bug 和原有问题
- **修复：** 报告失败，并在继续前取得明确许可

### 硬编码初始化命令

- **问题：** 在使用不同工具的项目中会失效
- **修复：** 根据项目文件自动检测（`package.json` 等）

## 示例工作流

```
你：我正在使用 using-git-worktrees skill 来建立一个隔离工作区。

[检查 .worktrees/ - 存在]
[验证已忽略 - git check-ignore 确认 .worktrees/ 已被忽略]
[创建 worktree：git worktree add .worktrees/auth -b feature/auth]
[运行 npm install]
[运行 npm test - 47 个通过]

Worktree 已准备好，位置：/Users/jesse/myproject/.worktrees/auth
测试通过（47 个测试，0 个失败）
可以开始实现 auth 功能
```

## 危险信号

**永远不要：**
- 在未验证目录已被忽略时创建 worktree（项目本地目录）
- 跳过基线测试验证
- 在测试失败时不经询问就继续
- 在目录位置不明确时自行假设
- 跳过 `CLAUDE.md` 检查

**始终要：**
- 遵循目录优先级：已有目录 > `CLAUDE.md` > 询问
- 对项目本地目录验证其已被忽略
- 自动检测并运行项目初始化
- 验证干净的测试基线

## 集成

**由以下流程调用：**
- **brainstorming**（第 4 阶段）——当设计已批准并要进入实现时，必须使用
- **subagent-driven-development**——在执行任何任务之前必须使用
- **executing-plans**——在执行任何任务之前必须使用
- 任何需要隔离工作区的 skill

**搭配使用：**
- **finishing-a-development-branch**——工作完成后收尾清理时必须使用
