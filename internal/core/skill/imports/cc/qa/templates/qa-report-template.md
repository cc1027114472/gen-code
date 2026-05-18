# QA 报告：{APP_NAME}

| 字段 | 值 |
|------|----|
| **日期** | {DATE} |
| **URL** | {URL} |
| **分支** | {BRANCH} |
| **提交** | {COMMIT_SHA} ({COMMIT_DATE}) |
| **PR** | {PR_NUMBER} ({PR_URL}) 或 `—` |
| **层级** | Quick / Standard / Exhaustive |
| **范围** | {SCOPE 或 "Full app"} |
| **耗时** | {DURATION} |
| **访问页面数** | {COUNT} |
| **截图数** | {COUNT} |
| **框架** | {DETECTED 或 "Unknown"} |
| **索引** | [All QA runs](./index.md) |

## 健康分：{SCORE}/100

| 类别 | 分数 |
|------|------|
| Console | {0-100} |
| Links | {0-100} |
| Visual | {0-100} |
| Functional | {0-100} |
| UX | {0-100} |
| Performance | {0-100} |
| Accessibility | {0-100} |

## Top 3 Things to Fix

1. **{ISSUE-NNN}: {title}**，{one-line description}
2. **{ISSUE-NNN}: {title}**，{one-line description}
3. **{ISSUE-NNN}: {title}**，{one-line description}

## Console 健康

| 错误 | 次数 | 首次出现位置 |
|------|------|--------------|
| {error message} | {N} | {URL} |

## Summary

| 严重级别 | 数量 |
|----------|------|
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 0 |
| **Total** | **0** |

## Issues

### ISSUE-001: {Short title}

| 字段 | 值 |
|------|----|
| **Severity** | critical / high / medium / low |
| **Category** | visual / functional / ux / content / performance / console / accessibility |
| **URL** | {page URL} |

**Description:** {What is wrong, expected vs actual.}

**Repro Steps:**

1. 打开 {URL}
   ![Step 1](screenshots/issue-001-step-1.png)
2. {Action}
   ![Step 2](screenshots/issue-001-step-2.png)
3. **Observe:** {what goes wrong}
   ![Result](screenshots/issue-001-result.png)

---

## Fixes Applied（如适用）

| Issue | Fix Status | Commit | Files Changed |
|-------|------------|--------|---------------|
| ISSUE-NNN | verified / best-effort / reverted / deferred | {SHA} | {files} |

### Before / After 证据

#### ISSUE-NNN: {title}
**Before:** ![Before](screenshots/issue-NNN-before.png)
**After:** ![After](screenshots/issue-NNN-after.png)

---

## Regression Tests

| Issue | Test File | Status | Description |
|-------|-----------|--------|-------------|
| ISSUE-NNN | path/to/test | committed / deferred / skipped | description |

### Deferred Tests

#### ISSUE-NNN: {title}
**Precondition:** {setup state that triggers the bug}  
**Action:** {what the user does}  
**Expected:** {correct behavior}  
**Why deferred:** {reason}

---

## Ship Readiness

| 指标 | 值 |
|------|----|
| Health score | {before} → {after} ({delta}) |
| Issues found | N |
| Fixes applied | N (verified: X, best-effort: Y, reverted: Z) |
| Deferred | N |

**PR Summary:** "QA found N issues, fixed M, health score X → Y."

---

## Regression（如适用）

| 指标 | Baseline | Current | Delta |
|------|----------|---------|-------|
| Health score | {N} | {N} | {+/-N} |
| Issues | {N} | {N} | {+/-N} |

**Fixed since baseline:** {list}  
**New since baseline:** {list}
