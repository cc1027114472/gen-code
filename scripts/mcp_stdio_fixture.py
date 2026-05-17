import json
import sys


def main() -> int:
    try:
        payload = json.load(sys.stdin)
    except Exception as exc:  # noqa: BLE001
        json.dump({"ok": False, "error": f"invalid request: {exc}"}, sys.stdout)
        return 1

    tool_name = str(payload.get("toolName", "")).strip()
    arguments = payload.get("arguments") or {}

    if tool_name == "echo":
        message = str(arguments.get("message", ""))
        json.dump(
            {
                "ok": True,
                "summary": "mcp tool external-fixture/echo executed",
                "result": {"echo": message},
            },
            sys.stdout,
        )
        return 0

    if tool_name == "sum":
        values = arguments.get("values") or []
        total = sum(int(value) for value in values)
        json.dump(
            {
                "ok": True,
                "summary": "mcp tool external-fixture/sum executed",
                "result": {"total": total},
            },
            sys.stdout,
        )
        return 0

    if tool_name == "fail":
        json.dump({"ok": False, "error": "fixture forced failure"}, sys.stdout)
        return 0

    json.dump({"ok": False, "error": f"unknown tool: {tool_name}"}, sys.stdout)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
