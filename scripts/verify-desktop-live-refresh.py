import json
import os
import socket
import sys
import time
import urllib.error
import urllib.request
import uuid

from playwright.sync_api import TimeoutError as PlaywrightTimeoutError
from playwright.sync_api import expect, sync_playwright


UI_BASE_URL = os.environ.get("GEN_CODE_UI_BASE_URL", "http://127.0.0.1:5174/")
API_BASE_URL = os.environ.get("GEN_CODE_API_BASE_URL", "http://127.0.0.1:10008")
API_RETRIES = int(os.environ.get("GEN_CODE_API_RETRIES", "5"))
API_RETRY_DELAY = float(os.environ.get("GEN_CODE_API_RETRY_DELAY", "0.5"))
ACCEPTANCE_MODE = os.environ.get("GEN_CODE_ACCEPTANCE_MODE", "full").strip().lower() or "full"
ARTIFACT_DIR = os.environ.get("GEN_CODE_ARTIFACT_DIR", os.path.join("tmp", "desktop-smoke-artifacts"))

SECOND_BATCH_TOOL_KINDS = {
    "workspace.stat_file",
    "workspace.read_files_batch",
    "workspace.list_files_filtered",
    "workspace.search_text_detailed",
}

EXPECTED_REMOTE_COPY_TEXT = [
    "线程工作台",
    "工作区线程",
    "运行链路",
    "刷新方式",
    "结果抽屉",
]

EXPECTED_REMOTE_COPY_GROUPS = [
    ["本地预览", "收起预览", "展开预览"],
]

UNEXPECTED_REMOTE_COPY_TEXT = [
    "Thread 工作台",
    "latest task ",
    "workspace loading",
    "threads 0",
    "desktop local-fallback active; manual refresh mode",
]

AGENT_FAILURE_MATRIX = {
    "success_resume_baseline": {"lane": "remote-canonical", "required": True},
    "approval_rejected": {"lane": "remote-canonical", "required": True},
    "child_task_failed": {"lane": "remote-canonical", "required": True},
}

FALLBACK_EVIDENCE_MATRIX = {
    "recovered_as_failed": {"lane": "fallback-evidence", "required": True},
}


def summarize_refresh_mode(page) -> dict:
    body_text = page.locator("body").inner_text(timeout=5000)
    if "SSE 实时刷新" in body_text:
        return {
            "label": "SSE 实时刷新",
            "detail": "UI reported active SSE live refresh",
            "supportsSSE": True,
            "sseConnected": True,
            "evidence": "SSE 实时刷新",
        }
    if "SSE 重连中" in body_text:
        return {
            "label": "SSE 重连中",
            "detail": "UI reported SSE support but no active connection",
            "supportsSSE": True,
            "sseConnected": False,
            "evidence": "SSE 重连中",
        }
    if "SSE 已连接" in body_text:
        return {
            "label": "SSE 已连接",
            "detail": "UI reported active SSE connection in status detail",
            "supportsSSE": True,
            "sseConnected": True,
            "evidence": "SSE 已连接",
        }
    if "SSE 已断开" in body_text:
        return {
            "label": "SSE 已断开",
            "detail": "UI reported SSE disconnect and fallback to manual refresh",
            "supportsSSE": True,
            "sseConnected": False,
            "evidence": "SSE 已断开",
        }
    if "手动刷新" in body_text:
        return {
            "label": "手动刷新",
            "detail": "UI reported manual refresh fallback",
            "supportsSSE": False,
            "sseConnected": False,
            "evidence": "手动刷新",
        }
    if "刷新方式" in body_text and "当前未接入 SSE" in body_text:
        return {
            "label": "手动刷新",
            "detail": "UI reported no SSE endpoint and manual refresh mode",
            "supportsSSE": False,
            "sseConnected": False,
            "evidence": "当前未接入 SSE",
        }
    return {
        "label": "unknown",
        "detail": "UI refresh mode text was not detected",
        "supportsSSE": None,
        "sseConnected": None,
        "evidence": "",
    }


def assert_desktop_copy_and_runtime_lane(page, runtime_status: dict, refresh_mode: dict) -> dict:
    body = page.locator("body")
    body_text = ""
    for _ in range(10):
        body_text = body.inner_text(timeout=5000)
        if "共享运行时 / remote-app-server" in body_text and "可信链路 / canonical" in body_text:
            break
        page.wait_for_timeout(500)

    for expected in EXPECTED_REMOTE_COPY_TEXT:
        if expected not in body_text:
            fail(
                f"expected desktop copy text {expected!r} to be visible in remote acceptance lane",
                category="desktop-copy-assertion",
                details={"missingText": expected},
            )

    for unexpected in UNEXPECTED_REMOTE_COPY_TEXT:
        if unexpected in body_text:
            fail(
                f"unexpected stale desktop copy text {unexpected!r} was visible in remote acceptance lane",
                category="desktop-copy-assertion",
                details={"unexpectedText": unexpected},
            )

    for group in EXPECTED_REMOTE_COPY_GROUPS:
        if not any(item in body_text for item in group):
            fail(
                f"expected at least one desktop copy variant from {group!r} to be visible in remote acceptance lane",
                category="desktop-copy-assertion",
                details={"missingGroup": group},
            )

    expected_lane = "remote-app-server"
    expected_trust = "canonical"
    if expected_lane not in body_text:
        fail(
            f"expected runtime lane token {expected_lane!r} to be visible",
            category="runtime-lane-assertion",
            details={"runtimeSource": runtime_status.get('runtimeSource', '')},
        )
    if expected_trust not in body_text:
        fail(
            f"expected runtime trust token {expected_trust!r} to be visible",
            category="runtime-lane-assertion",
            details={"runtimeTrust": runtime_status.get('runtimeTrust', '')},
        )
    if refresh_mode.get("evidence") and refresh_mode["evidence"] not in body_text:
        fail(
            f"expected refresh mode evidence {refresh_mode['evidence']!r} to be visible",
            category="refresh-mode-assertion",
            details=refresh_mode,
        )

    return {
        "checkedTexts": EXPECTED_REMOTE_COPY_TEXT,
        "checkedGroups": EXPECTED_REMOTE_COPY_GROUPS,
        "unexpectedTexts": UNEXPECTED_REMOTE_COPY_TEXT,
        "runtimeLaneLabel": expected_lane,
        "runtimeTrustLabel": expected_trust,
        "refreshEvidence": refresh_mode.get("evidence", ""),
    }


def fail(message: str, *, category: str, details=None):
    payload = {
        "ok": False,
        "category": category,
        "error": message,
    }
    if details is not None:
        payload["details"] = details
    raise RuntimeError(json.dumps(payload, ensure_ascii=False))


def normalize_failure(exc: Exception) -> dict:
    try:
        return json.loads(str(exc))
    except json.JSONDecodeError:
        pass

    details = {
        "exceptionType": type(exc).__name__,
        "exception": str(exc),
    }

    if isinstance(exc, (urllib.error.HTTPError, urllib.error.URLError, socket.timeout, ConnectionResetError, OSError)):
        return {
            "ok": False,
            "category": "api-unavailable",
            "error": "runtime API was not reachable during desktop acceptance",
            "details": details,
        }

    if isinstance(exc, PlaywrightTimeoutError):
        return {
            "ok": False,
            "category": "page-load-failed",
            "error": "desktop page did not load in time during acceptance",
            "details": details,
        }

    message = str(exc)
    if "thread card was not rendered" in message:
        return {
            "ok": False,
            "category": "page-load-failed",
            "error": "desktop page loaded, but the active thread card never rendered",
            "details": details,
        }

    return {
        "ok": False,
        "category": "unknown",
        "error": message,
        "details": details,
    }


def ensure_artifact_dir() -> str:
    os.makedirs(ARTIFACT_DIR, exist_ok=True)
    return ARTIFACT_DIR


def write_json_artifact(name: str, payload: dict) -> str:
    artifact_dir = ensure_artifact_dir()
    path = os.path.join(artifact_dir, name)
    with open(path, "w", encoding="utf-8") as handle:
        json.dump(payload, handle, ensure_ascii=False, indent=2)
    return path


def write_png_artifact(name: str, payload: bytes) -> str:
    artifact_dir = ensure_artifact_dir()
    path = os.path.join(artifact_dir, name)
    with open(path, "wb") as handle:
        handle.write(payload)
    return path


def api(method: str, path: str, data=None):
    body = None
    headers = {}
    if data is not None:
        body = json.dumps(data).encode("utf-8")
        headers["Content-Type"] = "application/json"

    last_error = None
    for attempt in range(API_RETRIES):
        request = urllib.request.Request(API_BASE_URL + path, data=body, headers=headers, method=method)
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                payload = json.loads(response.read().decode("utf-8"))
            return payload["data"]
        except (ConnectionResetError, urllib.error.URLError, socket.timeout, OSError) as exc:
            last_error = exc
            if attempt == API_RETRIES - 1:
                fail(
                    f"runtime API request failed for {method} {path}",
                    category="api-unavailable",
                    details={
                        "method": method,
                        "path": path,
                        "baseUrl": API_BASE_URL,
                        "exceptionType": type(exc).__name__,
                        "exception": str(exc),
                    },
                )
            time.sleep(API_RETRY_DELAY * (attempt + 1))

    if last_error is not None:
        fail(
            f"runtime API request failed for {method} {path}",
            category="api-unavailable",
            details={
                "method": method,
                "path": path,
                "baseUrl": API_BASE_URL,
                "exceptionType": type(last_error).__name__,
                "exception": str(last_error),
            },
        )
    fail(
        f"runtime API request failed for {method} {path}",
        category="api-unavailable",
        details={"method": method, "path": path, "baseUrl": API_BASE_URL},
    )


def ensure_canonical_runtime() -> dict:
    status = api("GET", "/api/runtime/status")
    runtime_source = status.get("runtimeSource", "")
    runtime_trust = status.get("runtimeTrust", "")
    canonical_runtime_url = (status.get("canonicalRuntimeUrl", "") or "").rstrip("/")
    expected_runtime_url = API_BASE_URL.rstrip("/")

    if runtime_source != "remote-app-server":
        fail(
            f"expected canonical remote runtime source 'remote-app-server', got {runtime_source!r}",
            category="runtime-lane-assertion",
            details=status,
        )
    if runtime_trust != "canonical":
        fail(
            f"expected canonical runtime trust 'canonical', got {runtime_trust!r}",
            category="runtime-lane-assertion",
            details=status,
        )
    if canonical_runtime_url and canonical_runtime_url != expected_runtime_url:
        fail(
            f"canonical runtime URL mismatch: expected {expected_runtime_url}, got {canonical_runtime_url}",
            category="runtime-lane-assertion",
            details=status,
        )
    return status


def create_thread(name: str, permission_mode: str = "ask-user") -> dict:
    return api(
        "POST",
        "/api/threads",
        {
            "name": name,
            "permissionMode": permission_mode,
        },
    )


def activate_thread(thread_id: str):
    return api("POST", f"/api/threads/{thread_id}/activate", {})


def create_task(thread_id: str, title: str, kind: str, task_input) -> dict:
    return api(
        "POST",
        f"/api/threads/{thread_id}/tasks",
        {
            "title": title,
            "kind": kind,
            "input": json.dumps(task_input, ensure_ascii=False),
        },
    )


def run_task(thread_id: str, task_id: str) -> dict:
    return api("POST", f"/api/threads/{thread_id}/tasks/{task_id}/run", {})


def reject_task(thread_id: str, task_id: str) -> dict:
    return api("POST", f"/api/threads/{thread_id}/tasks/{task_id}/reject", {})


def wait_for_active_thread(page, thread_id: str):
    last_error = None
    for attempt in range(3):
        thread_card = page.locator(f'[data-testid="thread-card-{thread_id}"]')
        try:
            expect(thread_card).to_be_visible(timeout=15000)
            if thread_card.get_attribute("data-active") != "true":
                thread_card.click()
            expect(thread_card).to_have_attribute("data-active", "true", timeout=15000)
            return thread_card
        except Exception as exc:
            last_error = exc
            if attempt == 2:
                raise
            page.reload(wait_until="domcontentloaded", timeout=30000)
            page.wait_for_timeout(2000)
    if last_error is not None:
        raise last_error
    raise RuntimeError("thread card was not rendered")


def wait_for_task(thread_id: str, predicate, timeout_seconds: float = 20.0):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        tasks = api("GET", f"/api/threads/{thread_id}/tasks")["items"]
        for item in reversed(tasks):
            if predicate(item):
                return item
        time.sleep(0.5)
    raise RuntimeError("timed out waiting for matching task")


def wait_for_task_terminal(thread_id: str, task_id: str, timeout_seconds: float = 20.0):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        tasks = api("GET", f"/api/threads/{thread_id}/tasks")["items"]
        for item in tasks:
            if item.get("id") == task_id and item.get("status") in {"completed", "failed"}:
                return item
        time.sleep(0.5)
    raise RuntimeError("timed out waiting for matching task")


def find_latest_child_task(thread_id: str, parent_task_id: str, status: str | None = None) -> dict:
    items = api("GET", f"/api/threads/{thread_id}/tasks")["items"]
    matches = [item for item in items if item.get("parentTaskId") == parent_task_id]
    if status is not None:
        matches = [item for item in matches if item.get("status") == status]
    if not matches:
        status_note = f" with status {status!r}" if status is not None else ""
        raise AssertionError(f"expected at least one child task for parent {parent_task_id!r}{status_note}")
    matches.sort(key=lambda item: item.get("updatedAt", "") or item.get("createdAt", ""))
    return matches[-1]


def find_child_tasks(thread_id: str, parent_task_id: str) -> list[dict]:
    return [
        item
        for item in api("GET", f"/api/threads/{thread_id}/tasks")["items"]
        if item.get("parentTaskId") == parent_task_id
    ]


def wait_for_write_execution(thread_id: str, predicate, timeout_seconds: float = 20.0):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        items = api("GET", f"/api/threads/{thread_id}/write-executions")["items"]
        for item in reversed(items):
            if predicate(item):
                return item
        time.sleep(0.5)
    raise RuntimeError("timed out waiting for matching write execution")


def wait_for_message(thread_id: str, predicate, timeout_seconds: float = 20.0):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        items = api("GET", f"/api/threads/{thread_id}/messages")["items"]
        for item in reversed(items):
            if predicate(item):
                return item
        time.sleep(0.5)
    raise RuntimeError("timed out waiting for matching message")


def wait_for_tool_call(thread_id: str, predicate, timeout_seconds: float = 20.0):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        items = api("GET", f"/api/threads/{thread_id}/tool-calls")["items"]
        for item in reversed(items):
            if predicate(item):
                return item
        time.sleep(0.5)
    raise RuntimeError("timed out waiting for matching tool call")


def wait_for_artifact(thread_id: str, predicate, timeout_seconds: float = 20.0):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        items = api("GET", f"/api/threads/{thread_id}/artifacts")["items"]
        for item in reversed(items):
            if predicate(item):
                return item
        time.sleep(0.5)
    raise RuntimeError("timed out waiting for matching artifact")


def wait_for_runtime_flag(thread_id: str, predicate, timeout_seconds: float = 20.0):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        items = api("GET", f"/api/threads/{thread_id}/runtime-flags")["items"]
        for item in reversed(items):
            if predicate(item):
                return item
        time.sleep(0.5)
    raise RuntimeError("timed out waiting for matching runtime flag")


def assert_article_contains(page, *texts: str, timeout: int = 15000):
    locator = page.locator("article")
    for text in texts:
        locator = locator.filter(has_text=text)
    expect(locator.first).to_be_visible(timeout=timeout)
    return locator.first


def refresh_thread_view(page, thread_id: str):
    page.reload(wait_until="domcontentloaded", timeout=30000)
    page.wait_for_timeout(2000)
    wait_for_active_thread(page, thread_id)


def assert_result_card_contains(page, test_id: str, *texts: str, timeout: int = 15000):
    locator = page.get_by_test_id(test_id)
    expect(locator).to_be_visible(timeout=timeout)
    for text in texts:
        expect(locator).to_contain_text(text, timeout=timeout)
    return locator


def activate_thread_in_ui(page, thread_id: str):
    activate_thread(thread_id)
    refresh_thread_view(page, thread_id)


def run_direct_tool_scenario(page, thread_id: str, scenario: dict) -> dict:
    created_task = create_task(thread_id, scenario["title"], scenario["kind"], scenario["input"])
    task_id = created_task["id"]
    run_task(thread_id, task_id)
    completed_task = wait_for_task(thread_id, lambda item: item["id"] == task_id and item["status"] == "completed")

    expected_summary = scenario["summary_contains"]
    if expected_summary not in completed_task["resultSummary"]:
        raise AssertionError(
            f"task {scenario['kind']} summary mismatch: expected substring {expected_summary!r}, got {completed_task['resultSummary']!r}"
        )

    tool_call = wait_for_tool_call(
        thread_id,
        lambda item: item["toolId"] == scenario["kind"] and item["status"] == "completed" and expected_summary in item["summary"],
    )
    visibility = {
        "taskCardVisible": False,
        "toolKindVisible": False,
    }
    try:
        assert_article_contains(page, scenario["title"])
        visibility["taskCardVisible"] = True
    except Exception:
        try:
            assert_article_contains(page, tool_call["toolId"])
            visibility["toolKindVisible"] = True
        except Exception:
            refresh_thread_view(page, thread_id)
            try:
                assert_article_contains(page, scenario["title"], timeout=10000)
                visibility["taskCardVisible"] = True
            except Exception:
                try:
                    assert_article_contains(page, tool_call["toolId"], timeout=10000)
                    visibility["toolKindVisible"] = True
                except Exception:
                    if not scenario.get("ui_optional", False):
                        raise
    return {
        "task": completed_task,
        "toolCall": tool_call,
        "visibility": visibility,
        "input": scenario["input"],
    }


def run_thread_mutation_scenario(page, thread_id: str, scenario: dict) -> dict:
    scenario_result = run_direct_tool_scenario(page, thread_id, scenario)
    completed_task = scenario_result["task"]
    kind = scenario["kind"]

    if kind == "thread.toolcall.append":
        record = wait_for_tool_call(
            thread_id,
            lambda item: item["toolId"] == scenario["input"]["toolId"]
            and item["status"] == scenario["input"]["status"]
            and item["summary"] == scenario["input"]["summary"],
        )
    elif kind == "thread.artifact.append":
        record = wait_for_artifact(
            thread_id,
            lambda item: item["path"] == scenario["input"]["path"] and item["kind"] == scenario["input"]["kind"],
        )
    elif kind == "thread.runtimeflag.set":
        record = wait_for_runtime_flag(
            thread_id,
            lambda item: item["key"] == scenario["input"]["key"] and item["value"] == scenario["input"]["value"],
        )
    else:
        raise AssertionError(f"unsupported thread mutation scenario kind: {kind}")

    return {
        "task": completed_task,
        "record": record,
        "visibility": scenario_result["visibility"],
    }


def run_mcp_execution_scenario(page, thread_id: str, scenario: dict) -> dict:
    scenario_result = run_direct_tool_scenario(page, thread_id, scenario)
    completed_task = scenario_result["task"]
    record = wait_for_tool_call(
        thread_id,
        lambda item: item["toolId"] == scenario["kind"]
        and item["status"] == "completed"
        and scenario["summary_contains"] in item["summary"],
    )
    return {
        "task": completed_task,
        "record": record,
        "visibility": scenario_result["visibility"],
        "serverId": scenario["input"]["serverId"],
        "toolName": scenario["input"]["toolName"],
        "transport": "stdio-fixture",
    }


def wait_for_new_assistant_message(thread_id: str, baseline_count: int, timeout_seconds: float = 60.0) -> dict:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        items = api("GET", f"/api/threads/{thread_id}/messages")["items"]
        assistant_messages = [item for item in items if item.get("role") == "assistant"]
        if len(assistant_messages) > baseline_count:
            return assistant_messages[-1]
        time.sleep(0.5)
    raise RuntimeError("timed out waiting for a new assistant message")


def assert_required_child_sequence(child_kinds: list[str], expected_sequence: list[set[str]]):
    scan_index = 0
    for expected_group in expected_sequence:
        matched = False
        while scan_index < len(child_kinds):
            if child_kinds[scan_index] in expected_group:
                matched = True
                scan_index += 1
                break
            scan_index += 1
        if not matched:
            raise AssertionError(
                f"expected child task sequence {expected_sequence!r}, got {child_kinds!r}"
            )


def run_agent_scenario(page, thread_id: str, scenario: dict) -> dict:
    title = f"{scenario['title']} {scenario['runId']}"
    baseline_assistant_messages = len(
        [item for item in api("GET", f"/api/threads/{thread_id}/messages")["items"] if item.get("role") == "assistant"]
    )
    created_task = create_task(
        thread_id,
        title,
        "agent.run",
        {
            "goal": scenario["goal"],
            "maxSteps": scenario.get("maxSteps", 4),
        },
    )
    task_id = created_task["id"]
    expected_plan_mode = scenario["expectedPlanMode"]
    if created_task.get("agentPlanMode") != expected_plan_mode:
        raise AssertionError(
            f"expected created agentPlanMode={expected_plan_mode}, got {created_task.get('agentPlanMode')!r}"
        )
    run_task(thread_id, task_id)

    parent_task = wait_for_task_terminal(thread_id, task_id, timeout_seconds=60.0)
    if parent_task.get("status") != "completed":
        raise AssertionError(
            f"agent scenario {expected_plan_mode} failed: {parent_task.get('resultSummary', '')}"
        )
    if parent_task.get("agentPlanMode") != expected_plan_mode:
        raise AssertionError(
            f"expected completed agent task plan mode {expected_plan_mode}, got {parent_task.get('agentPlanMode')!r}"
        )
    child_tasks = api("GET", f"/api/threads/{thread_id}/tasks")["items"]
    completed_children = [
        item
        for item in child_tasks
        if item.get("parentTaskId") == task_id and item.get("status") == "completed"
    ]
    if len(completed_children) < 1:
        raise AssertionError("expected at least 1 completed child task")

    child_kinds = [item["kind"] for item in completed_children]
    assert_required_child_sequence(child_kinds, scenario["expectedChildSequence"])
    required_kind_names = set().union(*scenario["expectedChildSequence"])
    visible_children = [item for item in completed_children if item["kind"] in required_kind_names]
    required_observed_kinds = scenario.get("requiredObservedKinds", [])
    for required_kind in required_observed_kinds:
        if required_kind not in child_kinds:
            raise AssertionError(f"expected agent scenario to observe child kind {required_kind}, got {child_kinds!r}")

    latest_message = wait_for_new_assistant_message(thread_id, baseline_assistant_messages, timeout_seconds=30.0)

    refresh_thread_view(page, thread_id)

    try:
        assert_article_contains(page, title, timeout=8000)
    except Exception:
        refresh_thread_view(page, thread_id)
    child_visible = False
    child_visibility_mode = ""
    for child in visible_children[: min(3, len(visible_children))]:
        try:
            assert_article_contains(page, child["title"], child["resultSummary"], timeout=10000)
            child_visible = True
            child_visibility_mode = "title+summary"
        except Exception:
            fallback_signal = child["kind"].replace("workspace.", "").replace("_", " ")
            try:
                assert_article_contains(page, child["title"], fallback_signal, timeout=10000)
                child_visible = True
                child_visibility_mode = "title+kind"
            except Exception:
                try:
                    assert_article_contains(page, child["title"], timeout=10000)
                    child_visible = True
                    child_visibility_mode = "title-only"
                except Exception:
                    continue
    if not child_visible:
        refresh_thread_view(page, thread_id)
        for child in visible_children[: min(3, len(visible_children))]:
            try:
                assert_article_contains(page, child["title"], timeout=8000)
                child_visible = True
                child_visibility_mode = "title-only-after-refresh"
                break
            except Exception:
                continue
    if not child_visible and expected_plan_mode not in {"stat_then_read", "list_then_read"}:
        raise AssertionError(f"expected at least one child task indicator in UI for plan mode {expected_plan_mode}")
    try:
        expect(page.locator("article").filter(has_text=latest_message["content"]).first).to_be_visible(timeout=12000)
    except Exception:
        refresh_thread_view(page, thread_id)
        expect(page.locator("article").filter(has_text=latest_message["content"]).first).to_be_visible(timeout=12000)

    return {
        "task": parent_task,
        "childTasks": visible_children,
        "message": latest_message,
        "visibility": {
            "parentVisible": True,
            "childVisible": child_visible,
            "childVisibilityMode": child_visibility_mode,
            "assistantMessageVisible": True,
        },
    }


def run_recovery_continuation_scenario(page, thread_id: str, run_id: str) -> dict:
    title = f"Agent recovery continuation {run_id}"
    baseline_assistant_messages = len(
        [item for item in api("GET", f"/api/threads/{thread_id}/messages")["items"] if item.get("role") == "assistant"]
    )
    created_task = create_task(
        thread_id,
        title,
        "agent.run",
        {
            "goal": (
                "Apply a patch to playwright-recovery-note.txt that adds one line saying recovery evidence, "
                "then answer in one short sentence about what you changed. Do not do any extra work."
            ),
            "maxSteps": 4,
        },
    )
    task_id = created_task["id"]
    run_task(thread_id, task_id)

    waiting_parent = wait_for_task(
        thread_id,
        lambda item: item["id"] == task_id and item["status"] == "waiting_for_approval",
    )
    child_tasks = api("GET", f"/api/threads/{thread_id}/tasks")["items"]
    waiting_child = next(
        (
            item
            for item in child_tasks
            if item.get("parentTaskId") == task_id and item.get("status") == "needs_approval"
        ),
        None,
    )
    if waiting_child is None:
        raise AssertionError("expected a child task requiring approval for recovery continuation scenario")
    if waiting_parent.get("waitingStatus") != "waiting_for_approval":
        raise AssertionError(
            f"expected waitingStatus waiting_for_approval, got {waiting_parent.get('waitingStatus')!r}"
        )
    if waiting_parent.get("latestChildTaskId") != waiting_child["id"]:
        raise AssertionError(
            f"expected latestChildTaskId {waiting_child['id']!r}, got {waiting_parent.get('latestChildTaskId')!r}"
        )

    refresh_thread_view(page, thread_id)
    parentVisibleAfterRefresh = False
    waitingIndicatorVisible = False
    latestAgentCardVisible = False
    latestChildCardVisible = False
    try:
        assert_article_contains(page, title, timeout=10000)
        parentVisibleAfterRefresh = True
        waitingIndicatorVisible = page.locator("article").filter(has_text=title).filter(has_text="等待审批").first.is_visible()
    except Exception:
        refresh_thread_view(page, thread_id)
        assert_article_contains(page, title, timeout=10000)
        parentVisibleAfterRefresh = True
        waitingIndicatorVisible = page.locator("article").filter(has_text=title).filter(has_text="等待审批").first.is_visible()

    latest_agent_card = page.get_by_test_id("latest-agent-card")
    expect(latest_agent_card).to_contain_text(title, timeout=15000)
    expect(latest_agent_card).to_contain_text("等待审批", timeout=15000)
    latestAgentCardVisible = latest_agent_card.is_visible()

    latest_child_card = page.get_by_test_id("latest-child-task-card")
    expect(latest_child_card).to_contain_text(waiting_child["title"], timeout=15000)
    expect(latest_child_card).to_contain_text(waiting_child["id"], timeout=15000)
    latestChildCardVisible = latest_child_card.is_visible()

    pending_child_card = page.locator("article").filter(has_text=waiting_child["title"]).filter(has_text="approval required").first
    expect(pending_child_card).to_be_visible(timeout=15000)

    api("POST", f"/api/threads/{thread_id}/tasks/{waiting_child['id']}/approve", {})
    approved_child = wait_for_task_terminal(thread_id, waiting_child["id"], timeout_seconds=60.0)
    if approved_child.get("status") != "completed":
        raise AssertionError(f"expected approved child status completed, got {approved_child.get('status')!r}")

    resumed_parent = wait_for_task_terminal(thread_id, task_id, timeout_seconds=60.0)
    if resumed_parent.get("status") != "completed":
        raise AssertionError(f"expected resumed parent completed, got {resumed_parent.get('status')!r}")

    latest_message = wait_for_new_assistant_message(thread_id, baseline_assistant_messages, timeout_seconds=30.0)
    expect(page.locator("article").filter(has_text=latest_message["content"]).first).to_be_visible(timeout=12000)
    expect(latest_agent_card).to_contain_text("已完成", timeout=15000)
    expect(latest_agent_card).to_contain_text("Agent 已完成本轮闭环", timeout=15000)
    expect(page.get_by_test_id("latest-message-card")).to_contain_text(latest_message["content"], timeout=15000)

    return {
        "parentTaskId": task_id,
        "parentTitle": title,
        "parentStatusBeforeResume": waiting_parent["status"],
        "waitingStatus": waiting_parent.get("waitingStatus", ""),
        "latestChildTaskId": waiting_parent.get("latestChildTaskId", ""),
        "childTaskId": waiting_child["id"],
        "childStatusBeforeResume": waiting_child["status"],
        "parentStatusAfterResume": resumed_parent["status"],
        "resultSummary": resumed_parent.get("resultSummary", ""),
        "parentVisibleAfterRefresh": parentVisibleAfterRefresh,
        "waitingIndicatorVisible": waitingIndicatorVisible,
        "latestAgentCardVisible": latestAgentCardVisible,
        "latestChildCardVisible": latestChildCardVisible,
        "messageId": latest_message["id"],
    }


def run_agent_approval_rejected_scenario(page, thread_id: str, run_id: str) -> dict:
    title = f"Agent approval rejected {run_id}"
    created_task = create_task(
        thread_id,
        title,
        "agent.run",
        {
            "goal": (
                "Apply a patch to approval-rejected-note.txt that adds one line saying approval rejected evidence, "
                "then answer in one short sentence about what changed. Do not do any extra work."
            ),
            "maxSteps": 4,
        },
    )
    task_id = created_task["id"]
    run_task(thread_id, task_id)

    waiting_parent = wait_for_task(
        thread_id,
        lambda item: item["id"] == task_id and item["status"] == "waiting_for_approval",
    )
    waiting_child = find_latest_child_task(thread_id, task_id, "needs_approval")

    refresh_thread_view(page, thread_id)
    assert_article_contains(page, title, timeout=10000)

    latest_agent_card = page.get_by_test_id("latest-agent-card")
    expect(latest_agent_card).to_contain_text(title, timeout=15000)
    expect(latest_agent_card).to_contain_text("等待审批", timeout=15000)

    latest_child_card = page.get_by_test_id("latest-child-task-card")
    expect(latest_child_card).to_contain_text(waiting_child["title"], timeout=15000)
    expect(latest_child_card).to_contain_text(waiting_child["id"], timeout=15000)

    pending_child_card = page.locator("article").filter(has_text=waiting_child["title"]).filter(has_text="approval required").first
    expect(pending_child_card).to_be_visible(timeout=15000)

    rejected_child = reject_task(thread_id, waiting_child["id"])
    if rejected_child.get("status") != "failed":
        raise AssertionError(f"expected rejected child status failed, got {rejected_child.get('status')!r}")

    failed_parent = wait_for_task_terminal(thread_id, task_id, timeout_seconds=60.0)
    if failed_parent.get("status") != "failed":
        raise AssertionError(f"expected approval rejected parent failed, got {failed_parent.get('status')!r}")
    if not failed_parent.get("resultSummary", "").startswith("agent failed: child approval rejected:"):
        raise AssertionError(
            f"expected approval rejected summary prefix, got {failed_parent.get('resultSummary', '')!r}"
        )

    refresh_thread_view(page, thread_id)
    assert_article_contains(page, title, timeout=10000)
    expect(latest_agent_card).to_contain_text("状态：子任务审批已拒绝", timeout=15000)
    expect(latest_child_card).to_contain_text("approval rejected:", timeout=15000)

    return {
        "parentTaskId": task_id,
        "parentTitle": title,
        "parentStatusBeforeReject": waiting_parent["status"],
        "childTaskId": waiting_child["id"],
        "childStatusBeforeReject": waiting_child["status"],
        "childStatusAfterReject": rejected_child["status"],
        "parentStatusAfterReject": failed_parent["status"],
        "resultSummary": failed_parent.get("resultSummary", ""),
        "latestAgentCardVisible": latest_agent_card.is_visible(),
        "latestChildCardVisible": latest_child_card.is_visible(),
    }


def run_agent_child_task_failed_scenario(page, run_id: str) -> dict:
    thread_name = f"Playwright Agent Child Failed {run_id}"
    created_thread = create_thread(thread_name, permission_mode="read-only")
    thread_id = created_thread["id"]
    activate_thread_in_ui(page, thread_id)

    title = f"Agent child failed {run_id}"
    created_task = create_task(
        thread_id,
        title,
        "agent.run",
        {
            "goal": (
                "Apply a patch to child-task-failed-note.txt that adds one line saying child task failed evidence, "
                "then answer in one short sentence about what changed. Do not do any extra work."
            ),
            "maxSteps": 4,
        },
    )
    task_id = created_task["id"]
    run_task(thread_id, task_id)

    failed_parent = wait_for_task_terminal(thread_id, task_id, timeout_seconds=60.0)
    if failed_parent.get("status") != "failed":
        raise AssertionError(f"expected child task failed parent failed, got {failed_parent.get('status')!r}")
    if not failed_parent.get("resultSummary", "").startswith("agent child task failed:"):
        raise AssertionError(
            f"expected child task failed summary prefix, got {failed_parent.get('resultSummary', '')!r}"
        )

    failed_child = find_latest_child_task(thread_id, task_id, "failed")
    if failed_child.get("kind") != "workspace.apply_patch":
        raise AssertionError(f"expected failed child kind workspace.apply_patch, got {failed_child.get('kind')!r}")

    refresh_thread_view(page, thread_id)
    assert_article_contains(page, title, timeout=10000)

    latest_agent_card = page.get_by_test_id("latest-agent-card")
    expect(latest_agent_card).to_contain_text(title, timeout=15000)
    expect(latest_agent_card).to_contain_text("状态：子任务失败", timeout=15000)
    expect(latest_agent_card).to_contain_text("permission denied", timeout=15000)

    latest_child_card = page.get_by_test_id("latest-child-task-card")
    expect(latest_child_card).to_contain_text(failed_child["title"], timeout=15000)
    expect(latest_child_card).to_contain_text(failed_child["id"], timeout=15000)

    return {
        "threadId": thread_id,
        "threadName": thread_name,
        "parentTaskId": task_id,
        "parentTitle": title,
        "childTaskId": failed_child["id"],
        "childKind": failed_child["kind"],
        "parentStatus": failed_parent["status"],
        "parentSummary": failed_parent.get("resultSummary", ""),
        "childStatus": failed_child["status"],
        "childSummary": failed_child.get("resultSummary", ""),
        "latestAgentCardVisible": latest_agent_card.is_visible(),
        "latestChildCardVisible": latest_child_card.is_visible(),
    }


def build_recovered_as_failed_evidence() -> dict:
    return {
        "lane": FALLBACK_EVIDENCE_MATRIX["recovered_as_failed"]["lane"],
        "mode": "evidence-only",
        "browserAutomation": "not attempted",
        "reason": (
            "recovered_as_failed is restart-dependent and remains backed by persisted desktop fallback evidence "
            "instead of the canonical remote browser gate"
        ),
        "resultSummaryPrefix": "agent recovery failed:",
        "workflowLabelContains": "recovered_as_failed",
        "evidenceTests": [
            "TestDesktopFallbackAgentRecoveredAsFailedPersistsAcrossRestart",
        ],
    }


def run_smoke_acceptance(page, runtime_status: dict, thread_id: str) -> dict:
    refresh_mode = summarize_refresh_mode(page)
    desktop_copy_runtime = assert_desktop_copy_and_runtime_lane(page, runtime_status, refresh_mode)
    return {
        "mode": "smoke",
        "runtimeSource": runtime_status.get("runtimeSource", ""),
        "runtimeTrust": runtime_status.get("runtimeTrust", ""),
        "canonicalRuntimeUrl": runtime_status.get("canonicalRuntimeUrl", ""),
        "uiBaseUrl": UI_BASE_URL,
        "apiBaseUrl": API_BASE_URL,
        "threadId": thread_id,
        "refreshMode": refresh_mode,
        "copyAndRuntimeConsistency": desktop_copy_runtime,
    }


def write_failure_screenshot(page) -> str | None:
    try:
      screenshot = page.screenshot(full_page=True)
      return write_png_artifact("desktop-smoke-failure.png", screenshot)
    except Exception:
      return None


def main() -> int:
    run_id = f"{int(time.time())}-{uuid.uuid4().hex[:8]}"
    thread_name = f"Playwright Acceptance {run_id}"
    task_title = f"Playwright Live Refresh {run_id}"
    patch_path = f"playwright-live-refresh-{run_id}.txt"
    rollback_task_title = "Rollback latest write execution"

    runtime_status = ensure_canonical_runtime()
    created_thread = create_thread(thread_name)
    thread_id = created_thread["id"]
    activate_thread(thread_id)

    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(headless=True)
        page = browser.new_page(viewport={"width": 1600, "height": 1000})
        try:
            page.add_init_script(
                f"""
                window.__GENCODE_RUNTIME_BASE_URL__ = {json.dumps(API_BASE_URL)};
                """
            )
            try:
                page.goto(UI_BASE_URL, wait_until="domcontentloaded", timeout=30000)
                page.wait_for_timeout(3000)
                wait_for_active_thread(page, thread_id)
            except Exception as exc:
                if isinstance(exc, PlaywrightTimeoutError) or "thread card was not rendered" in str(exc):
                    fail(
                        "desktop page did not reach a usable thread workbench state",
                        category="page-load-failed",
                        details={
                            "uiBaseUrl": UI_BASE_URL,
                            "threadId": thread_id,
                            "exceptionType": type(exc).__name__,
                            "exception": str(exc),
                        },
                    )
                raise
            if ACCEPTANCE_MODE == "smoke":
                result = {
                    "ok": True,
                    "acceptanceMode": "smoke",
                    "threadName": thread_name,
                    **run_smoke_acceptance(page, runtime_status, thread_id),
                }
                emit_release_baseline(result)
                write_json_artifact("desktop-smoke-summary.json", result)
                print(json.dumps(result, ensure_ascii=False))
                return 0

            refresh_mode = summarize_refresh_mode(page)
            desktop_copy_runtime = assert_desktop_copy_and_runtime_lane(page, runtime_status, refresh_mode)

            direct_results = []
            direct_results.append(
                run_direct_tool_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Stat go.mod {run_id}",
                        "kind": "workspace.stat_file",
                        "input": {"path": "go.mod"},
                        "summary_contains": "stat go.mod: file",
                    },
                )
            )
            direct_results.append(
                run_direct_tool_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Read go.mod direct {run_id}",
                        "kind": "workspace.read_file",
                        "input": {"path": "go.mod"},
                        "summary_contains": "read go.mod:",
                    },
                )
            )
            direct_results.append(
                run_direct_tool_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"List root files direct {run_id}",
                        "kind": "workspace.list_files",
                        "input": {"path": "."},
                        "summary_contains": "listed",
                    },
                )
            )
            direct_results.append(
                run_direct_tool_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Read batch root files {run_id}",
                        "kind": "workspace.read_files_batch",
                        "input": {"paths": ["go.mod", "AGENTS.md"]},
                        "summary_contains": "read 2 files",
                    },
                )
            )
            direct_results.append(
                run_direct_tool_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"List runner go files {run_id}",
                        "kind": "workspace.list_files_filtered",
                        "input": {"path": "internal/core/runner", "pattern": "*.go", "includeDirs": False},
                        "summary_contains": "listed",
                    },
                )
            )
            direct_results.append(
                run_direct_tool_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Search detailed stat kind {run_id}",
                        "kind": "workspace.search_text_detailed",
                        "input": {"query": "KindWorkspaceStat", "path": "internal/core/runner", "limit": 20},
                        "summary_contains": "detailed matches",
                    },
                )
            )
            mcp_execution_results = []
            mcp_execution_results.append(
                run_mcp_execution_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Invoke MCP echo {run_id}",
                        "kind": "mcp.tool.invoke",
                        "input": {"serverId": "external-fixture", "toolName": "echo", "arguments": {"message": "hello"}},
                        "summary_contains": "mcp tool external-fixture/echo executed",
                    },
                )
            )
            thread_mutation_results = []
            thread_mutation_results.append(
                run_thread_mutation_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Append tool call {run_id}",
                        "kind": "thread.toolcall.append",
                        "input": {"toolId": "workspace.read_file", "status": "completed", "summary": "read finished"},
                        "summary_contains": "tool call workspace.read_file appended",
                        "ui_optional": True,
                    },
                )
            )
            thread_mutation_results.append(
                run_thread_mutation_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Append artifact {run_id}",
                        "kind": "thread.artifact.append",
                        "input": {"path": "artifacts/acceptance-note.md", "kind": "markdown"},
                        "summary_contains": "artifact markdown appended",
                        "ui_optional": True,
                    },
                )
            )
            thread_mutation_results.append(
                run_thread_mutation_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Set runtime flag {run_id}",
                        "kind": "thread.runtimeflag.set",
                        "input": {"key": "acceptance.mode", "value": run_id},
                        "summary_contains": "runtime flag acceptance.mode updated",
                        "ui_optional": True,
                    },
                )
            )

            agent_results = []
            agent_results.append(
                run_agent_scenario(
                    page,
                    thread_id,
                    {
                        "title": "Agent second-batch acceptance",
                        "runId": run_id,
                        "goal": (
                            "Use list_files_filtered on internal/core/runner with pattern *.go, "
                            "then use read_files_batch on internal/core/runner/agent_loop.go and internal/core/runner/runner.go, "
                            "then answer in one short sentence about what files you inspected. Do not write files."
                        ),
                        "expectedPlanMode": "filter_then_read",
                        "expectedChildSequence": [
                            {"workspace.list_files_filtered"},
                            {"workspace.read_files_batch", "workspace.read_file"},
                        ],
                        "maxSteps": 4,
                    },
                )
            )
            agent_results.append(
                run_agent_scenario(
                    page,
                    thread_id,
                    {
                        "title": "Agent search detailed acceptance",
                        "runId": run_id,
                        "goal": (
                            "Use search_text in internal/core/runner for KindWorkspaceStat, "
                            "then use search_text_detailed in internal/core/runner for KindWorkspaceStat with line details, "
                            "then answer in one short sentence about where it appears. Do not write files."
                        ),
                        "expectedPlanMode": "search_then_detailed",
                        "expectedChildSequence": [
                            {"workspace.search_text"},
                            {"workspace.search_text_detailed"},
                        ],
                        "maxSteps": 4,
                    },
                )
            )
            agent_results.append(
                run_agent_scenario(
                    page,
                    thread_id,
                    {
                        "title": "Agent stat read acceptance",
                        "runId": run_id,
                        "goal": (
                            "Use stat_file on go.mod to confirm it exists and inspect metadata, "
                            "then use read_files_batch on go.mod to read the content, "
                            "then answer in one short sentence about what you found. Do not write files."
                        ),
                        "expectedPlanMode": "stat_then_read",
                        "expectedChildSequence": [
                            {"workspace.stat_file"},
                            {"workspace.read_files_batch", "workspace.read_file"},
                        ],
                        "maxSteps": 4,
                    },
                )
            )
            agent_results.append(
                run_agent_scenario(
                    page,
                    thread_id,
                    {
                        "title": "Agent list read acceptance",
                        "runId": run_id,
                        "goal": (
                            "Use list_files on internal/core/runner first, "
                            "then use read_file on internal/core/runner/agent_loop.go, "
                            "then answer in one short sentence about what you inspected. Do not write files."
                        ),
                        "expectedPlanMode": "list_then_read",
                        "expectedChildSequence": [
                            {"workspace.list_files"},
                            {"workspace.read_file", "workspace.read_files_batch"},
                        ],
                        "requiredObservedKinds": ["workspace.list_files", "workspace.read_file"],
                        "maxSteps": 4,
                    },
                )
            )
            recovery_result = run_recovery_continuation_scenario(page, thread_id, run_id)
            approval_rejected_result = run_agent_approval_rejected_scenario(page, thread_id, run_id)
            created_task = create_task(
                thread_id,
                task_title,
                "workspace.apply_patch",
                {
                    "path": patch_path,
                    "patch": (
                        "*** Begin Patch\n"
                        f"*** Add File: {patch_path}\n"
                        "+playwright live refresh\n"
                        "*** End Patch\n"
                    ),
                },
            )
            created_task_id = created_task["id"]

            pending_card = page.locator("article").filter(has_text=task_title).filter(has_text="approval required").first
            expect(pending_card).to_be_visible(timeout=15000)
            approval_visible_before_approve = pending_card.is_visible()

            api("POST", f"/api/threads/{thread_id}/tasks/{created_task_id}/approve", {})
            approved_task = wait_for_task_terminal(thread_id, created_task_id, timeout_seconds=60.0)

            apply_execution = wait_for_write_execution(
                thread_id,
                lambda item: item["taskId"] == created_task_id and item["operation"] == "apply" and item["status"] == "completed",
            )

            write_execution_panel = page.get_by_test_id("write-execution-panel")
            expect(write_execution_panel).to_contain_text("applied patch", timeout=15000)
            expect(write_execution_panel).to_contain_text(patch_path, timeout=15000)
            write_execution_apply_visible = True

            rollback_button = page.get_by_test_id(f'rollback-latest-{apply_execution["id"]}')
            expect(rollback_button).to_be_visible(timeout=15000)
            expect(rollback_button).to_be_enabled(timeout=15000)
            rollback_button.click()

            rollback_pending = page.locator("article").filter(has_text=rollback_task_title).filter(has_text="approval required for rollback").first
            expect(rollback_pending).to_be_visible(timeout=15000)
            expect(rollback_pending).to_contain_text(patch_path, timeout=15000)
            rollback_approval_visible = rollback_pending.is_visible()

            rollback_task = wait_for_task(
                thread_id,
                lambda item: item["title"] == rollback_task_title
                and item["kind"] == "workspace.apply_patch.rollback"
                and apply_execution["id"] in item["inputSummary"],
            )
            rollback_task_id = rollback_task["id"]

            api("POST", f"/api/threads/{thread_id}/tasks/{rollback_task_id}/approve", {})
            rollback_approved = wait_for_task_terminal(thread_id, rollback_task_id, timeout_seconds=60.0)

            rollback_execution = wait_for_write_execution(
                thread_id,
                lambda item: item["taskId"] == rollback_task_id and item["operation"] == "rollback" and item["status"] == "completed",
            )

            rollback_completed = page.locator("article").filter(has_text=rollback_task_title).first
            expect(rollback_completed).to_be_visible(timeout=15000)
            rollback_task_visible = rollback_completed.is_visible()

            expect(write_execution_panel).to_contain_text("rolled back patch", timeout=15000)
            expect(write_execution_panel).to_contain_text(patch_path, timeout=15000)
            write_execution_rollback_visible = True
            child_failed_result = run_agent_child_task_failed_scenario(page, run_id)
            recovered_as_failed_evidence = build_recovered_as_failed_evidence()

            acceptance_report = {
                "remote": {
                    "mode": "browser-live",
                    "runtimeSource": runtime_status.get("runtimeSource", ""),
                    "runtimeTrust": runtime_status.get("runtimeTrust", ""),
                    "canonicalRuntimeUrl": runtime_status.get("canonicalRuntimeUrl", ""),
                    "uiBaseUrl": UI_BASE_URL,
                    "apiBaseUrl": API_BASE_URL,
                    "refreshMode": refresh_mode,
                    "copyAndRuntimeConsistency": desktop_copy_runtime,
                    "parentChildVisibility": {
                        "agentScenarioCount": len(agent_results),
                        "allParentsVisible": all(item["visibility"]["parentVisible"] for item in agent_results),
                        "allAssistantMessagesVisible": all(
                            item["visibility"]["assistantMessageVisible"] for item in agent_results
                        ),
                        "childVisibleScenarioCount": sum(1 for item in agent_results if item["visibility"]["childVisible"]),
                        "scenarios": [
                            {
                                "taskId": item["task"]["id"],
                                "planMode": item["task"].get("agentPlanMode", ""),
                                "childVisible": item["visibility"]["childVisible"],
                                "childVisibilityMode": item["visibility"]["childVisibilityMode"],
                                "childTaskKinds": [child["kind"] for child in item["childTasks"]],
                            }
                            for item in agent_results
                        ],
                    },
                    "agentFailureMatrix": {
                        "definition": AGENT_FAILURE_MATRIX,
                        "successResumeBaseline": recovery_result,
                        "approvalRejected": approval_rejected_result,
                        "childTaskFailed": child_failed_result,
                    },
                    "recoveryContinuation": recovery_result,
                    "approvalVisibility": {
                        "applyApprovalVisible": approval_visible_before_approve,
                        "rollbackApprovalVisible": rollback_approval_visible,
                        "approvalPanelExpected": True,
                    },
                    "writeExecutionVisibility": {
                        "applyVisible": write_execution_apply_visible,
                        "rollbackVisible": write_execution_rollback_visible,
                        "rollbackTaskVisible": rollback_task_visible,
                        "panelTestId": "write-execution-panel",
                    },
                    "directToolVisibility": {
                        "scenarioCount": len(direct_results),
                        "visibleByTitleCount": sum(1 for item in direct_results if item["visibility"]["taskCardVisible"]),
                        "visibleByToolKindFallbackCount": sum(
                            1 for item in direct_results if item["visibility"]["toolKindVisible"]
                        ),
                    },
                    "mcpExecutionVisibility": {
                        "scenarioCount": len(mcp_execution_results),
                        "visibleByTitleCount": sum(
                            1 for item in mcp_execution_results if item["visibility"]["taskCardVisible"]
                        ),
                        "visibleByToolKindFallbackCount": sum(
                            1 for item in mcp_execution_results if item["visibility"]["toolKindVisible"]
                        ),
                    },
                },
                "fallback": {
                    "mode": "evidence-only",
                    "browserAutomation": "not attempted",
                    "reason": (
                        "desktop local-fallback remains evidence-only because it does not provide the same stable "
                        "browser automation and SSE lane as the canonical remote app-server path"
                    ),
                    "agentFailureEvidence": {
                        "definition": FALLBACK_EVIDENCE_MATRIX,
                        "recoveredAsFailed": recovered_as_failed_evidence,
                    },
                    "evidenceTests": [
                        "TestDesktopFallbackPersistsAcrossAppRestart",
                        "TestDesktopFallbackTaskSummariesKeepParentAndWaitingFields",
                        "TestDesktopFallbackAgentWaitingForApprovalPersistsAcrossRestart",
                        "TestDesktopFallbackAgentWaitingForTaskPersistsAcrossRestart",
                        "TestDesktopFallbackAgentRecoveredAsFailedPersistsAcrossRestart",
                        "TestDesktopFallbackWriteExecutionsPersistAcrossRestart",
                    ],
                },
            }

            result = {
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
                "directToolTasks": [
                    {
                        "id": item["task"]["id"],
                        "title": item["task"]["title"],
                        "kind": item["task"]["kind"],
                        "input": item["input"],
                        "resultSummary": item["task"]["resultSummary"],
                        "visibility": item["visibility"],
                    }
                    for item in direct_results
                ],
                "mcpExecutionResults": [
                    {
                        "taskId": item["task"]["id"],
                        "title": item["task"]["title"],
                        "kind": item["task"]["kind"],
                        "serverId": item["serverId"],
                        "toolName": item["toolName"],
                        "transport": item["transport"],
                        "resultSummary": item["task"]["resultSummary"],
                        "record": item["record"],
                        "visibility": item["visibility"],
                    }
                    for item in mcp_execution_results
                ],
                "threadMutationTasks": [
                    {
                        "taskId": item["task"]["id"],
                        "title": item["task"]["title"],
                        "kind": item["task"]["kind"],
                        "resultSummary": item["task"]["resultSummary"],
                        "record": item["record"],
                        "visibility": item["visibility"],
                    }
                    for item in thread_mutation_results
                ],
                "agentScenarios": [
                    {
                        "taskId": item["task"]["id"],
                        "title": item["task"]["title"],
                        "planMode": item["task"].get("agentPlanMode", ""),
                        "resultSummary": item["task"]["resultSummary"],
                        "childTaskKinds": [child["kind"] for child in item["childTasks"]],
                        "observedChildTaskKinds": [child["kind"] for child in item["childTasks"]],
                        "messageId": item["message"]["id"],
                        "message": item["message"]["content"],
                        "visibility": item["visibility"],
                    }
                    for item in agent_results
                ],
                "agentFailureMatrix": acceptance_report["remote"]["agentFailureMatrix"],
                "recoveryContinuation": recovery_result,
                "runtimeSource": runtime_status.get("runtimeSource", ""),
                "runtimeTrust": runtime_status.get("runtimeTrust", ""),
                "canonicalRuntimeUrl": runtime_status.get("canonicalRuntimeUrl", ""),
                "uiBaseUrl": UI_BASE_URL,
                "apiBaseUrl": API_BASE_URL,
                "acceptanceMode": "full",
                "refreshMode": refresh_mode,
                "fallbackEvidenceMode": acceptance_report["fallback"]["mode"],
                "acceptanceReport": acceptance_report,
            }
            emit_release_baseline(result)
            write_json_artifact("desktop-full-summary.json", result)
            print(json.dumps(result, ensure_ascii=False))
            return 0
        finally:
            browser.close()


def validate_release_baseline(result: dict) -> None:
    required_release_keys = [
        "runtimeSource",
        "runtimeTrust",
        "uiBaseUrl",
        "apiBaseUrl",
        "acceptanceMode",
        "refreshMode",
    ]
    missing = [key for key in required_release_keys if key not in result]
    if missing:
        raise AssertionError(f"missing release-baseline key(s): {', '.join(missing)}")

    if result.get("acceptanceMode") == "smoke":
        for key in ("copyAndRuntimeConsistency", "threadId"):
            if key not in result:
                raise AssertionError(f"missing smoke acceptance key: {key}")
        return

    for key in ("fallbackEvidenceMode", "acceptanceReport"):
        if key not in result:
            raise AssertionError(f"missing full acceptance key: {key}")

    acceptance_report = result.get("acceptanceReport") or {}
    required_acceptance_keys = [
        "remote",
        "fallback",
    ]
    missing_acceptance = [key for key in required_acceptance_keys if key not in acceptance_report]
    if missing_acceptance:
        raise AssertionError(f"missing acceptance-report key(s): {', '.join(missing_acceptance)}")

    remote_report = acceptance_report.get("remote") or {}
    for key in ("runtimeSource", "runtimeTrust", "uiBaseUrl", "apiBaseUrl", "refreshMode", "agentFailureMatrix"):
        if key not in remote_report:
            raise AssertionError(f"missing remote acceptance key: {key}")

    agent_failure_matrix = remote_report.get("agentFailureMatrix") or {}
    for key in (
        "definition",
        "successResumeBaseline",
        "approvalRejected",
        "childTaskFailed",
    ):
        if key not in agent_failure_matrix:
            raise AssertionError(f"missing agent failure matrix key: {key}")

    fallback_report = acceptance_report.get("fallback") or {}
    if fallback_report.get("mode") != "evidence-only":
        raise AssertionError("fallback evidence must remain evidence-only")
    if fallback_report.get("browserAutomation") != "not attempted":
        raise AssertionError("fallback lane must not be treated as browser automation acceptance")
    fallback_agent_evidence = fallback_report.get("agentFailureEvidence") or {}
    fallback_recovered = fallback_agent_evidence.get("recoveredAsFailed") or {}
    if fallback_recovered.get("mode") != "evidence-only":
        raise AssertionError("fallback recovered_as_failed evidence must remain evidence-only")


def emit_release_baseline(result: dict) -> None:
    validate_release_baseline(result)


if __name__ == "__main__":
    failure_page = None
    try:
        raise SystemExit(main())
    except Exception as exc:
        parsed = normalize_failure(exc)
        write_json_artifact("desktop-smoke-failure.json", parsed)
        if "page" in globals():
            try:
                write_failure_screenshot(globals()["page"])
            except Exception:
                pass
        print(json.dumps(parsed, ensure_ascii=False), file=sys.stderr)
        raise
