import json
import os
import sys
import time
import urllib.request
import uuid

from playwright.sync_api import expect, sync_playwright


UI_BASE_URL = os.environ.get("GEN_CODE_UI_BASE_URL", "http://127.0.0.1:5174/")
API_BASE_URL = os.environ.get("GEN_CODE_API_BASE_URL", "http://127.0.0.1:10008")


def api(method: str, path: str, data=None):
    body = None
    headers = {}
    if data is not None:
        body = json.dumps(data).encode("utf-8")
        headers["Content-Type"] = "application/json"
    request = urllib.request.Request(API_BASE_URL + path, data=body, headers=headers, method=method)
    with urllib.request.urlopen(request, timeout=30) as response:
        payload = json.loads(response.read().decode("utf-8"))
    return payload["data"]


def create_thread(name: str) -> dict:
    return api(
        "POST",
        "/api/threads",
        {
            "name": name,
            "permissionMode": "ask-user",
        },
    )


def activate_thread(thread_id: str):
    return api("POST", f"/api/threads/{thread_id}/activate", {})


def wait_for_active_thread(page, thread_id: str):
    thread_card = page.locator(f'[data-testid="thread-card-{thread_id}"]')
    expect(thread_card).to_be_visible(timeout=15000)
    if thread_card.get_attribute("data-active") != "true":
        thread_card.click()
    expect(thread_card).to_have_attribute("data-active", "true", timeout=15000)
    return thread_card


def wait_for_task(thread_id: str, predicate, timeout_seconds: float = 20.0):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        tasks = api("GET", f"/api/threads/{thread_id}/tasks")["items"]
        for item in reversed(tasks):
            if predicate(item):
                return item
        time.sleep(0.5)
    raise RuntimeError("timed out waiting for matching task")


def wait_for_write_execution(thread_id: str, predicate, timeout_seconds: float = 20.0):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        items = api("GET", f"/api/threads/{thread_id}/write-executions")["items"]
        for item in reversed(items):
            if predicate(item):
                return item
        time.sleep(0.5)
    raise RuntimeError("timed out waiting for matching write execution")


def main() -> int:
    run_id = f"{int(time.time())}-{uuid.uuid4().hex[:8]}"
    thread_name = f"Playwright Acceptance {run_id}"
    task_title = f"Playwright Live Refresh {run_id}"
    patch_path = f"playwright-live-refresh-{run_id}.txt"
    rollback_task_title = "Rollback latest write execution"

    created_thread = create_thread(thread_name)
    thread_id = created_thread["id"]
    activate_thread(thread_id)

    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(headless=True)
        page = browser.new_page(viewport={"width": 1600, "height": 1000})
        try:
            page.goto(UI_BASE_URL, wait_until="domcontentloaded", timeout=30000)
            page.wait_for_timeout(3000)
            wait_for_active_thread(page, thread_id)

            created_task = api(
                "POST",
                f"/api/threads/{thread_id}/tasks",
                {
                    "title": task_title,
                    "kind": "workspace.apply_patch",
                    "input": json.dumps(
                        {
                            "path": patch_path,
                            "patch": (
                                "*** Begin Patch\n"
                                f"*** Add File: {patch_path}\n"
                                "+playwright live refresh\n"
                                "*** End Patch\n"
                            ),
                        }
                    ),
                },
            )
            created_task_id = created_task["id"]

            pending_card = page.locator("article").filter(has_text=task_title).filter(has_text="approval required").first
            expect(pending_card).to_be_visible(timeout=15000)

            approved_task = api("POST", f"/api/threads/{thread_id}/tasks/{created_task_id}/approve", {})

            apply_execution = wait_for_write_execution(
                thread_id,
                lambda item: item["taskId"] == created_task_id and item["operation"] == "apply" and item["status"] == "completed",
            )

            write_execution_panel = page.get_by_test_id("write-execution-panel")
            expect(write_execution_panel).to_contain_text(apply_execution["resultSummary"], timeout=15000)
            expect(write_execution_panel).to_contain_text(patch_path, timeout=15000)

            rollback_button = page.get_by_test_id(f'rollback-latest-{apply_execution["id"]}')
            expect(rollback_button).to_be_visible(timeout=15000)
            expect(rollback_button).to_be_enabled(timeout=15000)
            rollback_button.click()

            rollback_pending = page.locator("article").filter(has_text=rollback_task_title).filter(has_text="approval required for rollback").first
            expect(rollback_pending).to_be_visible(timeout=15000)
            expect(rollback_pending).to_contain_text(patch_path, timeout=15000)

            rollback_task = wait_for_task(
                thread_id,
                lambda item: item["title"] == rollback_task_title
                and item["kind"] == "workspace.apply_patch.rollback"
                and apply_execution["id"] in item["inputSummary"],
            )
            rollback_task_id = rollback_task["id"]

            rollback_approved = api("POST", f"/api/threads/{thread_id}/tasks/{rollback_task_id}/approve", {})

            rollback_execution = wait_for_write_execution(
                thread_id,
                lambda item: item["taskId"] == rollback_task_id and item["operation"] == "rollback" and item["status"] == "completed",
            )

            rollback_completed = page.locator("article").filter(has_text=rollback_task_title).filter(has_text=rollback_execution["resultSummary"]).first
            expect(rollback_completed).to_be_visible(timeout=15000)

            expect(write_execution_panel).to_contain_text(rollback_execution["resultSummary"], timeout=15000)
            expect(write_execution_panel).to_contain_text(apply_execution["id"], timeout=15000)

            print(
                json.dumps(
                    {
                        "ok": True,
                        "threadId": thread_id,
                        "threadName": thread_name,
                        "taskId": created_task_id,
                        "taskTitle": task_title,
                        "createdStatus": created_task["status"],
                        "approvedStatus": approved_task["status"],
                        "writeExecutionId": apply_execution["id"],
                        "rollbackTaskId": rollback_task_id,
                        "rollbackTaskTitle": rollback_task_title,
                        "rollbackStatus": rollback_approved["status"],
                        "rollbackWriteExecutionId": rollback_execution["id"],
                        "uiBaseUrl": UI_BASE_URL,
                        "apiBaseUrl": API_BASE_URL,
                    },
                    ensure_ascii=False,
                )
            )
            return 0
        finally:
            browser.close()


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:
        print(json.dumps({"ok": False, "error": str(exc)}, ensure_ascii=False), file=sys.stderr)
        raise
