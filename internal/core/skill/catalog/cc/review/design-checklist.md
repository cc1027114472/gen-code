# Design Review Checklist（Lite）

只有当 diff 触及前端文件时才运行这个检查表，否则静默跳过。

## 输出格式

```text
Design Review: N issues (X auto-fixable, Y need input, Z possible)
```

没有问题就写：

```text
Design Review: No issues found.
```

## 核心检查项

### AI Slop 检测

- 大量紫色 / 蓝紫渐变
- 三栏对称特征卡片，图标圆底 + 标题 + 两行描述
- 大面积 `text-align: center`
- 几乎所有元素都用同一个大圆角
- 通用 hero 套话文案

### Typography

- 正文小于 16px
- 新增字体族超过 3 个
- 标题层级跳跃
- 使用 Papyrus、Comic Sans、Impact 等黑名单字体

### Spacing / Layout

- 固定宽度却没有响应式兜底
- 正文容器没有 `max-width`
- 新增 `!important`

### Interaction States

- 交互元素没有 hover / focus 状态
- `outline: none` 却没有替代焦点样式
- 触控目标可能小于 44px

## DESIGN.md 约束

如果项目根目录存在 `DESIGN.md` 或 `design-system.md`，优先按该文件约束颜色、字体和 spacing scale，不要拿通用规则硬压项目既有设计系统。
