---
name: codex-pr-body
description: Update the title and body of one or more pull requests.
---

## 确定 Pull Request

调用此 skill 时，要更新的 PR 可能会被显式指定，但在常见情况下，需要从用户当前正在处理的分支 / 提交中推断出要更新的 PR。对于普通的 Git 用法（即不是下文讨论的 Sapling），你可能需要组合使用 `git branch` 和 `gh pr view <branch> --repo openai/codex --json number --jq '.number'` 来确定与当前分支 / 提交关联的 PR。

## Pull Request 正文内容

调用时，使用 `gh` 编辑 pull request 的正文和标题，使之反映指定 PR 的内容。务必检查现有的 pull request 正文，看看是否有需要保留的关键信息。例如，绝不要删除现有 pull request 正文中的图片，因为如果你删掉它，作者可能没有办法恢复。

解释为什么要进行此更改至关重要。如果调用此 skill 的当前对话中已经讨论过动机，务必在 pull request 正文中体现出来。

正文也应该解释改了什么，但这部分应放在为什么之后。

将讨论限制在该提交的净变化上。通常不建议讨论那些在 pull request 开发过程中曾尝试但后来又撤销的更改。在重写 pull request 正文时，当这些细节对未来读者不再合适 / 不再有价值时，你可能需要将其删除。

避免引用我本地磁盘上的绝对路径。当谈到仓库内的路径时，只需使用仓库相对路径。

通常说明该更改是如何被验证的是有帮助的。不过，没有必要提及那些 CI 会自动检查的事项，例如，不要把“运行了 `just fmt`”写进测试计划。但指出为了验证 pull request 引入的新行为而专门新增的测试，通常是合适的。

使用 Markdown 以专业方式格式化 pull request。确保在行内引用时，“代码相关的内容”使用单反引号包裹。引用代码或展示 shell 记录时，围栏代码块很有用。此外，在引用与该变更相关的现有代码片段时，使用 GitHub 永久链接。

务必引用任何相关的 pull request 或 issue，尽管没有必要在它自己的 PR 正文中引用该 pull request。

如果由于此更改，https://developers.openai.com/codex 上有文档需要更新，请在 pull request 末尾附近用一个单独章节注明。如果没有需要更新的文档，则省略此章节。

## 处理 Stack

有时一个 pull request 由一组相互构建于彼此之上的提交栈组成。在这些情况下，PR 正文应反映整个提交栈引入的净变化，而不是构成该栈的各个单独提交。

类似地，有时用户可能在使用 Sapling 之类的工具来处理 stacked pull requests，此时 PR 的 `base` 可能不是 `main`，而是该栈中另一个 PR 的 `head` 所在分支。在这种情况下，务必只讨论针对该 stacked base 打开的这个 PR 的 `base` 与 `head` 之间的净变化，而不是相对于 `main` 的变化。

## Sapling（堆叠分支工具）

如果存在 `.git/sl/store`，则说明这个 Git 仓库由 Sapling SCM (https://sapling-scm.com) 管理。

在 Sapling 中，运行以下命令以查看当前修订是否关联了一个 GitHub pull request：

```shell
sl log --template '{github_pull_request_url}' -r .
```

或者，你可以运行 `sl sl` 来查看当前开发分支，以及当前提交是否关联了一个 GitHub pull request。例如，如果输出是：

```
  @  cb032b31cf  72 minutes ago  mbolin  #11412
╭─╯  tui: show non-file layer content in /debug-config
│
o  fdd0cd1de9  Today at 20:09  origin/main
│
~
```

- `@` 表示当前提交是 `cb032b31cf`
- 这是一个开发分支，包含一个从 `origin/main` 分出的单独提交
- 它关联着 GitHub pull request #11412
