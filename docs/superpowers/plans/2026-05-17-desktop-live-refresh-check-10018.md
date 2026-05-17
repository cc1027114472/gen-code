# Desktop Live Refresh Check For Remote 10018

## Purpose

This note records a successful browser-level acceptance run against:

- UI: `http://127.0.0.1:5174/`
- remote runtime: `http://127.0.0.1:10018`

It exists because the old default remote port `10008` was still occupied by a stale long-lived service during this session.

## What Was Adjusted

Two small script-level fixes were needed so the acceptance lane would target the correct runtime and survive transient local HTTP resets:

- add minimal retry logic for API requests in `scripts/verify-desktop-live-refresh.py`
- inject `window.__GENCODE_RUNTIME_BASE_URL__` before page load so the UI reads the same runtime base URL as the test script

These were acceptance-lane fixes, not product-behavior workarounds.

## Result

Successful run:

```json
{
  "ok": true,
  "threadId": "thread-117",
  "threadName": "Playwright Acceptance 1779003510-6e4e068d",
  "taskId": "task-179",
  "taskTitle": "Playwright Live Refresh 1779003510-6e4e068d",
  "createdStatus": "needs_approval",
  "approvedStatus": "completed",
  "writeExecutionId": "writeexec-29",
  "rollbackTaskId": "task-180",
  "rollbackTaskTitle": "Rollback latest write execution",
  "rollbackStatus": "completed",
  "rollbackWriteExecutionId": "writeexec-30",
  "uiBaseUrl": "http://127.0.0.1:5174/",
  "apiBaseUrl": "http://127.0.0.1:10018"
}
```

## Verified Chain

The successful run verified the following browser-visible sequence:

1. Create an isolated `ask-user` thread
2. Create a `workspace.apply_patch` write task
3. See the pending approval card appear in the UI
4. Approve the task
5. See the write execution panel update with the completed apply summary
6. Click `rollback latest`
7. See the rollback approval card appear in the UI
8. Approve rollback
9. See rollback completion and the new rollback write execution summary in the UI

## Conclusion

This provides a real desktop/browser acceptance proof for the current implementation, using the trusted clean-port remote runtime on `10018`.
