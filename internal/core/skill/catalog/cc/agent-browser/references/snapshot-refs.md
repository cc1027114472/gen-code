# 快照与 Refs

为 AI agents 提供紧凑元素引用，大幅降低上下文消耗。

**相关内容：** 完整命令参考见 [commands.md](commands.md)，快速开始见 [SKILL.md](../SKILL.md)。

## 目录

- [Refs 如何工作](#refs-如何工作)
- [Snapshot 命令](#snapshot-命令)
- [如何使用 Refs](#如何使用-refs)
- [Ref 生命周期](#ref-生命周期)
- [最佳实践](#最佳实践)
- [Ref 记号细节](#ref-记号细节)
- [故障排查](#故障排查)

## Refs 如何工作

传统方式：

```
完整 DOM/HTML → AI 解析 → CSS selector → 执行动作（约 3000-5000 tokens）
```

agent-browser 方式：

```
紧凑快照 → 分配 @refs → 直接交互（约 200-400 tokens）
```

## Snapshot 命令

```bash
# 基础快照（显示页面结构）
agent-browser snapshot

# 交互快照（-i 标志）- 推荐
agent-browser snapshot -i
```

### 快照输出格式

```
Page: Example Site - Home
URL: https://example.com

@e1 [header]
  @e2 [nav]
    @e3 [a] "Home"
    @e4 [a] "Products"
    @e5 [a] "About"
  @e6 [button] "Sign In"

@e7 [main]
  @e8 [h1] "Welcome"
  @e9 [form]
    @e10 [input type="email"] placeholder="Email"
    @e11 [input type="password"] placeholder="Password"
    @e12 [button type="submit"] "Log In"

@e13 [footer]
  @e14 [a] "Privacy Policy"
```

## 如何使用 Refs

拿到 refs 之后，就可以直接交互：

```bash
# 点击 "Sign In" 按钮
agent-browser click @e6

# 填写邮箱输入框
agent-browser fill @e10 "user@example.com"

# 填写密码
agent-browser fill @e11 "password123"

# 提交表单
agent-browser click @e12
```

## Ref 生命周期

**重要：** 页面发生变化时，refs 会失效！

```bash
# 获取初始快照
agent-browser snapshot -i
# @e1 [button] "Next"

# 点击会触发页面变化
agent-browser click @e1

# 必须重新 snapshot 才能获得新的 refs！
agent-browser snapshot -i
# @e1 [h1] "Page 2"  ← 现在它已经是另一个元素了！
```

## 最佳实践

### 1. 交互前始终先做快照

```bash
# 正确
agent-browser open https://example.com
agent-browser snapshot -i          # 先拿到 refs
agent-browser click @e1            # 再使用 ref

# 错误
agent-browser open https://example.com
agent-browser click @e1            # ref 还不存在！
```

### 2. 导航后重新做快照

```bash
agent-browser click @e5            # 导航到新页面
agent-browser snapshot -i          # 获取新的 refs
agent-browser click @e1            # 使用新的 refs
```

### 3. 动态变化后重新做快照

```bash
agent-browser click @e1            # 打开下拉框
agent-browser snapshot -i          # 查看下拉项
agent-browser click @e7            # 选择项目
```

### 4. 对特定区域做快照

对于复杂页面，可对特定区域做快照：

```bash
# 只对表单做快照
agent-browser snapshot @e9
```

## Ref 记号细节

```
@e1 [tag type="value"] "text content" placeholder="hint"
│    │   │             │               │
│    │   │             │               └─ 额外属性
│    │   │             └─ 可见文本
│    │   └─ 展示的关键属性
│    └─ HTML 标签名
└─ 唯一 ref ID
```

### 常见模式

```
@e1 [button] "Submit"                    # 带文本的按钮
@e2 [input type="email"]                 # 邮箱输入框
@e3 [input type="password"]              # 密码输入框
@e4 [a href="/page"] "Link Text"         # 锚点链接
@e5 [select]                             # 下拉框
@e6 [textarea] placeholder="Message"     # 文本域
@e7 [div class="modal"]                  # 容器（在相关时显示）
@e8 [img alt="Logo"]                     # 图片
@e9 [checkbox] checked                   # 已勾选复选框
@e10 [radio] selected                    # 已选中单选按钮
```

## Iframes

快照会自动检测并内联 iframe 内容。当主 frame 执行 snapshot 时，每个 `Iframe` 节点都会被解析，其子无障碍树会直接插入到输出中。分配给 iframe 内元素的 refs 会携带 frame 上下文，因此像 `click`、`fill`、`type` 这样的交互无需手动切换 frame 即可使用。

```bash
agent-browser snapshot -i
# @e1 [heading] "Checkout"
# @e2 [Iframe] "payment-frame"
#   @e3 [input] "Card number"
#   @e4 [input] "Expiry"
#   @e5 [button] "Pay"
# @e6 [button] "Cancel"

# 直接使用 refs 与 iframe 内元素交互
agent-browser fill @e3 "4111111111111111"
agent-browser fill @e4 "12/28"
agent-browser click @e5
```

**关键细节：**

- 仅展开一层 iframe 嵌套（iframe 中的 iframe 不会继续递归）
- 如果跨域 iframe 阻止访问无障碍树，则会被静默跳过
- 空 iframe 或不含交互内容的 iframe 不会出现在输出中
- 若要仅针对单个 iframe 做快照，可使用 `frame @ref` 后再执行 `snapshot -i`

## 故障排查

### “Ref not found” 错误

```bash
# ref 可能已经变化，重新做快照
agent-browser snapshot -i
```

### 元素未出现在快照中

```bash
# 先向下滚动，让元素进入视图
agent-browser scroll down 1000
agent-browser snapshot -i

# 或等待动态内容加载
agent-browser wait 1000
agent-browser snapshot -i
```

### 元素太多

```bash
# 对特定容器做快照
agent-browser snapshot @e5

# 或对只需内容提取的场景改用 get text
agent-browser get text @e5
```
