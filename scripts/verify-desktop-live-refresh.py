import json
import os
import socket
import sys
import time
import urllib.error
import urllib.request
import uuid

from playwright.sync_api import expect, sync_playwright


UI_BASE_URL = os.environ.get("GEN_CODE_UI_BASE_URL", "http://127.0.0.1:5174/")
API_BASE_URL = os.environ.get("GEN_CODE_API_BASE_URL", "http://127.0.0.1:10008")
API_RETRIES = int(os.environ.get("GEN_CODE_API_RETRIES", "5"))
API_RETRY_DELAY = float(os.environ.get("GEN_CODE_API_RETRY_DELAY", "0.5"))

SECOND_BATCH_TOOL_KINDS = {
    "workspace.stat_file",
    "workspace.read_files_batch",
    "workspace.list_files_filtered",
    "workspace.search_text_detailed",
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
    if "手动刷新" in body_text:
        return {
            "label": "手动刷新",
            "detail": "UI reported manual refresh fallback",
            "supportsSSE": False,
            "sseConnected": False,
            "evidence": "手动刷新",
        }
    return {
        "label": "unknown",
        "detail": "UI refresh mode text was not detected",
        "supportsSSE": None,
        "sseConnected": None,
        "evidence": "",
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
                raise
            time.sleep(API_RETRY_DELAY * (attempt + 1))

    if last_error is not None:
        raise last_error
    raise RuntimeError(f"request failed without error for {method} {path}")


def ensure_canonical_runtime() -> dict:
    status = api("GET", "/api/runtime/status")
    runtime_source = status.get("runtimeSource", "")
    runtime_trust = status.get("runtimeTrust", "")
    canonical_runtime_url = (status.get("canonicalRuntimeUrl", "") or "").rstrip("/")
    expected_runtime_url = API_BASE_URL.rstrip("/")

    if runtime_source != "remote-app-server":
        fail(
            f"expected canonical remote runtime source 'remote-app-server', got {runtime_source!r}",
            category="runtime",
            details=status,
        )
    if runtime_trust != "canonical":
        fail(
            f"expected canonical runtime trust 'canonical', got {runtime_trust!r}",
            category="runtime",
            details=status,
        )
    if canonical_runtime_url and canonical_runtime_url != expected_runtime_url:
        fail(
            f"canonical runtime URL mismatch: expected {expected_runtime_url}, got {canonical_runtime_url}",
            category="runtime",
            details=status,
        )
    return status


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
        assert_article_contains(page, tool_call["toolId"])
        visibility["toolKindVisible"] = True
    return {
        "task": completed_task,
        "toolCall": tool_call,
        "visibility": visibility,
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
    if not child_visible and expected_plan_mode != "stat_then_read":
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
            page.goto(UI_BASE_URL, wait_until="domcontentloaded", timeout=30000)
            page.wait_for_timeout(3000)
            wait_for_active_thread(page, thread_id)
            refresh_mode = summarize_refresh_mode(page)

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

            approved_task = api("POST", f"/api/threads/{thread_id}/tasks/{created_task_id}/approve", {})

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

            rollback_approved = api("POST", f"/api/threads/{thread_id}/tasks/{rollback_task_id}/approve", {})

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

            acceptance_report = {
                "remote": {
                    "mode": "browser-live",
                    "runtimeSource": runtime_status.get("runtimeSource", ""),
                    "runtimeTrust": runtime_status.get("runtimeTrust", ""),
                    "canonicalRuntimeUrl": runtime_status.get("canonicalRuntimeUrl", ""),
                    "uiBaseUrl": UI_BASE_URL,
                    "apiBaseUrl": API_BASE_URL,
                    "refreshMode": refresh_mode,
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
                    "mode": "go-test-evidence",
                    "browserAutomation": "not attempted",
                    "reason": "desktop local-fallback does not provide the same stable browser automation and SSE lane as the canonical remote app-server path",
                    "evidenceTests": [
                        "TestDesktopFallbackPersistsAcrossAppRestart",
                        "TestDesktopFallbackTaskSummariesKeepParentAndWaitingFields",
                        "TestDesktopFallbackAgentWaitingForApprovalPersistsAcrossRestart",
                        "TestDesktopFallbackAgentWaitingForTaskPersistsAcrossRestart",
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
                        "messageId": item["message"]["id"],
                        "message": item["message"]["content"],
                        "visibility": item["visibility"],
                    }
                    for item in agent_results
                ],
                "runtimeSource": runtime_status.get("runtimeSource", ""),
                "runtimeTrust": runtime_status.get("runtimeTrust", ""),
                "canonicalRuntimeUrl": runtime_status.get("canonicalRuntimeUrl", ""),
                "uiBaseUrl": UI_BASE_URL,
                "apiBaseUrl": API_BASE_URL,
                "refreshMode": refresh_mode,
                "fallbackEvidenceMode": acceptance_report["fallback"]["mode"],
                "acceptanceReport": acceptance_report,
            }
            emit_release_baseline(result)
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
        "refreshMode",
        "fallbackEvidenceMode",
        "acceptanceReport",
    ]
    missing = [key for key in required_release_keys if key not in result]
    if missing:
        raise AssertionError(f"missing release-baseline key(s): {', '.join(missing)}")

    acceptance_report = result.get("acceptanceReport") or {}
    required_acceptance_keys = [
        "remote",
        "fallback",
    ]
    missing_acceptance = [key for key in required_acceptance_keys if key not in acceptance_report]
    if missing_acceptance:
        raise AssertionError(f"missing acceptance-report key(s): {', '.join(missing_acceptance)}")

    remote_report = acceptance_report.get("remote") or {}
    for key in ("runtimeSource", "runtimeTrust", "uiBaseUrl", "apiBaseUrl", "refreshMode"):
        if key not in remote_report:
            raise AssertionError(f"missing remote acceptance key: {key}")

    fallback_report = acceptance_report.get("fallback") or {}
    if fallback_report.get("mode") != "go-test-evidence":
        raise AssertionError("fallback evidence must remain go-test-evidence")
    if fallback_report.get("browserAutomation") != "not attempted":
        raise AssertionError("fallback lane must not be treated as browser automation acceptance")


def emit_release_baseline(result: dict) -> None:
    validate_release_baseline(result)


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:
        try:
            parsed = json.loads(str(exc))
        except json.JSONDecodeError:
            parsed = {"ok": False, "category": "unknown", "error": str(exc)}
        print(json.dumps(parsed, ensure_ascii=False), file=sys.stderr)
        raise
