#!/usr/bin/env python3
"""Validate local runtime prerequisites for gen-code."""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parent.parent
GO_VERSION_PATTERN = re.compile(r"\bgo(\d+)\.(\d+)(?:\.(\d+))?\b")
PLAIN_GO_DIRECTIVE_PATTERN = re.compile(r"^go\s+(\d+)\.(\d+)(?:\.(\d+))?$")
NODE_VERSION_PATTERN = re.compile(r"\bv?(\d+)\.(\d+)\.(\d+)\b")


class RuntimeToolError(Exception):
    """Raised when the runtime check tool cannot complete."""


@dataclass(frozen=True)
class CheckItem:
    name: str
    status: str
    detail: str


@dataclass
class RuntimeCheckResult:
    status: str
    workspace_path: Path
    checks: list[CheckItem]
    errors: list[str]
    warnings: list[str]
    suggestions: list[str]


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate gen-code runtime prerequisites.")
    parser.add_argument("--workspace", help="Workspace root to inspect.")
    parser.add_argument(
        "--require-env",
        action="store_true",
        help="Treat a missing .env as a failure instead of a warning.",
    )
    parser.add_argument(
        "--strict",
        action="store_true",
        help="Treat non-blocking warnings as failures where applicable.",
    )
    return parser.parse_args(argv)


def resolve_workspace(path_value: str | None) -> Path:
    if path_value:
        workspace = Path(path_value).resolve()
    else:
        workspace = ROOT_DIR
    if not workspace.exists():
        raise RuntimeToolError(f"workspace path does not exist: {workspace}")
    if not workspace.is_dir():
        raise RuntimeToolError(f"workspace path is not a directory: {workspace}")
    return workspace


def run_command(args: list[str], cwd: Path) -> tuple[int, str]:
    try:
        completed = subprocess.run(
            args,
            cwd=cwd,
            capture_output=True,
            text=True,
            check=False,
        )
    except OSError as exc:
        raise RuntimeToolError(f"failed to execute {' '.join(args)!r}: {exc}") from exc
    output = (completed.stdout or completed.stderr).strip()
    return completed.returncode, output


def parse_go_requirement(go_mod_path: Path) -> tuple[int, int, int]:
    try:
        content = go_mod_path.read_text(encoding="utf-8")
    except OSError as exc:
        raise RuntimeToolError(f"failed to read {go_mod_path}: {exc}") from exc
    for line in content.splitlines():
        stripped = line.strip()
        if stripped.startswith("toolchain "):
            continue
        if not stripped.startswith("go "):
            continue
        match = PLAIN_GO_DIRECTIVE_PATTERN.match(stripped)
        if not match:
            raise RuntimeToolError(f"could not parse Go version requirement from {go_mod_path}")
        return normalize_version(match)
    raise RuntimeToolError(f"missing go version declaration in {go_mod_path}")


def normalize_version(match: re.Match[str]) -> tuple[int, int, int]:
    major = int(match.group(1))
    minor = int(match.group(2))
    patch = int(match.group(3) or "0")
    return (major, minor, patch)


def parse_go_version(output: str) -> tuple[int, int, int] | None:
    match = GO_VERSION_PATTERN.search(output)
    if not match:
        return None
    return normalize_version(match)


def parse_node_version(output: str) -> tuple[int, int, int] | None:
    match = NODE_VERSION_PATTERN.search(output)
    if not match:
        return None
    return (int(match.group(1)), int(match.group(2)), int(match.group(3)))


def format_version(version: tuple[int, int, int]) -> str:
    return f"{version[0]}.{version[1]}.{version[2]}"


def add_file_check(
    checks: list[CheckItem],
    errors: list[str],
    warnings: list[str],
    path: Path,
    label: str,
    *,
    required: bool,
    strict: bool,
) -> None:
    if path.is_file():
        checks.append(CheckItem(label, "ok", "present"))
        return
    checks.append(CheckItem(label, "fail" if required or strict else "warn", "missing"))
    message = f"{label} is missing: {path}"
    if required or strict:
        errors.append(message)
    else:
        warnings.append(message)


def add_directory_check(
    checks: list[CheckItem],
    errors: list[str],
    path: Path,
    label: str,
) -> None:
    if path.is_dir():
        checks.append(CheckItem(label, "ok", "present"))
        return
    checks.append(CheckItem(label, "fail", "missing"))
    errors.append(f"{label} directory is missing: {path}")


def evaluate_runtime(workspace: Path, require_env: bool, strict: bool) -> RuntimeCheckResult:
    checks: list[CheckItem] = []
    errors: list[str] = []
    warnings: list[str] = []
    suggestions: list[str] = []

    go_mod_path = workspace / "go.mod"
    add_file_check(checks, errors, warnings, go_mod_path, "go.mod", required=True, strict=strict)
    add_file_check(
        checks,
        errors,
        warnings,
        workspace / ".env.example",
        ".env.example",
        required=True,
        strict=strict,
    )
    add_file_check(
        checks,
        errors,
        warnings,
        workspace / "tools" / "check_config.py",
        "tools/check_config.py",
        required=True,
        strict=strict,
    )
    add_file_check(
        checks,
        errors,
        warnings,
        workspace / ".env",
        ".env",
        required=require_env,
        strict=strict and require_env,
    )
    add_directory_check(checks, errors, workspace / "internal", "internal")
    add_directory_check(checks, errors, workspace / "internal" / "core", "internal/core")

    python_version = format_version((sys.version_info.major, sys.version_info.minor, sys.version_info.micro))
    checks.append(CheckItem("python", "ok", python_version))

    required_go_version: tuple[int, int, int] | None = None
    if go_mod_path.is_file():
        required_go_version = parse_go_requirement(go_mod_path)

    try:
        go_code, go_output = run_command(["go", "version"], workspace)
    except RuntimeToolError:
        checks.append(CheckItem("go", "fail", "not found"))
        errors.append("go executable is not available on PATH")
        suggestions.append("Install Go or make it available on PATH.")
    else:
        go_version = parse_go_version(go_output)
        if go_code != 0 or not go_version:
            checks.append(CheckItem("go", "fail", go_output or "version probe failed"))
            errors.append("failed to determine Go version")
            suggestions.append("Verify that the go executable works correctly.")
        elif required_go_version and go_version < required_go_version:
            checks.append(
                CheckItem(
                    "go",
                    "fail",
                    f"found {format_version(go_version)}, require >= {format_version(required_go_version)}",
                )
            )
            errors.append("Go version is below the version required by go.mod")
            suggestions.append(
                f"Install Go {format_version(required_go_version)} or enable GOTOOLCHAIN=auto."
            )
        else:
            detail = format_version(go_version)
            if required_go_version:
                detail = f"{detail} (require >= {format_version(required_go_version)})"
            checks.append(CheckItem("go", "ok", detail))

    try:
        node_code, node_output = run_command(["node", "--version"], workspace)
    except RuntimeToolError:
        checks.append(CheckItem("node", "warn" if not strict else "fail", "not found"))
        message = "node executable is not available on PATH"
        if strict:
            errors.append(message)
        else:
            warnings.append(message)
        suggestions.append("Install Node.js if you need browser or frontend-related flows.")
    else:
        node_version = parse_node_version(node_output)
        if node_code != 0 or not node_version:
            checks.append(CheckItem("node", "warn" if not strict else "fail", node_output or "version probe failed"))
            message = "failed to determine Node.js version"
            if strict:
                errors.append(message)
            else:
                warnings.append(message)
        else:
            checks.append(CheckItem("node", "ok", format_version(node_version)))

    if (workspace / ".env").is_file() is False:
        suggestions.append("Create .env from .env.example before the first runtime start.")

    status = "FAIL" if errors else ("WARN" if warnings else "PASS")
    return RuntimeCheckResult(
        status=status,
        workspace_path=workspace,
        checks=checks,
        errors=errors,
        warnings=warnings,
        suggestions=dedupe_preserve_order(suggestions),
    )


def dedupe_preserve_order(items: list[str]) -> list[str]:
    seen: set[str] = set()
    result: list[str] = []
    for item in items:
        if item in seen:
            continue
        seen.add(item)
        result.append(item)
    return result


def format_result(result: RuntimeCheckResult) -> str:
    lines = [
        f"Status: {result.status}",
        f"Workspace: {result.workspace_path}",
        "",
        "Checks:",
    ]
    for item in result.checks:
        lines.append(f"  - {item.name}: {item.status} ({item.detail})")

    lines.extend(["", "Errors:"])
    if result.errors:
        lines.extend(f"  - {item}" for item in result.errors)
    else:
        lines.append("  - none")

    lines.extend(["", "Warnings:"])
    if result.warnings:
        lines.extend(f"  - {item}" for item in result.warnings)
    else:
        lines.append("  - none")

    lines.extend(["", "Suggestions:"])
    if result.suggestions:
        lines.extend(f"  - {item}" for item in result.suggestions)
    else:
        lines.append("  - none")

    return "\n".join(lines)


def run(argv: list[str]) -> int:
    try:
        args = parse_args(argv)
        workspace = resolve_workspace(args.workspace)
        result = evaluate_runtime(workspace, args.require_env, args.strict)
        print(format_result(result))
        return 1 if result.status == "FAIL" else 0
    except RuntimeToolError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 2


def main() -> int:
    return run(sys.argv[1:])


if __name__ == "__main__":
    raise SystemExit(main())
