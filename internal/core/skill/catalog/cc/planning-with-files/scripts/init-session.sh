#!/bin/bash
# 为新会话初始化规划文件
# 用法：./init-session.sh [project-name]

set -e

PROJECT_NAME="${1:-project}"
DATE=$(date +%Y-%m-%d)

echo "正在为以下项目初始化规划文件：$PROJECT_NAME"

# 如果 task_plan.md 不存在则创建
if [ ! -f "task_plan.md" ]; then
    cat > task_plan.md << 'EOF'
# 任务计划：[简短描述]

## 目标
[用一句话描述最终状态]

## 当前阶段
阶段 1

## 各阶段

### 阶段 1：需求与探索
- [ ] 理解用户意图
- [ ] 识别约束
- [ ] 记录到 findings.md
- **Status:** in_progress

### 阶段 2：规划与结构
- [ ] 明确方案
- [ ] 创建项目结构
- **Status:** pending

### 阶段 3：实施
- [ ] 执行计划
- [ ] 在执行前先写入文件
- **Status:** pending

### 阶段 4：测试与验证
- [ ] 验证需求已满足
- [ ] 记录测试结果
- **Status:** pending

### 阶段 5：交付
- [ ] 审查输出
- [ ] 向用户交付
- **Status:** pending

## 已做决策
| 决策 | 原因 |
|------|------|

## 遇到的错误
| 错误 | 处理方式 |
|------|----------|
EOF
    echo "已创建 task_plan.md"
else
    echo "task_plan.md 已存在，跳过创建"
fi

# 如果 findings.md 不存在则创建
if [ ! -f "findings.md" ]; then
    cat > findings.md << 'EOF'
# 发现与决策

## 需求
-

## 研究发现
-

## 技术决策
| 决策 | 原因 |
|------|------|

## 遇到的问题
| 问题 | 处理方式 |
|------|----------|

## 资源
-
EOF
    echo "已创建 findings.md"
else
    echo "findings.md 已存在，跳过创建"
fi

# 如果 progress.md 不存在则创建
if [ ! -f "progress.md" ]; then
    cat > progress.md << EOF
# 进度日志

## 会话：$DATE

### 当前状态
- **Phase:** 1 - 需求与探索
- **Started:** $DATE

### 已执行操作
-

### 测试结果
| 测试 | 期望结果 | 实际结果 | 状态 |
|------|----------|----------|------|

### 错误
| 错误 | 处理方式 |
|------|----------|
EOF
    echo "已创建 progress.md"
else
    echo "progress.md 已存在，跳过创建"
fi

echo ""
echo "规划文件初始化完成！"
echo "文件：task_plan.md、findings.md、progress.md"
