#!/bin/bash
# 模板：表单自动化工作流
# 目的：填写并提交带校验的网页表单
# 用法：./form-automation.sh <form-url>
#
# 这个模板演示 snapshot-interact-verify 模式：
# 1. 导航到表单
# 2. 做快照，获取元素 refs
# 3. 使用 refs 填写字段
# 4. 提交并验证结果
#
# 定制方式：根据你表单的 snapshot 输出，更新 refs（@e1、@e2 等）

set -euo pipefail

FORM_URL="${1:?用法：$0 <form-url>}"

echo "表单自动化：$FORM_URL"

# 第 1 步：导航到表单
agent-browser open "$FORM_URL"
agent-browser wait --load networkidle

# 第 2 步：做快照，识别表单元素
echo ""
echo "表单结构："
agent-browser snapshot -i

# 第 3 步：填写表单字段（根据 snapshot 输出定制这些 refs）
#
# 常见字段类型：
#   agent-browser fill @e1 "John Doe"           # 文本输入框
#   agent-browser fill @e2 "user@example.com"   # 邮箱输入框
#   agent-browser fill @e3 "SecureP@ss123"      # 密码输入框
#   agent-browser select @e4 "Option Value"     # 下拉框
#   agent-browser check @e5                     # 复选框
#   agent-browser click @e6                     # 单选按钮
#   agent-browser fill @e7 "Multi-line text"    # 文本域
#   agent-browser upload @e8 /path/to/file.pdf  # 文件上传
#
# 取消注释并修改：
# agent-browser fill @e1 "Test User"
# agent-browser fill @e2 "test@example.com"
# agent-browser click @e3  # 提交按钮

# 第 4 步：等待提交完成
# agent-browser wait --load networkidle
# agent-browser wait --url "**/success"  # 或等待跳转

# 第 5 步：验证结果
echo ""
echo "结果："
agent-browser get url
agent-browser snapshot -i

# 可选：抓取证据
agent-browser screenshot /tmp/form-result.png
echo "截图已保存：/tmp/form-result.png"

# 清理
agent-browser close
echo "完成"
