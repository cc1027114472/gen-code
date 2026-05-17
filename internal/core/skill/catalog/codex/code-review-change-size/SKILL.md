---
name: code-review-change-size
description: Change size guidance (800 lines)
---

除非该变更是机械性的，否则总变更行数不应超过 800 行。
对于复杂逻辑变更，规模应低于 500 行。

如果变更更大，解释它是否可以拆分为可审查的阶段，并指出最小的、连贯的、可以优先落地的阶段。
基于实际 diff、依赖关系和受影响的调用点来给出分阶段建议。

