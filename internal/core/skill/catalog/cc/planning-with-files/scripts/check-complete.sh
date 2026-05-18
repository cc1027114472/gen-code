#!/bin/bash
# 检查 task_plan.md 中的所有阶段是否已完成
# 始终以 0 退出，使用标准输出报告状态
# 由 Stop hook 调用，用于汇报任务完成状态

PLAN_FILE="${1:-task_plan.md}"

if [ ! -f "$PLAN_FILE" ]; then
    echo "[planning-with-files] 未找到 task_plan.md，当前没有激活的规划会话。"
    exit 0
fi

# 统计阶段总数
TOTAL=$(grep -c "### Phase" "$PLAN_FILE" || true)

# 优先检查 **Status:** 格式
COMPLETE=$(grep -cF "**Status:** complete" "$PLAN_FILE" || true)
IN_PROGRESS=$(grep -cF "**Status:** in_progress" "$PLAN_FILE" || true)
PENDING=$(grep -cF "**Status:** pending" "$PLAN_FILE" || true)

# 兜底：如果没有 **Status:**，则检查内联 [complete] 格式
if [ "$COMPLETE" -eq 0 ] && [ "$IN_PROGRESS" -eq 0 ] && [ "$PENDING" -eq 0 ]; then
    COMPLETE=$(grep -c "\[complete\]" "$PLAN_FILE" || true)
    IN_PROGRESS=$(grep -c "\[in_progress\]" "$PLAN_FILE" || true)
    PENDING=$(grep -c "\[pending\]" "$PLAN_FILE" || true)
fi

# 若为空则默认置为 0
: "${TOTAL:=0}"
: "${COMPLETE:=0}"
: "${IN_PROGRESS:=0}"
: "${PENDING:=0}"

# 报告状态，始终以 0 退出；任务未完成属于正常状态
if [ "$COMPLETE" -eq "$TOTAL" ] && [ "$TOTAL" -gt 0 ]; then
    echo "[planning-with-files] 所有阶段均已完成（$COMPLETE/$TOTAL）。如果用户还有额外工作，请先在 task_plan.md 中新增阶段，再继续开始。"
else
    echo "[planning-with-files] 任务进行中（已完成 $COMPLETE/$TOTAL 个阶段）。停止前请更新 progress.md。"
    if [ "$IN_PROGRESS" -gt 0 ]; then
        echo "[planning-with-files] 仍有 $IN_PROGRESS 个阶段处于进行中。"
    fi
    if [ "$PENDING" -gt 0 ]; then
        echo "[planning-with-files] 仍有 $PENDING 个阶段处于待开始。"
    fi
fi
exit 0
