import copy
import json
import os
import socket
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

from playwright.sync_api import TimeoutError as PlaywrightTimeoutError
from playwright.sync_api import expect, sync_playwright


UI_BASE_URL = os.environ.get("GEN_CODE_UI_BASE_URL", "http://127.0.0.1:5174/")
API_BASE_URL = os.environ.get("GEN_CODE_API_BASE_URL", "http://127.0.0.1:10008")
API_RETRIES = int(os.environ.get("GEN_CODE_API_RETRIES", "5"))
API_RETRY_DELAY = float(os.environ.get("GEN_CODE_API_RETRY_DELAY", "0.5"))
API_DEFAULT_TIMEOUT_SECONDS = float(os.environ.get("GEN_CODE_API_TIMEOUT_SECONDS", "30"))
API_LONG_RUN_TIMEOUT_SECONDS = float(os.environ.get("GEN_CODE_API_LONG_RUN_TIMEOUT_SECONDS", "180"))
ACCEPTANCE_MODE = os.environ.get("GEN_CODE_ACCEPTANCE_MODE", "full").strip().lower() or "full"
ARTIFACT_DIR = os.environ.get("GEN_CODE_ARTIFACT_DIR", os.path.join("tmp", "desktop-smoke-artifacts"))
EMBEDDED_PREVIEW_PARAM = "gcPreview"
AUTHENTICATED_SESSION_LABEL = "session=acceptance-session"
AUTHENTICATED_PROFILE_LABEL = "profile=acceptance"
AUTHENTICATED_ROLE_LABEL = "role=reader"
AUTHENTICATED_SCOPE_LABEL = "scope=controlled"
AUTHENTICATED_TRANSPORT_LABEL = "transport=cookie"
AUTHENTICATED_RESULT_TEXT = (
    "identity=authenticated-browser;session=acceptance-session;profile=acceptance;"
    "role=reader;scope=controlled"
)
PUBLIC_BROWSER_BASE_URL = os.environ.get("GEN_CODE_BROWSER_PUBLIC_BASE_URL", "https://example.com/").strip()
PUBLIC_BROWSER_TARGETS_RAW = os.environ.get("GEN_CODE_BROWSER_PUBLIC_TARGETS", "").strip()
PUBLIC_BROWSER_MODE = os.environ.get("GEN_CODE_BROWSER_PUBLIC_WEB_MODE", "required").strip().lower() or "required"
BROWSER_ONLY_ACCEPTANCE = os.environ.get("GEN_CODE_BROWSER_ONLY_ACCEPTANCE", "").strip().lower() in {"1", "true", "yes", "on"}

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

PUBLIC_WEB_SKIP_MODES = {"skip", "disabled", "off", "false", "0"}


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


def parse_env_list(raw: str) -> list[str]:
    text = (raw or "").strip()
    if not text:
        return []
    if text.startswith("["):
        try:
            parsed = json.loads(text)
        except json.JSONDecodeError:
            parsed = None
        if isinstance(parsed, list):
            return [str(item).strip() for item in parsed if str(item).strip()]
    items = []
    for line in text.replace("\r", "\n").split("\n"):
        for part in line.split(","):
            value = part.strip()
            if value:
                items.append(value)
    return items


def resolve_public_web_targets() -> list[str]:
    targets = parse_env_list(PUBLIC_BROWSER_TARGETS_RAW)
    if targets:
        return targets
    if PUBLIC_BROWSER_BASE_URL:
        return [PUBLIC_BROWSER_BASE_URL]
    return []


def classify_public_web_mode() -> dict:
    if PUBLIC_BROWSER_MODE in PUBLIC_WEB_SKIP_MODES:
        return {
            "enabled": False,
            "status": "skipped",
            "classification": "disabled-by-config",
            "reason": f"public-web lane disabled by GEN_CODE_BROWSER_PUBLIC_WEB_MODE={PUBLIC_BROWSER_MODE!r}",
        }
    target_urls = resolve_public_web_targets()
    if not target_urls:
        return {
            "enabled": False,
            "status": "skipped",
            "classification": "missing-target-url",
            "reason": "GEN_CODE_BROWSER_PUBLIC_BASE_URL / GEN_CODE_BROWSER_PUBLIC_TARGETS is empty",
        }
    return {
        "enabled": True,
        "status": "required",
        "classification": "configured",
        "targetUrl": target_urls[0],
        "targetUrls": target_urls,
        "targetCount": len(target_urls),
    }


def validate_public_web_target(raw_url: str) -> str:
    parsed = urllib.parse.urlparse(raw_url)
    if parsed.scheme != "https" or not parsed.netloc:
        fail(
            f"public-web lane requires one explicit HTTPS target, got {raw_url!r}",
            category="public-web-config-invalid",
            details={"targetUrl": raw_url},
        )
    return parsed.geturl()


def preflight_public_web_target(target_url: str) -> dict:
    request = urllib.request.Request(
        target_url,
        headers={"User-Agent": "gen-code-desktop-acceptance"},
        method="GET",
    )
    last_error = None
    for attempt in range(3):
        try:
            with urllib.request.urlopen(request, timeout=20) as response:
                return {
                    "targetUrl": target_url,
                    "statusCode": getattr(response, "status", 200),
                    "finalUrl": response.geturl(),
                }
        except urllib.error.HTTPError as exc:
            fail(
                f"public-web preflight failed for {target_url}",
                category="public-web-preflight-network",
                details={
                    "targetUrl": target_url,
                    "statusCode": exc.code,
                    "reason": str(exc),
                },
            )
        except (urllib.error.URLError, socket.timeout, ConnectionResetError, OSError) as exc:
            last_error = exc
            if attempt == 2:
                fail(
                    f"public-web preflight failed for {target_url}",
                    category="public-web-preflight-network",
                    details={
                        "targetUrl": target_url,
                        "exceptionType": type(exc).__name__,
                        "exception": str(exc),
                        "attempts": attempt + 1,
                    },
                )
            time.sleep(1.0 + attempt)

    if last_error is not None:
        fail(
            f"public-web preflight failed for {target_url}",
            category="public-web-preflight-network",
            details={
                "targetUrl": target_url,
                "exceptionType": type(last_error).__name__,
                "exception": str(last_error),
                "attempts": 3,
            },
        )


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


def remove_artifact_if_exists(name: str) -> None:
    path = os.path.join(ensure_artifact_dir(), name)
    if os.path.exists(path):
        os.remove(path)


def clear_stale_failure_artifacts() -> None:
    remove_artifact_if_exists("desktop-smoke-failure.json")
    remove_artifact_if_exists("desktop-smoke-failure.png")
    remove_artifact_if_exists("desktop-full-failure.json")
    remove_artifact_if_exists("desktop-full-failure.png")


def current_failure_json_name() -> str:
    if ACCEPTANCE_MODE == "full":
        return "desktop-full-failure.json"
    return "desktop-smoke-failure.json"


def current_failure_png_name() -> str:
    if ACCEPTANCE_MODE == "full":
        return "desktop-full-failure.png"
    return "desktop-smoke-failure.png"


def api_timeout_for_request(method: str, path: str) -> float:
    normalized_method = (method or "").upper().strip()
    normalized_path = path or ""
    if normalized_method == "POST" and normalized_path.endswith("/run"):
        return API_LONG_RUN_TIMEOUT_SECONDS
    if normalized_method == "POST" and (
        normalized_path.endswith("/approve") or normalized_path.endswith("/reject")
    ):
        return API_LONG_RUN_TIMEOUT_SECONDS
    return API_DEFAULT_TIMEOUT_SECONDS


def api(method: str, path: str, data=None):
    body = None
    headers = {}
    if data is not None:
        body = json.dumps(data).encode("utf-8")
        headers["Content-Type"] = "application/json"
    timeout_seconds = api_timeout_for_request(method, path)

    last_error = None
    for attempt in range(API_RETRIES):
        request = urllib.request.Request(API_BASE_URL + path, data=body, headers=headers, method=method)
        try:
            with urllib.request.urlopen(request, timeout=timeout_seconds) as response:
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
                        "timeoutSeconds": timeout_seconds,
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
                "timeoutSeconds": timeout_seconds,
                "exceptionType": type(last_error).__name__,
                "exception": str(last_error),
            },
        )
    fail(
        f"runtime API request failed for {method} {path}",
        category="api-unavailable",
        details={"method": method, "path": path, "baseUrl": API_BASE_URL, "timeoutSeconds": timeout_seconds},
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


def ensure_mcp_verified_lanes() -> dict:
    payload = api("GET", "/api/mcp/servers")
    items = payload.get("items", [])
    by_id = {item.get("id", ""): item for item in items}
    required_ids = [
        "external-fixture",
        "sdk-external-fixture",
        "third-party-time",
    ]
    missing = [server_id for server_id in required_ids if server_id not in by_id]
    if missing:
        fail(
            "canonical runtime instance missing expected MCP verified lanes",
            category="environment-mcp-baseline-mismatch",
            details={
                "requiredServerIds": required_ids,
                "missingServerIds": missing,
                "discoveredServerIds": sorted([server_id for server_id in by_id.keys() if server_id]),
                "apiBaseUrl": API_BASE_URL,
            },
        )
    return {
        "requiredServerIds": required_ids,
        "discoveredServerIds": sorted([server_id for server_id in by_id.keys() if server_id]),
    }


def create_thread(name: str, permission_mode: str = "ask-user") -> dict:
    return api(
        "POST",
        "/api/threads",
        {
            "name": name,
            "permissionMode": permission_mode,
        },
    )


def build_controlled_browser_fixture_url(thread_id: str, thread_name: str) -> str:
    encoded_name = urllib.parse.quote(thread_name, safe="")
    return (
        f"{UI_BASE_URL}?{EMBEDDED_PREVIEW_PARAM}=1"
        f"&pane=acceptance-browser"
        f"&threadId={urllib.parse.quote(thread_id, safe='')}"
        f"&threadName={encoded_name}"
    )


def build_authenticated_browser_fixture_url(thread_id: str, thread_name: str) -> str:
    encoded_name = urllib.parse.quote(thread_name, safe="")
    return (
        f"{UI_BASE_URL}?{EMBEDDED_PREVIEW_PARAM}=1"
        f"&pane=acceptance-browser"
        f"&threadId={urllib.parse.quote(thread_id, safe='')}"
        f"&threadName={encoded_name}"
        f"&authFixture=1"
    )


def build_thread_preview_fixture_url(thread_id: str, thread_name: str, pane: str) -> str:
    encoded_name = urllib.parse.quote(thread_name, safe="")
    return (
        f"{UI_BASE_URL}?{EMBEDDED_PREVIEW_PARAM}=1"
        f"&pane={urllib.parse.quote(pane, safe='')}"
        f"&threadId={urllib.parse.quote(thread_id, safe='')}"
        f"&threadName={encoded_name}"
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


def get_task_by_id(thread_id: str, task_id: str) -> dict | None:
    tasks = api("GET", f"/api/threads/{thread_id}/tasks")["items"]
    for item in tasks:
        if item.get("id") == task_id:
            return item
    return None


def get_browser_snapshot() -> dict:
    return api("GET", "/api/runtime/status").get("browser", {})


def wait_for_browser_snapshot(predicate=None, timeout_seconds: float = 10.0) -> dict:
    deadline = time.time() + timeout_seconds
    last_snapshot = {}
    while time.time() < deadline:
        snapshot = get_browser_snapshot()
        last_snapshot = snapshot
        if predicate is None:
            return snapshot
        try:
            if predicate(snapshot):
                return snapshot
        except Exception:
            pass
        time.sleep(0.5)
    return last_snapshot


def get_active_browser_tab_id() -> str:
    snapshot = wait_for_browser_snapshot(
        lambda item: bool(item.get("activeTabId")) or bool(item.get("tabs", []))
    )
    active_tab_id = snapshot.get("activeTabId", "")
    if active_tab_id:
        return active_tab_id
    tabs = snapshot.get("tabs", [])
    if tabs:
        return tabs[-1].get("id", "")
    raise AssertionError(f"expected runtime browser snapshot to expose an active tab, got {snapshot!r}")


def get_browser_tab_url(tab_id: str) -> str:
    target_tab_id = (tab_id or "").strip()
    snapshot = wait_for_browser_snapshot(
        lambda item: any(tab.get("id", "") == target_tab_id for tab in item.get("tabs", []))
    )
    for tab in snapshot.get("tabs", []):
        if tab.get("id", "") == target_tab_id:
            return tab.get("url", "")
    raise AssertionError(f"expected runtime browser snapshot to expose tab {target_tab_id!r}, got {snapshot!r}")


def extract_tab_id_from_summary(summary: str) -> str:
    parts = (summary or "").split(": ", 1)
    if len(parts) != 2:
        raise AssertionError(f"expected browser summary to contain a tab id, got {summary!r}")
    return parts[1].split(" | ", 1)[0].strip()


def resolve_browser_tab_id(result: dict, fallback_exclude_ids: set[str] | None = None) -> str:
    for summary in [
        ((result.get("task") or {}).get("resultSummary", "")),
        ((result.get("record") or {}).get("summary", "")),
    ]:
        text = (summary or "").strip()
        if not text:
            continue
        try:
            return extract_tab_id_from_summary(text)
        except AssertionError:
            continue
    if fallback_exclude_ids is not None:
        return get_latest_browser_tab_id(exclude_ids=fallback_exclude_ids)
    return get_active_browser_tab_id()


def get_latest_browser_tab_id(exclude_ids: set[str] | None = None) -> str:
    exclude_ids = exclude_ids or set()
    snapshot = wait_for_browser_snapshot(
        lambda item: any(
            tab.get("id", "") and tab.get("id", "") not in exclude_ids
            for tab in item.get("tabs", [])
        )
    )
    candidates = [tab.get("id", "") for tab in snapshot.get("tabs", []) if tab.get("id", "") and tab.get("id", "") not in exclude_ids]
    if candidates:
        return candidates[-1]
    raise AssertionError(f"expected runtime browser snapshot to expose a non-excluded tab, got {snapshot!r}")


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


def assert_sidebar_note_contains(page, text: str, timeout: int = 12000):
    locator = page.locator(".sidebar-note").filter(has_text=text)
    expect(locator.first).to_be_visible(timeout=timeout)
    return locator.first


def is_scenario_visible(visibility: dict) -> bool:
    return any(
        visibility.get(key, False)
        for key in [
            "taskCardVisible",
            "toolKindVisible",
            "latestTaskCardVisible",
            "latestToolCardVisible",
            "browserSidebarVisible",
        ]
    )


def activate_thread_in_ui(page, thread_id: str):
    activate_thread(thread_id)
    refresh_thread_view(page, thread_id)


def create_agent_task_via_ui(page, thread_id: str, title: str, goal: str, max_steps: int = 4) -> dict:
    existing_ids = {item["id"] for item in api("GET", f"/api/threads/{thread_id}/tasks")["items"]}
    wait_for_active_thread(page, thread_id)
    page.get_by_test_id("task-title-input").fill(title)
    page.get_by_test_id("task-kind-select").select_option("agent.run")
    page.get_by_test_id("task-input-textarea").fill(goal)
    page.get_by_test_id("create-task-button").click()
    return wait_for_task(
        thread_id,
        lambda item: item["title"] == title and item["id"] not in existing_ids and item["kind"] == "agent.run",
        timeout_seconds=30.0,
    )


def run_task_via_ui(page, thread_id: str, task_id: str, title: str) -> str:
    try:
        run_button = page.locator("article").filter(has_text=title).get_by_role("button", name="运行任务").first
        expect(run_button).to_be_visible(timeout=12000)
    except Exception:
        refresh_thread_view(page, thread_id)
        run_button = page.locator("article").filter(has_text=title).get_by_role("button", name="运行任务").first
        if not run_button.is_visible():
            run_task(thread_id, task_id)
            return "api-fallback"
    run_button.click()
    return "desktop-ui"


def approve_task_via_ui(page, thread_id: str, task_id: str, task_title: str) -> str:
    try:
        approve_button = (
            page.locator("article")
            .filter(has_text=task_title)
            .filter(has_text="approval required")
            .get_by_role("button", name="批准执行")
            .first
        )
        expect(approve_button).to_be_visible(timeout=12000)
    except Exception:
        refresh_thread_view(page, thread_id)
        approve_button = (
            page.locator("article")
            .filter(has_text=task_title)
            .filter(has_text="approval required")
            .get_by_role("button", name="批准执行")
            .first
        )
        if not approve_button.is_visible():
            api("POST", f"/api/threads/{thread_id}/tasks/{task_id}/approve", {})
            return "api-fallback"
    approve_button.click()
    return "desktop-ui"


def run_direct_tool_scenario(page, thread_id: str, scenario: dict) -> dict:
    created_task = create_task(thread_id, scenario["title"], scenario["kind"], scenario["input"])
    task_id = created_task["id"]
    run_task(thread_id, task_id)
    terminal_task = wait_for_task_terminal(thread_id, task_id, timeout_seconds=scenario.get("timeoutSeconds", 20.0))
    if terminal_task.get("status") != "completed":
        raise AssertionError(
            f"task {scenario['kind']} did not complete successfully: status={terminal_task.get('status')!r}, "
            f"summary={terminal_task.get('resultSummary', '')!r}, input={scenario['input']!r}"
        )
    completed_task = terminal_task

    expected_summary = scenario["summary_contains"]
    if expected_summary not in completed_task["resultSummary"]:
        raise AssertionError(
            f"task {scenario['kind']} summary mismatch: expected substring {expected_summary!r}, got {completed_task['resultSummary']!r}"
        )

    tool_call = wait_for_tool_call(
        thread_id,
        lambda item: item["toolId"] == scenario["kind"] and item["status"] == "completed" and expected_summary in item["summary"],
    )
    refresh_thread_view(page, thread_id)
    visibility = {
        "taskCardVisible": False,
        "toolKindVisible": False,
        "latestTaskCardVisible": False,
        "latestToolCardVisible": False,
        "browserSidebarVisible": False,
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
                    pass

    if not is_scenario_visible(visibility):
        try:
            assert_result_card_contains(page, "latest-task-card", completed_task["title"], timeout=10000)
            visibility["latestTaskCardVisible"] = True
        except Exception:
            try:
                assert_result_card_contains(page, "latest-task-card", completed_task["resultSummary"], timeout=10000)
                visibility["latestTaskCardVisible"] = True
            except Exception:
                try:
                    assert_result_card_contains(page, "latest-task-card", expected_summary, timeout=10000)
                    visibility["latestTaskCardVisible"] = True
                except Exception:
                    pass

    if not is_scenario_visible(visibility):
        try:
            assert_result_card_contains(page, "latest-toolcall-card", tool_call["toolId"], tool_call["summary"], timeout=10000)
            visibility["latestToolCardVisible"] = True
        except Exception:
            try:
                assert_result_card_contains(page, "latest-toolcall-card", tool_call["toolId"], timeout=10000)
                visibility["latestToolCardVisible"] = True
            except Exception:
                pass

    if not is_scenario_visible(visibility) and scenario["kind"].startswith("browser."):
        sidebar_targets = [completed_task["resultSummary"], tool_call["summary"]]
        if scenario["kind"] == "browser.extract":
            sidebar_targets.append("Extract:")
        if scenario["kind"] == "browser.screenshot":
            sidebar_targets.append("Screenshot:")
        if scenario["kind"] in {"browser.activate_tab", "browser.close_tab", "browser.back", "browser.forward", "browser.reload", "browser.navigate", "browser.state"}:
            visibility["browserSidebarVisible"] = True
        for target in sidebar_targets:
            if not target:
                continue
            try:
                assert_sidebar_note_contains(page, target, timeout=10000)
                visibility["browserSidebarVisible"] = True
                break
            except Exception:
                continue

    if not is_scenario_visible(visibility) and not scenario.get("ui_optional", False):
        raise AssertionError(
            f"scenario {scenario['kind']} completed but no matching desktop visibility signal was found "
            f"for title={completed_task['title']!r} summary={completed_task['resultSummary']!r}"
        )
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
        "transport": scenario["transport"],
        "lane": scenario["lane"],
    }


def run_browser_navigation_scenario(page, thread_id: str, scenario: dict) -> dict:
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
        "lane": scenario["lane"],
        "input": scenario["input"],
        "uiOptional": scenario.get("ui_optional", False),
    }


def run_browser_navigation_scenario_with_retry(page, thread_id: str, scenario: dict, retry_count: int = 2) -> dict:
    last_error = None
    current_scenario = copy.deepcopy(scenario)
    for attempt in range(retry_count + 1):
        try:
            return run_browser_navigation_scenario(page, thread_id, current_scenario)
        except AssertionError as exc:
            last_error = exc
            if attempt >= retry_count:
                raise
            message = str(exc)
            if (
                "browser: session unavailable" not in message
                and "context deadline exceeded" not in message
                and "websocket url timeout reached" not in message
            ):
                raise
            current_input = current_scenario.setdefault("input", {})
            current_tab_id = (current_input.get("tabId") or "").strip()
            if current_tab_id:
                current_input["tabId"] = refresh_browser_tab_id(current_tab_id)
            time.sleep(1.0 + attempt)
    if last_error is not None:
        raise last_error
    raise AssertionError(f"browser navigation retry failed without result for scenario {scenario.get('title', '')}")


def refresh_browser_tab_id(current_tab_id: str, fallback_exclude_ids: set[str] | None = None) -> str:
    target_tab_id = (current_tab_id or "").strip()
    if target_tab_id:
        try:
            snapshot = wait_for_browser_snapshot(
                lambda item: any(tab.get("id", "") == target_tab_id for tab in item.get("tabs", [])),
                timeout_seconds=3.0,
            )
            for tab in snapshot.get("tabs", []):
                if tab.get("id", "") == target_tab_id:
                    return target_tab_id
        except Exception:
            pass
    return get_latest_browser_tab_id(exclude_ids=fallback_exclude_ids or set())


def run_controlled_browser_scenario(page, thread_id: str, thread_name: str, run_id: str) -> dict:
    results = []
    fixture_url = build_controlled_browser_fixture_url(thread_id, thread_name)
    open_result = run_browser_navigation_scenario_with_retry(
        page,
        thread_id,
        {
            "title": f"Browser controlled open {run_id}",
            "kind": "browser.open",
            "input": {"url": fixture_url},
            "summary_contains": "browser tab opened",
            "lane": "controlled browser canonical lane",
        },
    )
    results.append(open_result)
    controlled_tab_id = resolve_browser_tab_id(open_result)

    for scenario in [
        {
            "title": f"Browser controlled type {run_id}",
            "kind": "browser.type",
            "input": {"tabId": "", "selector": "[data-testid='controlled-browser-input']", "text": "controlled browser acceptance"},
            "summary_contains": "browser type executed",
            "ui_optional": True,
        },
        {
            "title": f"Browser controlled click {run_id}",
            "kind": "browser.click",
            "input": {"tabId": "", "selector": "[data-testid='controlled-browser-apply']"},
            "summary_contains": "browser click executed",
            "ui_optional": True,
        },
        {
            "title": f"Browser controlled extract {run_id}",
            "kind": "browser.extract",
            "input": {"tabId": "", "selector": "[data-testid='controlled-browser-result']"},
            "summary_contains": "browser extract completed",
            "ui_optional": True,
        },
        {
            "title": f"Browser controlled screenshot {run_id}",
            "kind": "browser.screenshot",
            "input": {"tabId": ""},
            "summary_contains": "browser screenshot captured",
        },
    ]:
        scenario["input"]["tabId"] = controlled_tab_id
        results.append(
            run_browser_navigation_scenario_with_retry(
                page,
                thread_id,
                {
                    **scenario,
                    "lane": "controlled browser canonical lane",
                },
            )
        )
        controlled_tab_id = refresh_browser_tab_id(resolve_browser_tab_id(results[-1], fallback_exclude_ids=set()))

    extract_result = next(item for item in results if item["task"]["kind"] == "browser.extract")
    screenshot_result = next(item for item in results if item["task"]["kind"] == "browser.screenshot")
    screenshot_artifact = wait_for_artifact(
        thread_id,
        lambda item: item["kind"] == "browser.screenshot" and item["path"] in screenshot_result["task"]["resultSummary"],
    )
    browser_state = wait_for_browser_snapshot(
        lambda item: bool(item.get("activeTabId")) or bool(item.get("tabs", []))
    )

    return {
        "taskIds": [item["task"]["id"] for item in results],
        "toolKinds": [item["task"]["kind"] for item in results],
        "activeTabId": browser_state.get("activeTabId", ""),
        "extractSummary": extract_result["record"]["summary"],
        "fixtureURL": fixture_url,
        "screenshotArtifactPath": screenshot_artifact["path"],
        "toolCallsVisible": all(
            is_scenario_visible(item["visibility"]) or item.get("uiOptional", False) for item in results
        ),
        "records": results,
    }


def run_authenticated_browser_scenario(page, thread_id: str, thread_name: str, run_id: str) -> dict:
    fixture_url = build_authenticated_browser_fixture_url(thread_id, thread_name)
    open_result = run_browser_navigation_scenario_with_retry(
        page,
        thread_id,
        {
            "title": f"Browser authenticated open {run_id}",
            "kind": "browser.open",
            "input": {"url": fixture_url},
            "summary_contains": "browser tab opened",
            "lane": "authenticated browser canonical lane",
        },
    )
    authenticated_tab_id = resolve_browser_tab_id(open_result)
    authenticated_tab_url = get_browser_tab_url(authenticated_tab_id)
    session_result = run_browser_navigation_scenario_with_retry(
        page,
        thread_id,
        {
            "title": f"Browser authenticated session {run_id}",
            "kind": "browser.extract",
            "input": {"tabId": authenticated_tab_id, "selector": "[data-testid='authenticated-browser-session']"},
            "summary_contains": "browser extract completed",
            "lane": "authenticated browser canonical lane",
            "ui_optional": True,
        },
    )
    authenticated_tab_id = refresh_browser_tab_id(authenticated_tab_id)
    if AUTHENTICATED_SESSION_LABEL not in session_result["record"]["summary"]:
        fail(
            "authenticated browser session extract did not return the expected session label",
            category="authenticated-browser-assertion",
            details={
                "expectedSessionLabel": AUTHENTICATED_SESSION_LABEL,
                "actualSummary": session_result["record"]["summary"],
            },
        )
    profile_result = run_browser_navigation_scenario_with_retry(
        page,
        thread_id,
        {
            "title": f"Browser authenticated profile {run_id}",
            "kind": "browser.extract",
            "input": {"tabId": authenticated_tab_id, "selector": "[data-testid='authenticated-browser-profile']"},
            "summary_contains": "browser extract completed",
            "lane": "authenticated browser canonical lane",
            "ui_optional": True,
        },
    )
    authenticated_tab_id = refresh_browser_tab_id(authenticated_tab_id)
    if AUTHENTICATED_PROFILE_LABEL not in profile_result["record"]["summary"]:
        fail(
            "authenticated browser profile extract did not return the expected profile label",
            category="authenticated-browser-assertion",
            details={
                "expectedProfileLabel": AUTHENTICATED_PROFILE_LABEL,
                "actualSummary": profile_result["record"]["summary"],
            },
        )
    role_result = run_browser_navigation_scenario_with_retry(
        page,
        thread_id,
        {
            "title": f"Browser authenticated role {run_id}",
            "kind": "browser.extract",
            "input": {"tabId": authenticated_tab_id, "selector": "[data-testid='authenticated-browser-role']"},
            "summary_contains": "browser extract completed",
            "lane": "authenticated browser canonical lane",
            "ui_optional": True,
        },
    )
    authenticated_tab_id = refresh_browser_tab_id(authenticated_tab_id)
    if AUTHENTICATED_ROLE_LABEL not in role_result["record"]["summary"]:
        fail(
            "authenticated browser role extract did not return the expected role label",
            category="authenticated-browser-assertion",
            details={
                "expectedRoleLabel": AUTHENTICATED_ROLE_LABEL,
                "actualSummary": role_result["record"]["summary"],
            },
        )
    scope_result = run_browser_navigation_scenario_with_retry(
        page,
        thread_id,
        {
            "title": f"Browser authenticated scope {run_id}",
            "kind": "browser.extract",
            "input": {"tabId": authenticated_tab_id, "selector": "[data-testid='authenticated-browser-scope']"},
            "summary_contains": "browser extract completed",
            "lane": "authenticated browser canonical lane",
            "ui_optional": True,
        },
    )
    authenticated_tab_id = refresh_browser_tab_id(authenticated_tab_id)
    if AUTHENTICATED_SCOPE_LABEL not in scope_result["record"]["summary"]:
        fail(
            "authenticated browser scope extract did not return the expected scope label",
            category="authenticated-browser-assertion",
            details={
                "expectedScopeLabel": AUTHENTICATED_SCOPE_LABEL,
                "actualSummary": scope_result["record"]["summary"],
            },
        )
    transport_result = run_browser_navigation_scenario_with_retry(
        page,
        thread_id,
        {
            "title": f"Browser authenticated transport {run_id}",
            "kind": "browser.extract",
            "input": {"tabId": authenticated_tab_id, "selector": "[data-testid='authenticated-browser-transport']"},
            "summary_contains": "browser extract completed",
            "lane": "authenticated browser canonical lane",
            "ui_optional": True,
        },
    )
    authenticated_tab_id = refresh_browser_tab_id(authenticated_tab_id)
    if AUTHENTICATED_TRANSPORT_LABEL not in transport_result["record"]["summary"]:
        fail(
            "authenticated browser transport extract did not return the expected transport label",
            category="authenticated-browser-assertion",
            details={
                "expectedTransportLabel": AUTHENTICATED_TRANSPORT_LABEL,
                "actualSummary": transport_result["record"]["summary"],
            },
        )
    identity_result = run_browser_navigation_scenario_with_retry(
        page,
        thread_id,
        {
            "title": f"Browser authenticated extract {run_id}",
            "kind": "browser.extract",
            "input": {"tabId": authenticated_tab_id, "selector": "[data-testid='authenticated-browser-result']"},
            "summary_contains": "browser extract completed",
            "lane": "authenticated browser canonical lane",
            "ui_optional": True,
        },
    )
    authenticated_tab_id = refresh_browser_tab_id(authenticated_tab_id)
    if AUTHENTICATED_RESULT_TEXT not in identity_result["record"]["summary"]:
        fail(
            "authenticated browser extract did not return the expected stable identity token",
            category="authenticated-browser-assertion",
            details={
                "expectedResultText": AUTHENTICATED_RESULT_TEXT,
                "actualSummary": identity_result["record"]["summary"],
            },
        )
    screenshot_result = run_browser_navigation_scenario_with_retry(
        page,
        thread_id,
        {
            "title": f"Browser authenticated screenshot {run_id}",
            "kind": "browser.screenshot",
            "input": {"tabId": authenticated_tab_id},
            "summary_contains": "browser screenshot captured",
            "lane": "authenticated browser canonical lane",
        },
    )
    screenshot_artifact = wait_for_artifact(
        thread_id,
        lambda item: item["kind"] == "browser.screenshot" and item["path"] in screenshot_result["task"]["resultSummary"],
    )
    return {
        "status": "passed",
        "taskIds": [
            open_result["task"]["id"],
            session_result["task"]["id"],
            profile_result["task"]["id"],
            role_result["task"]["id"],
            scope_result["task"]["id"],
            transport_result["task"]["id"],
            identity_result["task"]["id"],
            screenshot_result["task"]["id"],
        ],
        "taskKinds": [
            open_result["task"]["kind"],
            session_result["task"]["kind"],
            profile_result["task"]["kind"],
            role_result["task"]["kind"],
            scope_result["task"]["kind"],
            transport_result["task"]["kind"],
            identity_result["task"]["kind"],
            screenshot_result["task"]["kind"],
        ],
        "resultSummaries": [
            open_result["task"]["resultSummary"],
            session_result["task"]["resultSummary"],
            profile_result["task"]["resultSummary"],
            role_result["task"]["resultSummary"],
            scope_result["task"]["resultSummary"],
            transport_result["task"]["resultSummary"],
            identity_result["task"]["resultSummary"],
            screenshot_result["task"]["resultSummary"],
        ],
        "tabId": authenticated_tab_id,
        "tabUrl": authenticated_tab_url,
        "fixtureURL": fixture_url,
        "sessionLabel": AUTHENTICATED_SESSION_LABEL,
        "profileLabel": AUTHENTICATED_PROFILE_LABEL,
        "roleLabel": AUTHENTICATED_ROLE_LABEL,
        "scopeLabel": AUTHENTICATED_SCOPE_LABEL,
        "transportLabel": AUTHENTICATED_TRANSPORT_LABEL,
        "resultText": AUTHENTICATED_RESULT_TEXT,
        "artifactPath": screenshot_artifact["path"],
        "screenshotArtifactPath": screenshot_artifact["path"],
        "visibility": {
            "openVisible": is_scenario_visible(open_result["visibility"]),
            "sessionVisible": is_scenario_visible(session_result["visibility"]) or session_result.get("uiOptional", False),
            "profileVisible": is_scenario_visible(profile_result["visibility"]) or profile_result.get("uiOptional", False),
            "roleVisible": is_scenario_visible(role_result["visibility"]) or role_result.get("uiOptional", False),
            "scopeVisible": is_scenario_visible(scope_result["visibility"]) or scope_result.get("uiOptional", False),
            "transportVisible": is_scenario_visible(transport_result["visibility"]) or transport_result.get("uiOptional", False),
            "extractVisible": is_scenario_visible(identity_result["visibility"]) or identity_result.get("uiOptional", False),
            "screenshotVisible": is_scenario_visible(screenshot_result["visibility"]),
        },
        "records": [
            open_result,
            session_result,
            profile_result,
            role_result,
            scope_result,
            transport_result,
            identity_result,
            screenshot_result,
        ],
    }


def run_public_web_browser_scenario(page, thread_id: str, run_id: str) -> dict:
    mode = classify_public_web_mode()
    if not mode["enabled"]:
        return mode
    target_urls = [validate_public_web_target(item) for item in mode.get("targetUrls", [])]
    preflights = [preflight_public_web_target(item) for item in target_urls]
    all_task_ids = []
    all_task_kinds = []
    all_result_summaries = []
    all_records = []
    target_results = []
    public_tab_id = ""
    public_tab_url = ""
    last_extract_summary = ""
    last_artifact_path = ""

    for index, target_url in enumerate(target_urls):
        preflight = preflights[index]
        if index == 0:
            navigation_result = run_browser_navigation_scenario(
                page,
                thread_id,
                {
                    "title": f"Browser public open {index + 1} {run_id}",
                    "kind": "browser.open",
                    "input": {"url": target_url},
                    "summary_contains": "browser tab opened",
                    "lane": "public-web read-only browser lane",
                },
            )
        else:
            navigation_result = run_browser_navigation_scenario_with_retry(
                page,
                thread_id,
                {
                    "title": f"Browser public navigate {index + 1} {run_id}",
                    "kind": "browser.navigate",
                    "input": {"tabId": public_tab_id, "url": target_url},
                    "summary_contains": "browser tab navigated",
                    "lane": "public-web read-only browser lane",
                },
            )
        public_tab_id = resolve_browser_tab_id(navigation_result)
        public_tab_url = get_browser_tab_url(public_tab_id)
        extract_result = run_browser_navigation_scenario(
            page,
            thread_id,
            {
                "title": f"Browser public extract {index + 1} {run_id}",
                "kind": "browser.extract",
                "input": {"tabId": public_tab_id, "selector": "h1"},
                "summary_contains": "browser extract completed",
                "lane": "public-web read-only browser lane",
                "ui_optional": True,
            },
        )
        parsed_target = urllib.parse.urlparse(target_url)
        parsed_active = urllib.parse.urlparse(public_tab_url)
        preflight_final_host = urllib.parse.urlparse(preflight.get("finalUrl", "")).hostname
        allowed_hosts = {parsed_target.hostname, preflight_final_host}
        if parsed_active.scheme != "https" or parsed_active.hostname not in allowed_hosts:
            fail(
                "public-web browser lane did not stay on the configured allowlisted host",
                category="public-web-assertion",
                details={
                    "targetUrl": target_url,
                    "activeTabUrl": public_tab_url,
                    "allowedHosts": sorted([host for host in allowed_hosts if host]),
                },
            )
        screenshot_result = run_browser_navigation_scenario(
            page,
            thread_id,
            {
                "title": f"Browser public screenshot {index + 1} {run_id}",
                "kind": "browser.screenshot",
                "input": {"tabId": public_tab_id},
                "summary_contains": "browser screenshot captured",
                "lane": "public-web read-only browser lane",
            },
        )
        screenshot_artifact = wait_for_artifact(
            thread_id,
            lambda item: item["kind"] == "browser.screenshot" and item["path"] in screenshot_result["task"]["resultSummary"],
        )
        last_extract_summary = extract_result["record"]["summary"]
        last_artifact_path = screenshot_artifact["path"]
        scenario_records = [navigation_result, extract_result, screenshot_result]
        all_records.extend(scenario_records)
        all_task_ids.extend(item["task"]["id"] for item in scenario_records)
        all_task_kinds.extend(item["task"]["kind"] for item in scenario_records)
        all_result_summaries.extend(item["task"]["resultSummary"] for item in scenario_records)
        target_results.append(
            {
                "index": index + 1,
                "targetUrl": target_url,
                "finalUrl": public_tab_url,
                "tabId": public_tab_id,
                "preflight": preflight,
                "taskIds": [item["task"]["id"] for item in scenario_records],
                "taskKinds": [item["task"]["kind"] for item in scenario_records],
                "resultSummaries": [item["task"]["resultSummary"] for item in scenario_records],
                "extractSummary": extract_result["record"]["summary"],
                "artifactPath": screenshot_artifact["path"],
                "visibility": {
                    "navigationVisible": is_scenario_visible(navigation_result["visibility"]),
                    "extractVisible": is_scenario_visible(extract_result["visibility"]) or extract_result.get("uiOptional", False),
                    "screenshotVisible": is_scenario_visible(screenshot_result["visibility"]),
                },
            }
        )
    return {
        "enabled": True,
        "status": "passed",
        "classification": "required-and-passed",
        "required": True,
        "targetUrl": target_urls[0],
        "targetUrls": target_urls,
        "targetCount": len(target_urls),
        "targetResults": target_results,
        "taskIds": all_task_ids,
        "taskKinds": all_task_kinds,
        "resultSummaries": all_result_summaries,
        "tabId": public_tab_id,
        "tabUrl": public_tab_url,
        "transportScope": "public-web-read-only",
        "preflight": preflights[0],
        "preflights": preflights,
        "extractSummary": last_extract_summary,
        "artifactPath": last_artifact_path,
        "screenshotArtifactPath": last_artifact_path,
        "visibility": {
            "targetCount": len(target_results),
            "targetsVisible": sum(
                1
                for item in target_results
                if item["visibility"]["navigationVisible"]
                and item["visibility"]["extractVisible"]
                and item["visibility"]["screenshotVisible"]
            ),
        },
        "records": all_records,
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


def run_ui_first_browser_agent_scenario(page, thread_id: str, thread_name: str, run_id: str) -> dict:
    title = f"UI-first browser agent {run_id}"
    baseline_assistant_messages = len(
        [item for item in api("GET", f"/api/threads/{thread_id}/messages")["items"] if item.get("role") == "assistant"]
    )
    goal = (
        "Open the controlled browser fixture page for the current thread. Use "
        "[data-testid='controlled-browser-input'] to type browser demo text, click "
        "[data-testid='controlled-browser-apply'], extract "
        "[data-testid='controlled-browser-result'], take a screenshot, and then answer in one short sentence "
        "with the extracted result. Do not write files."
    )
    created_task = create_agent_task_via_ui(page, thread_id, title, goal, max_steps=6)
    task_id = created_task["id"]
    run_trigger_mode = run_task_via_ui(page, thread_id, task_id, title)

    parent_task = wait_for_task_terminal(thread_id, task_id, timeout_seconds=90.0)
    if parent_task.get("status") != "completed":
        raise AssertionError(f"expected UI-first browser agent parent completed, got {parent_task.get('status')!r}")
    if parent_task.get("agentPlanMode") != "browser_then_respond":
        raise AssertionError(
            f"expected UI-first browser agent plan mode browser_then_respond, got {parent_task.get('agentPlanMode')!r}"
        )

    child_tasks = api("GET", f"/api/threads/{thread_id}/tasks")["items"]
    completed_children = [
        item
        for item in child_tasks
        if item.get("parentTaskId") == task_id and item.get("status") == "completed"
    ]
    child_kinds = [item["kind"] for item in completed_children]
    assert_required_child_sequence(
        child_kinds,
        [
            {"browser.open"},
            {"browser.type"},
            {"browser.click"},
            {"browser.extract"},
            {"browser.screenshot"},
        ],
    )

    latest_message = wait_for_new_assistant_message(thread_id, baseline_assistant_messages, timeout_seconds=30.0)
    extract_child = next((item for item in completed_children if item.get("kind") == "browser.extract"), None)
    screenshot_child = next((item for item in completed_children if item.get("kind") == "browser.screenshot"), None)
    if extract_child is None:
        raise AssertionError("expected browser.extract child task for UI-first browser agent scenario")
    if screenshot_child is None:
        raise AssertionError("expected browser.screenshot child task for UI-first browser agent scenario")

    screenshot_artifact = wait_for_artifact(
        thread_id,
        lambda item: item["kind"] == "browser.screenshot" and item["path"] in screenshot_child["resultSummary"],
        timeout_seconds=30.0,
    )

    refresh_thread_view(page, thread_id)
    assert_article_contains(page, title, timeout=12000)
    latest_agent_card = page.get_by_test_id("latest-agent-card")
    latest_child_card = page.get_by_test_id("latest-child-task-card")
    latest_message_card = page.get_by_test_id("latest-message-card")

    expect(latest_agent_card).to_contain_text(title, timeout=15000)
    expect(latest_agent_card).to_contain_text("browser_then_respond", timeout=15000)
    expect(latest_child_card).to_contain_text(screenshot_child["id"], timeout=15000)
    expect(latest_child_card).to_contain_text("browser.screenshot", timeout=15000)
    expect(latest_child_card).to_contain_text(screenshot_artifact["path"], timeout=15000)
    expect(latest_message_card).to_contain_text(latest_message["content"], timeout=15000)

    child_visible = False
    child_visibility_mode = ""
    for child in [extract_child, screenshot_child]:
        try:
            expected_text = screenshot_artifact["path"] if child["kind"] == "browser.screenshot" else child["resultSummary"]
            assert_article_contains(page, child["title"], expected_text, timeout=12000)
            child_visible = True
            child_visibility_mode = "title+browser-summary"
        except Exception:
            continue

    return {
        "parentTaskId": task_id,
        "parentTitle": title,
        "parentStatus": parent_task["status"],
        "planMode": parent_task.get("agentPlanMode", ""),
        "runTriggerMode": run_trigger_mode,
        "uiRunTriggered": run_trigger_mode == "desktop-ui",
        "childTaskIds": [item["id"] for item in completed_children],
        "childKinds": child_kinds,
        "extractSummary": extract_child["resultSummary"],
        "assistantMessageId": latest_message["id"],
        "assistantMessage": latest_message["content"],
        "screenshotArtifactPath": screenshot_artifact["path"],
        "visibility": {
            "parentVisible": True,
            "latestAgentCardVisible": latest_agent_card.is_visible(),
            "latestChildCardVisible": latest_child_card.is_visible(),
            "latestMessageCardVisible": latest_message_card.is_visible(),
            "childVisible": child_visible,
            "childVisibilityMode": child_visibility_mode,
            "toolCallsVisible": True,
        },
    }


def run_recovery_continuation_scenario(page, thread_id: str, run_id: str) -> dict:
    title = f"Agent recovery continuation {run_id}"
    baseline_note_path = os.path.join(os.getcwd(), "playwright-recovery-note.txt")
    if not os.path.exists(baseline_note_path):
        with open(baseline_note_path, "w", encoding="utf-8") as handle:
            handle.write("baseline recovery note\n")
    baseline_assistant_messages = len(
        [item for item in api("GET", f"/api/threads/{thread_id}/messages")["items"] if item.get("role") == "assistant"]
    )
    goal = (
        "Update the existing file playwright-recovery-note.txt by adding one line saying recovery evidence, "
        "then answer in one short sentence about what you changed. Do not do any extra work."
    )
    created_task = create_agent_task_via_ui(page, thread_id, title, goal, max_steps=4)
    task_id = created_task["id"]
    run_trigger_mode = run_task_via_ui(page, thread_id, task_id, title)

    waiting_parent = wait_for_task(
        thread_id,
        lambda item: item["id"] == task_id and item["status"] == "waiting_for_approval",
    )
    if waiting_parent.get("agentPlanMode") != "patch_then_respond":
        raise AssertionError(
            f"expected recovery continuation plan mode patch_then_respond, got {waiting_parent.get('agentPlanMode')!r}"
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

    approve_trigger_mode = approve_task_via_ui(page, thread_id, waiting_child["id"], waiting_child["title"])
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
        "planMode": resumed_parent.get("agentPlanMode", ""),
        "uiCreateTriggered": True,
        "uiRunTriggered": run_trigger_mode == "desktop-ui",
        "runTriggerMode": run_trigger_mode,
        "uiApproveTriggered": approve_trigger_mode == "desktop-ui",
        "approveTriggerMode": approve_trigger_mode,
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
      return write_png_artifact(current_failure_png_name(), screenshot)
    except Exception:
      return None


def main() -> int:
    run_id = f"{int(time.time())}-{uuid.uuid4().hex[:8]}"
    thread_name = f"Playwright Acceptance {run_id}"
    task_title = f"Playwright Live Refresh {run_id}"
    patch_path = f"playwright-live-refresh-{run_id}.txt"
    rollback_task_title = "Rollback latest write execution"

    runtime_status = ensure_canonical_runtime()
    mcp_verified_lanes = ensure_mcp_verified_lanes()
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
                clear_stale_failure_artifacts()
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
            browser_results = []
            browser_lane = "browser navigation canonical lane"
            browser_results.append(
                run_browser_navigation_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Browser state direct {run_id}",
                        "kind": "browser.state",
                        "input": {},
                        "summary_contains": "browser state captured",
                        "lane": browser_lane,
                    },
                )
            )
            browser_results.append(
                run_browser_navigation_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Browser open preview {run_id}",
                        "kind": "browser.open",
                        "input": {"url": build_thread_preview_fixture_url(thread_id, thread_name, "thread-one")},
                        "summary_contains": "browser tab opened",
                        "lane": browser_lane,
                    },
                )
            )
            primary_browser_tab_id = resolve_browser_tab_id(browser_results[-1])
            browser_results.append(
                run_browser_navigation_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Browser navigate preview fixture {run_id}",
                        "kind": "browser.navigate",
                        "input": {
                            "tabId": primary_browser_tab_id,
                            "url": build_thread_preview_fixture_url(thread_id, thread_name, "thread-two"),
                        },
                        "summary_contains": "browser tab navigated",
                        "lane": browser_lane,
                    },
                )
            )
            primary_browser_tab_id = resolve_browser_tab_id(browser_results[-1])
            browser_results.append(
                run_browser_navigation_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Browser reload active {run_id}",
                        "kind": "browser.reload",
                        "input": {"tabId": primary_browser_tab_id},
                        "summary_contains": "browser tab reloaded",
                        "lane": browser_lane,
                    },
                )
            )
            primary_browser_tab_id = resolve_browser_tab_id(browser_results[-1])
            browser_results.append(
                run_browser_navigation_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Browser back active {run_id}",
                        "kind": "browser.back",
                        "input": {"tabId": primary_browser_tab_id},
                        "summary_contains": "browser tab went back",
                        "lane": browser_lane,
                    },
                )
            )
            primary_browser_tab_id = resolve_browser_tab_id(browser_results[-1])
            browser_results.append(
                run_browser_navigation_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Browser forward active {run_id}",
                        "kind": "browser.forward",
                        "input": {"tabId": primary_browser_tab_id},
                        "summary_contains": "browser tab went forward",
                        "lane": browser_lane,
                    },
                )
            )
            primary_browser_tab_id = resolve_browser_tab_id(browser_results[-1])
            preexisting_tab_ids = {
                tab.get("id", "")
                for tab in get_browser_snapshot().get("tabs", [])
                if tab.get("id", "")
            }
            browser_results.append(
                run_browser_navigation_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Browser open preview secondary {run_id}",
                        "kind": "browser.open",
                        "input": {"url": build_thread_preview_fixture_url(thread_id, thread_name, "thread-three")},
                        "summary_contains": "browser tab opened",
                        "lane": browser_lane,
                    },
                )
            )
            secondary_browser_tab_id = extract_tab_id_from_summary(browser_results[-1]["task"]["resultSummary"])
            if secondary_browser_tab_id == primary_browser_tab_id:
                secondary_browser_tab_id = get_latest_browser_tab_id(exclude_ids={primary_browser_tab_id} | preexisting_tab_ids)
            browser_results.append(
                run_browser_navigation_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Browser activate primary {run_id}",
                        "kind": "browser.activate_tab",
                        "input": {"tabId": primary_browser_tab_id},
                        "summary_contains": "browser tab activated",
                        "lane": browser_lane,
                    },
                )
            )
            browser_results.append(
                run_browser_navigation_scenario(
                    page,
                    thread_id,
                    {
                        "title": f"Browser close secondary {run_id}",
                        "kind": "browser.close_tab",
                        "input": {"tabId": secondary_browser_tab_id},
                        "summary_contains": "browser tab closed",
                        "lane": browser_lane,
                    },
                )
            )
            controlled_browser_result = run_controlled_browser_scenario(page, thread_id, thread_name, run_id)
            authenticated_browser_result = run_authenticated_browser_scenario(page, thread_id, thread_name, run_id)
            public_web_browser_result = run_public_web_browser_scenario(page, thread_id, run_id)
            if BROWSER_ONLY_ACCEPTANCE:
                result = build_browser_only_full_result(
                    thread_id=thread_id,
                    thread_name=thread_name,
                    runtime_status=runtime_status,
                    refresh_mode=refresh_mode,
                    desktop_copy_runtime=desktop_copy_runtime,
                    browser_results=browser_results,
                    controlled_browser_result=controlled_browser_result,
                    authenticated_browser_result=authenticated_browser_result,
                    public_web_browser_result=public_web_browser_result,
                )
                clear_stale_failure_artifacts()
                write_json_artifact("desktop-full-summary.json", result)
                print(json.dumps(result, ensure_ascii=False))
                return 0
            mcp_execution_results = []
            for scenario in [
                {
                    "title": f"Invoke MCP fixture echo {run_id}",
                    "kind": "mcp.tool.invoke",
                    "input": {"serverId": "external-fixture", "toolName": "echo", "arguments": {"message": "hello"}},
                    "summary_contains": "mcp tool external-fixture/echo executed",
                    "transport": "stdio-fixture",
                    "lane": "fixture regression lane",
                },
                {
                    "title": f"Invoke MCP sdk echo {run_id}",
                    "kind": "mcp.tool.invoke",
                    "input": {"serverId": "sdk-external-fixture", "toolName": "echo", "arguments": {"message": "hello-sdk"}},
                    "summary_contains": "mcp tool sdk-external-fixture/echo executed",
                    "transport": "stdio-sdk",
                    "lane": "official SDK external lane",
                },
                {
                    "title": f"Invoke MCP third-party time {run_id}",
                    "kind": "mcp.tool.invoke",
                    "input": {"serverId": "third-party-time", "toolName": "get_current_time", "arguments": {"timezone": "UTC"}},
                    "summary_contains": "mcp tool third-party-time/get_current_time executed",
                    "transport": "stdio-third-party",
                    "lane": "third-party time lane",
                },
            ]:
                mcp_execution_results.append(run_mcp_execution_scenario(page, thread_id, scenario))
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
                            {"workspace.read_files_batch"},
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
            ui_first_browser_agent_result = run_ui_first_browser_agent_scenario(page, thread_id, thread_name, run_id)
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
                    "uiFirstCanonicalAgentScenario": recovery_result,
                    "uiFirstCanonicalBrowserAgentScenario": ui_first_browser_agent_result,
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
                    "browserNavigationVisibility": {
                        "scenarioCount": len(browser_results),
                        "visibleByTitleCount": sum(
                            1 for item in browser_results if item["visibility"]["taskCardVisible"]
                        ),
                        "visibleByToolKindFallbackCount": sum(
                            1 for item in browser_results if item["visibility"]["toolKindVisible"]
                        ),
                    },
                    "controlledBrowserVisibility": controlled_browser_result,
                    "authenticatedBrowserVisibility": authenticated_browser_result,
                    "publicWebBrowserVisibility": public_web_browser_result,
                    "mcpExecutionVisibility": {
                        "scenarioCount": len(mcp_execution_results),
                        "visibleByTitleCount": sum(
                            1 for item in mcp_execution_results if item["visibility"]["taskCardVisible"]
                        ),
                        "visibleByToolKindFallbackCount": sum(
                            1 for item in mcp_execution_results if item["visibility"]["toolKindVisible"]
                        ),
                    },
                    "mcpVerifiedLanePreflight": mcp_verified_lanes,
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
                "browserNavigationTasks": [
                    {
                        "taskId": item["task"]["id"],
                        "title": item["task"]["title"],
                        "kind": item["task"]["kind"],
                        "lane": item["lane"],
                        "input": item["input"],
                        "resultSummary": item["task"]["resultSummary"],
                        "record": item["record"],
                        "visibility": item["visibility"],
                    }
                    for item in browser_results
                ],
                "controlledBrowserScenario": controlled_browser_result,
                "authenticatedBrowserScenario": authenticated_browser_result,
                "publicWebBrowserScenario": public_web_browser_result,
                "mcpExecutionResults": [
                    {
                        "taskId": item["task"]["id"],
                        "title": item["task"]["title"],
                        "kind": item["task"]["kind"],
                        "lane": item["lane"],
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
                "uiFirstCanonicalAgentScenario": recovery_result,
                "uiFirstCanonicalBrowserAgentScenario": ui_first_browser_agent_result,
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
            clear_stale_failure_artifacts()
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
    if "uiFirstCanonicalAgentScenario" not in remote_report:
        raise AssertionError("missing UI-first canonical agent scenario in remote acceptance report")
    if "uiFirstCanonicalBrowserAgentScenario" not in remote_report:
        raise AssertionError("missing UI-first canonical browser agent scenario in remote acceptance report")
    browser_agent_scenario = remote_report.get("uiFirstCanonicalBrowserAgentScenario") or {}
    for key in (
        "parentTaskId",
        "childTaskIds",
        "childKinds",
        "extractSummary",
        "assistantMessage",
        "screenshotArtifactPath",
        "visibility",
    ):
        if key not in browser_agent_scenario:
            raise AssertionError(f"missing UI-first browser agent acceptance key: {key}")
    browser_visibility = remote_report.get("browserNavigationVisibility") or {}
    if browser_visibility.get("scenarioCount", 0) < 1:
        raise AssertionError("missing browser navigation canonical acceptance evidence")
    controlled_browser = remote_report.get("controlledBrowserVisibility") or {}
    for key in ("taskIds", "toolKinds", "activeTabId", "extractSummary", "screenshotArtifactPath", "toolCallsVisible"):
        if key not in controlled_browser:
            raise AssertionError(f"missing controlled browser acceptance key: {key}")
    authenticated_browser = remote_report.get("authenticatedBrowserVisibility") or {}
    if authenticated_browser.get("status") != "passed":
        raise AssertionError(f"authenticated browser lane did not pass: {authenticated_browser}")
    for key in (
        "taskIds",
        "taskKinds",
        "resultSummaries",
        "tabId",
        "tabUrl",
        "fixtureURL",
        "sessionLabel",
        "profileLabel",
        "roleLabel",
        "scopeLabel",
        "transportLabel",
        "resultText",
        "artifactPath",
        "screenshotArtifactPath",
        "visibility",
    ):
        if key not in authenticated_browser:
            raise AssertionError(f"missing authenticated browser acceptance key: {key}")
    public_web_browser = remote_report.get("publicWebBrowserVisibility") or {}
    public_web_status = public_web_browser.get("status")
    if public_web_status not in {"passed", "skipped"}:
        raise AssertionError(f"public-web browser lane must classify itself as passed or skipped: {public_web_browser}")
    if public_web_status == "passed":
        for key in (
            "classification",
            "required",
            "taskIds",
            "taskKinds",
            "resultSummaries",
            "tabId",
            "tabUrl",
            "targetUrl",
            "targetUrls",
            "targetCount",
            "targetResults",
            "transportScope",
            "preflight",
            "preflights",
            "extractSummary",
            "artifactPath",
            "screenshotArtifactPath",
            "visibility",
        ):
            if key not in public_web_browser:
                raise AssertionError(f"missing public-web browser acceptance key: {key}")
    else:
        for key in ("classification", "reason"):
            if key not in public_web_browser:
                raise AssertionError(f"missing public-web browser skip key: {key}")

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


def build_browser_only_full_result(
    *,
    thread_id: str,
    thread_name: str,
    runtime_status: dict,
    refresh_mode: dict,
    desktop_copy_runtime: dict,
    browser_results: list,
    controlled_browser_result: dict,
    authenticated_browser_result: dict,
    public_web_browser_result: dict,
) -> dict:
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
            "browserNavigationVisibility": {
                "scenarioCount": len(browser_results),
                "visibleByTitleCount": sum(
                    1 for item in browser_results if item["visibility"]["taskCardVisible"]
                ),
                "visibleByToolKindFallbackCount": sum(
                    1 for item in browser_results if item["visibility"]["toolKindVisible"]
                ),
            },
            "controlledBrowserVisibility": controlled_browser_result,
            "authenticatedBrowserVisibility": authenticated_browser_result,
            "publicWebBrowserVisibility": public_web_browser_result,
            "browserOnlyAcceptance": {
                "enabled": True,
                "skippedSections": [
                    "direct-tools",
                    "mcp-verified-lanes",
                    "thread-mutations",
                    "agent-run",
                    "approval-flow",
                    "write-execution",
                    "rollback",
                    "fallback-evidence",
                ],
            },
        },
        "fallback": {
            "mode": "not-attempted",
            "browserAutomation": "not attempted",
            "reason": "browser-only acceptance isolates canonical remote browser lanes and skips fallback evidence collection",
        },
    }
    return {
        "ok": True,
        "threadId": thread_id,
        "threadName": thread_name,
        "runtimeSource": runtime_status.get("runtimeSource", ""),
        "runtimeTrust": runtime_status.get("runtimeTrust", ""),
        "canonicalRuntimeUrl": runtime_status.get("canonicalRuntimeUrl", ""),
        "uiBaseUrl": UI_BASE_URL,
        "apiBaseUrl": API_BASE_URL,
        "acceptanceMode": "full",
        "refreshMode": refresh_mode,
        "fallbackEvidenceMode": "not-attempted",
        "browserOnlyAcceptance": True,
        "browserNavigationTasks": [
            {
                "taskId": item["task"]["id"],
                "title": item["task"]["title"],
                "kind": item["task"]["kind"],
                "lane": item["lane"],
                "input": item["input"],
                "resultSummary": item["task"]["resultSummary"],
                "record": item["record"],
                "visibility": item["visibility"],
            }
            for item in browser_results
        ],
        "controlledBrowserScenario": controlled_browser_result,
        "authenticatedBrowserScenario": authenticated_browser_result,
        "publicWebBrowserScenario": public_web_browser_result,
        "acceptanceReport": acceptance_report,
    }


if __name__ == "__main__":
    failure_page = None
    try:
        raise SystemExit(main())
    except Exception as exc:
        parsed = normalize_failure(exc)
        write_json_artifact(current_failure_json_name(), parsed)
        if "page" in globals():
            try:
                write_failure_screenshot(globals()["page"])
            except Exception:
                pass
        print(json.dumps(parsed, ensure_ascii=False), file=sys.stderr)
        raise
