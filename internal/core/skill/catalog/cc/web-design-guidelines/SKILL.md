---
name: web-design-guidelines
description: 审查 UI 代码是否符合 Web Interface Guidelines。适用于用户提出“review my UI”“check accessibility”“audit design”“review UX”或“check my site against best practices”等请求时。
metadata:
  author: vercel
  version: "1.0.0"
  argument-hint: <file-or-pattern>
---

# Web 界面设计指南

审查文件是否符合 Web Interface Guidelines。

## 工作方式

1. 从下面的来源 URL 拉取最新指南
2. 读取指定文件（或提示用户提供文件/模式）
3. 对照拉取到的指南中的全部规则进行检查
4. 以精简的 `file:line` 格式输出发现的问题

## 指南来源

每次审查前都获取最新指南：

```
https://raw.githubusercontent.com/vercel-labs/web-interface-guidelines/main/command.md
```

使用 WebFetch 拉取最新规则。拉取到的内容中包含全部规则以及输出格式说明。

## 用法

当用户提供文件或模式参数时：
1. 从上面的来源 URL 拉取指南
2. 读取指定文件
3. 应用拉取到的指南中的全部规则
4. 使用指南要求的格式输出发现的问题

如果没有指定文件，则询问用户要审查哪些文件。
