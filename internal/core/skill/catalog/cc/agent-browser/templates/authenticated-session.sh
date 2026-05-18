#!/bin/bash
# 模板：已认证会话工作流
# 目的：登录一次，保存状态，供后续运行复用
# 用法：./authenticated-session.sh <login-url> [state-file]
#
# 推荐：优先使用认证保险箱，而不是这个模板：
#   echo "<pass>" | agent-browser auth save myapp --url <login-url> --username <user> --password-stdin
#   agent-browser auth login myapp
# 认证保险箱会安全存储凭据，LLM 不会看到密码。
#
# 环境变量：
#   APP_USERNAME - 登录用户名/邮箱
#   APP_PASSWORD - 登录密码
#
# 两种模式：
#   1. 探测模式（默认）：展示表单结构，便于你识别 refs
#   2. 登录模式：在你更新 refs 后执行实际登录
#
# 初始化步骤：
#   1. 先运行一次查看表单结构（探测模式）
#   2. 在下方 LOGIN FLOW 区域更新 refs
#   3. 设置 APP_USERNAME 和 APP_PASSWORD
#   4. 删除 DISCOVERY 区域

set -euo pipefail

LOGIN_URL="${1:?用法：$0 <login-url> [state-file]}"
STATE_FILE="${2:-./auth-state.json}"

echo "认证工作流：$LOGIN_URL"

# ================================================================
# 已保存状态：如果存在有效的已保存状态，则跳过登录
# ================================================================
if [[ -f "$STATE_FILE" ]]; then
    echo "正在从 $STATE_FILE 加载已保存状态..."
    if agent-browser --state "$STATE_FILE" open "$LOGIN_URL" 2>/dev/null; then
        agent-browser wait --load networkidle

        CURRENT_URL=$(agent-browser get url)
        if [[ "$CURRENT_URL" != *"login"* ]] && [[ "$CURRENT_URL" != *"signin"* ]]; then
            echo "会话恢复成功"
            agent-browser snapshot -i
            exit 0
        fi
        echo "会话已过期，开始重新登录..."
        agent-browser close 2>/dev/null || true
    else
        echo "状态加载失败，开始重新认证..."
    fi
    rm -f "$STATE_FILE"
fi

# ================================================================
# 探测模式：展示表单结构（完成配置后可删除）
# ================================================================
echo "正在打开登录页..."
agent-browser open "$LOGIN_URL"
agent-browser wait --load networkidle

echo ""
echo "登录表单结构："
echo "---"
agent-browser snapshot -i
echo "---"
echo ""
echo "下一步："
echo "  1. 记下 refs：username=@e?、password=@e?、submit=@e?"
echo "  2. 用你的 refs 更新下方 LOGIN FLOW 区域"
echo "  3. 设置：export APP_USERNAME='...' APP_PASSWORD='...'"
echo "  4. 删除这一段 DISCOVERY MODE"
echo ""
agent-browser close
exit 0

# ================================================================
# 登录流程：完成探测后取消注释并按需定制
# ================================================================
# : "${APP_USERNAME:?请设置 APP_USERNAME 环境变量}"
# : "${APP_PASSWORD:?请设置 APP_PASSWORD 环境变量}"
#
# agent-browser open "$LOGIN_URL"
# agent-browser wait --load networkidle
# agent-browser snapshot -i
#
# # 填写凭据（把 refs 更新为与你表单匹配的值）
# agent-browser fill @e1 "$APP_USERNAME"
# agent-browser fill @e2 "$APP_PASSWORD"
# agent-browser click @e3
# agent-browser wait --load networkidle
#
# # 验证登录成功
# FINAL_URL=$(agent-browser get url)
# if [[ "$FINAL_URL" == *"login"* ]] || [[ "$FINAL_URL" == *"signin"* ]]; then
#     echo "登录失败 - 仍停留在登录页"
#     agent-browser screenshot /tmp/login-failed.png
#     agent-browser close
#     exit 1
# fi
#
# # 保存状态供后续运行复用
# echo "正在将状态保存到 $STATE_FILE"
# agent-browser state save "$STATE_FILE"
# echo "登录成功"
# agent-browser snapshot -i
