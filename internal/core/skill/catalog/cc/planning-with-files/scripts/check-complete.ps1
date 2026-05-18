# 检查 task_plan.md 中的所有阶段是否已完成
# 始终以 0 退出，使用标准输出报告状态
# 由 Stop hook 调用，用于汇报任务完成状态

param(
    [string]$PlanFile = "task_plan.md"
)

if (-not (Test-Path $PlanFile)) {
    Write-Host '[planning-with-files] 未找到 task_plan.md，当前没有激活的规划会话。'
    exit 0
}

# 读取文件内容
$content = Get-Content $PlanFile -Raw

# 统计阶段总数
$TOTAL = ([regex]::Matches($content, "### Phase")).Count

# 优先检查 **Status:** 格式
$COMPLETE = ([regex]::Matches($content, "\*\*Status:\*\* complete")).Count
$IN_PROGRESS = ([regex]::Matches($content, "\*\*Status:\*\* in_progress")).Count
$PENDING = ([regex]::Matches($content, "\*\*Status:\*\* pending")).Count

# 兜底：如果没有 **Status:**，则检查内联 [complete] 格式
if ($COMPLETE -eq 0 -and $IN_PROGRESS -eq 0 -and $PENDING -eq 0) {
    $COMPLETE = ([regex]::Matches($content, "\[complete\]")).Count
    $IN_PROGRESS = ([regex]::Matches($content, "\[in_progress\]")).Count
    $PENDING = ([regex]::Matches($content, "\[pending\]")).Count
}

# 报告状态，始终以 0 退出；任务未完成属于正常状态
if ($COMPLETE -eq $TOTAL -and $TOTAL -gt 0) {
    Write-Host ('[planning-with-files] 所有阶段均已完成（' + $COMPLETE + '/' + $TOTAL + '）。如果用户还有额外工作，请先在 task_plan.md 中新增阶段，再继续开始。')
} else {
    Write-Host ('[planning-with-files] 任务进行中（已完成 ' + $COMPLETE + '/' + $TOTAL + ' 个阶段）。停止前请更新 progress.md。')
    if ($IN_PROGRESS -gt 0) {
        Write-Host ('[planning-with-files] 仍有 ' + $IN_PROGRESS + ' 个阶段处于进行中。')
    }
    if ($PENDING -gt 0) {
        Write-Host ('[planning-with-files] 仍有 ' + $PENDING + ' 个阶段处于待开始。')
    }
}
exit 0
