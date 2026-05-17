---
name: finishing-a-development-branch
description: Use when implementation is complete, all tests pass, and you need to decide how to integrate the work - guides completion of development work by presenting structured options for merge, PR, or cleanup
---

# 完成一个开发分支

## 概览

通过呈现清晰选项并处理所选工作流，来完成开发工作。

**核心原则：** 验证测试 -> 呈现选项 -> 执行选择 -> 清理。

**开始时要声明：** “我正在使用 `finishing-a-development-branch` skill 来完成这项工作。”

## 这个流程

### 步骤 1：验证测试

**在呈现选项之前，先验证测试通过：**

```bash
# 运行项目的测试套件
npm test / cargo test / pytest / go test ./...
```

**如果测试失败：**
```
测试失败（<N> 个失败）。必须先修复，才能完成：

[展示失败]

在测试通过之前，不能继续进行合并/PR。
```

停止。不要继续到步骤 2。

**如果测试通过：** 继续到步骤 2。

### 步骤 2：确定基准分支

```bash
# 尝试常见的基准分支
git merge-base HEAD main 2>/dev/null || git merge-base HEAD master 2>/dev/null
```

或者提问： “这个分支是从 `main` 分出来的 - 对吗？”

### 步骤 3：呈现选项

准确地呈现下面这 4 个选项：

```
实现已完成。你想怎么做？

1. 在本地合并回 <base-branch>
2. 推送并创建 Pull Request
3. 保持这个分支不动（我之后再处理）
4. 丢弃这项工作

选择哪一个？
```

**不要添加解释** - 让选项保持简洁。

### 步骤 4：执行选择

#### 选项 1：在本地合并

```bash
# 切换到基准分支
git checkout <base-branch>

# 拉取最新
git pull

# 合并功能分支
git merge <feature-branch>

# 在合并结果上验证测试
<test command>

# 如果测试通过
git branch -d <feature-branch>
```

然后：清理工作树（步骤 5）

#### 选项 2：推送并创建 PR

```bash
# 推送分支
git push -u origin <feature-branch>

# 创建 PR
gh pr create --title "<title>" --body "$(cat <<'EOF'
## 摘要
<2-3 条改动摘要>

## 测试计划
- [ ] <验证步骤>
EOF
)"
```

然后：清理工作树（步骤 5）

#### 选项 3：保持原样

汇报： “保持分支 `<name>` 不变。工作树保留在 `<path>`。”

**不要清理工作树。**

#### 选项 4：丢弃

**先确认：**
```
这将永久删除：
- 分支 <name>
- 所有提交：<commit-list>
- 位于 <path> 的工作树

请输入 'discard' 以确认。
```

等待精确确认。

如果确认：
```bash
git checkout <base-branch>
git branch -D <feature-branch>
```

然后：清理工作树（步骤 5）

### 步骤 5：清理工作树

**适用于选项 1、2、4：**

检查自己是否位于工作树中：
```bash
git worktree list | grep $(git branch --show-current)
```

如果是：
```bash
git worktree remove <worktree-path>
```

**对于选项 3：** 保留工作树。

## 快速参考

| 选项 | 合并 | 推送 | 保留工作树 | 清理分支 |
|--------|-------|------|---------------|----------------|
| 1. 本地合并 | 是 | - | - | 是 |
| 2. 创建 PR | - | 是 | 是 | - |
| 3. 保持原样 | - | - | 是 | - |
| 4. 丢弃 | - | - | - | 是（强制） |

## 常见错误

**跳过测试验证**
- **问题：** 合并损坏的代码，创建失败的 PR
- **修复：** 在给出选项之前，始终先验证测试

**开放式问题**
- **问题：** “接下来我该做什么？” -> 含糊不清
- **修复：** 准确地给出 4 个结构化选项

**自动清理工作树**
- **问题：** 在可能还需要它时就移除工作树（选项 2、3）
- **修复：** 只对选项 1 和 4 进行清理

**丢弃前没有确认**
- **问题：** 不小心删除工作
- **修复：** 要求输入 `discard` 进行确认

## 危险信号

**绝不要：**
- 在测试失败时继续
- 不验证合并结果中的测试就进行合并
- 未经确认就删除工作
- 没有明确请求就强推

**始终要：**
- 在给出选项之前验证测试
- 准确地呈现 4 个选项
- 对选项 4 获取输入式确认
- 只对选项 1 和 4 清理工作树

## 集成

**由以下 skill 调用：**
- **`subagent-driven-development`**（步骤 7）- 所有任务完成之后
- **`executing-plans`**（步骤 5）- 所有批次完成之后

**搭配：**
- **`using-git-worktrees`** - 清理由那个 skill 创建的工作树
