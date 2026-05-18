#!/usr/bin/env python3
"""Validate workspace structure and Python tool inventory for gen-code."""

from __future__ import annotations

import argparse
import sys
from dataclasses import dataclass
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parent.parent
EXPECTED_TOOLS = ("check_config.py", "check_runtime.py", "check_workspace.py")
SKILL_PYTHON_PATHS = (
    "internal/core/skill/catalog/cc/planning-with-files/scripts/session-catchup.py",
    "internal/core/skill/catalog/cc/react-vite-expert/scripts/analyze_bundle.py",
    "internal/core/skill/catalog/cc/react-vite-expert/scripts/create_component.py",
    "internal/core/skill/catalog/cc/react-vite-expert/scripts/create_hook.py",
    "internal/core/skill/catalog/codex/plugin-creator/scripts/create_basic_plugin.py",
    "internal/core/skill/catalog/codex/skill-creator/scripts/quick_validate.py",
)
EXCLUDED_PREFIXES = ("tests", "scripts")


class WorkspaceToolError(Exception):
    """Raised when the workspace tool cannot complete."""


@dataclass(frozen=True)
class CheckItem:
    name: str
    status: str
    detail: str


@dataclass
class WorkspaceCheckResult:
    status: str
    workspace_path: Path
    checks: list[CheckItem]
    errors: list[str]
    warnings: list[str]


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate gen-code workspace structure.")
    parser.add_argument("--workspace", help="Workspace root to inspect.")
    parser.add_argument(
        "--strict",
        action="store_true",
        help="Treat inventory drift warnings as failures.",
    )
    return parser.parse_args(argv)


def resolve_workspace(path_value: str | None) -> Path:
    workspace = Path(path_value).resolve() if path_value else ROOT_DIR
    if not workspace.exists():
        raise WorkspaceToolError(f"workspace path does not exist: {workspace}")
    if not workspace.is_dir():
        raise WorkspaceToolError(f"workspace path is not a directory: {workspace}")
    return workspace


def add_path_check(
    checks: list[CheckItem],
    errors: list[str],
    path: Path,
    label: str,
    *,
    directory: bool = False,
) -> None:
    exists = path.is_dir() if directory else path.is_file()
    if exists:
        checks.append(CheckItem(label, "ok", "present"))
        return
    checks.append(CheckItem(label, "fail", "missing"))
    kind = "directory" if directory else "file"
    errors.append(f"{label} {kind} is missing: {path}")


def gather_python_inventory(workspace: Path) -> list[str]:
    paths: list[str] = []
    for path in workspace.rglob("*.py"):
        if "__pycache__" in path.parts:
            continue
        relative = path.relative_to(workspace).as_posix()
        paths.append(relative)
    return sorted(paths)


def evaluate_workspace(workspace: Path, strict: bool) -> WorkspaceCheckResult:
    checks: list[CheckItem] = []
    errors: list[str] = []
    warnings: list[str] = []

    add_path_check(checks, errors, workspace / "go.mod", "go.mod")
    add_path_check(checks, errors, workspace / ".env.example", ".env.example")
    add_path_check(checks, errors, workspace / "tools", "tools", directory=True)
    add_path_check(checks, errors, workspace / "tests" / "tools", "tests/tools", directory=True)
    add_path_check(checks, errors, workspace / "internal" / "core" / "skill" / "catalog", "internal/core/skill/catalog", directory=True)

    for tool_name in EXPECTED_TOOLS:
        add_path_check(checks, errors, workspace / "tools" / tool_name, f"tools/{tool_name}")

    for skill_path in SKILL_PYTHON_PATHS:
        add_path_check(checks, errors, workspace / Path(skill_path), skill_path)

    inventory = gather_python_inventory(workspace)
    runtime_inventory = [item for item in inventory if item.startswith("tools/")]
    checks.append(CheckItem("python inventory", "ok", f"{len(inventory)} script(s) discovered"))
    checks.append(CheckItem("runtime tool inventory", "ok", ", ".join(runtime_inventory) if runtime_inventory else "none"))

    missing_skill_manifests: list[str] = []
    for skill_rel in (
        "internal/core/skill/catalog/cc/planning-with-files/skill.tools.json",
        "internal/core/skill/catalog/cc/react-vite-expert/skill.tools.json",
        "internal/core/skill/catalog/codex/plugin-creator/skill.tools.json",
        "internal/core/skill/catalog/codex/skill-creator/skill.tools.json",
    ):
        if not (workspace / skill_rel).is_file():
            missing_skill_manifests.append(skill_rel)

    if missing_skill_manifests:
        detail = ", ".join(missing_skill_manifests)
        checks.append(CheckItem("skill tool manifests", "fail" if strict else "warn", detail))
        message = f"missing skill tool manifests: {detail}"
        if strict:
            errors.append(message)
        else:
            warnings.append(message)
    else:
        checks.append(CheckItem("skill tool manifests", "ok", "present"))

    excluded_scripts = [item for item in inventory if any(item.startswith(prefix + "/") for prefix in EXCLUDED_PREFIXES)]
    checks.append(CheckItem("excluded python paths", "ok", f"{len(excluded_scripts)} excluded script(s)"))

    status = "FAIL" if errors else ("WARN" if warnings else "PASS")
    return WorkspaceCheckResult(
        status=status,
        workspace_path=workspace,
        checks=checks,
        errors=errors,
        warnings=warnings,
    )


def format_result(result: WorkspaceCheckResult) -> str:
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

    return "\n".join(lines)


def run(argv: list[str]) -> int:
    try:
        args = parse_args(argv)
        workspace = resolve_workspace(args.workspace)
        result = evaluate_workspace(workspace, args.strict)
        print(format_result(result))
        return 1 if result.status == "FAIL" else 0
    except WorkspaceToolError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 2


def main() -> int:
    return run(sys.argv[1:])


if __name__ == "__main__":
    raise SystemExit(main())
