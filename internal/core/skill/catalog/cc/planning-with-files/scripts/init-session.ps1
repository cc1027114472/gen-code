# 为新会话初始化规划文件
# 用法：.\init-session.ps1 [project-name]

param(
    [string]$ProjectName = "project"
)

$DATE = Get-Date -Format "yyyy-MM-dd"

Write-Host "正在为以下项目初始化规划文件：$ProjectName"

# 如果 task_plan.md 不存在则创建
if (-not (Test-Path "task_plan.md")) {
    @"
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
"@ | Out-File -FilePath "task_plan.md" -Encoding UTF8
    Write-Host "已创建 task_plan.md"
} else {
    Write-Host "task_plan.md 已存在，跳过创建"
}

# 如果 findings.md 不存在则创建
if (-not (Test-Path "findings.md")) {
    @"
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
"@ | Out-File -FilePath "findings.md" -Encoding UTF8
    Write-Host "已创建 findings.md"
} else {
    Write-Host "findings.md 已存在，跳过创建"
}

# 如果 progress.md 不存在则创建
if (-not (Test-Path "progress.md")) {
    @"
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
"@ | Out-File -FilePath "progress.md" -Encoding UTF8
    Write-Host "已创建 progress.md"
} else {
    Write-Host "progress.md 已存在，跳过创建"
}

Write-Host ""
Write-Host "规划文件初始化完成！"
Write-Host "文件：task_plan.md、findings.md、progress.md"
