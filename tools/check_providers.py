#!/usr/bin/env python3
"""Validate provider-oriented configuration expectations for gen-code."""

from __future__ import annotations

import argparse
import sys
from dataclasses import dataclass
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parent.parent
if str(ROOT_DIR) not in sys.path:
    sys.path.insert(0, str(ROOT_DIR))

from tools import check_config


class ProviderToolError(Exception):
    """Raised when the provider tool cannot complete."""


@dataclass
class ProviderCheckResult:
    status: str
    env_path: Path
    example_path: Path
    errors: list[str]
    warnings: list[str]
    provider_lines: list[str]


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate gen-code provider configuration.")
    parser.add_argument("--env-file", help="Path to the .env file to validate.")
    parser.add_argument("--example-file", help="Path to the .env.example baseline file.")
    parser.add_argument(
        "--strict",
        action="store_true",
        help="Treat provider warnings as failures.",
    )
    return parser.parse_args(argv)


def resolve_paths(args: argparse.Namespace) -> tuple[Path, Path]:
    env_path = Path(args.env_file).resolve() if args.env_file else check_config.default_env_path()
    example_path = Path(args.example_file).resolve() if args.example_file else ROOT_DIR / ".env.example"
    return env_path, example_path


def validate_provider_fields(
    env: dict[str, str],
    summaries: list[check_config.ProviderSummary],
    strict: bool,
) -> tuple[list[str], list[str], list[str]]:
    errors: list[str] = []
    warnings: list[str] = []
    provider_lines: list[str] = []

    for summary in summaries:
        spec = check_config.PROVIDER_SPECS[summary.name]
        base_url = check_config.first_non_empty(env, spec["base_url_keys"])
        auth_token = check_config.first_non_empty(env, spec["auth_token_keys"])
        model = check_config.first_non_empty(env, spec["model_keys"])

        provider_lines.append(
            f"{summary.name}: enabled={'true' if summary.enabled else 'false'} "
            f"source={summary.source} "
            f"baseUrl={'yes' if base_url else 'no'} "
            f"authToken={'yes' if auth_token else 'no'} "
            f"model={'yes' if model else 'no'}"
        )

        if summary.enabled:
            if not auth_token:
                errors.append(f"{summary.name} is enabled but missing auth token/api key")
            if not model:
                warnings.append(f"{summary.name} is enabled but missing model selection")
            if base_url and not (base_url.startswith("http://") or base_url.startswith("https://")):
                errors.append(f"{summary.name} base URL must start with http:// or https://")
        else:
            if summary.source == "explicit" and (base_url or auth_token or model):
                warnings.append(f"{summary.name} is explicitly disabled but still has configured provider fields")

    if strict and warnings:
        errors.extend(f"strict mode: {item}" for item in warnings)
        warnings = []

    return errors, warnings, provider_lines


def evaluate_providers(env_path: Path, example_path: Path, strict: bool) -> ProviderCheckResult:
    if not env_path.is_file():
        raise ProviderToolError(f".env file not found: {env_path}")
    if not example_path.is_file():
        raise ProviderToolError(f".env.example file not found: {example_path}")

    env = check_config.parse_env_file(env_path)
    example_env = check_config.parse_env_file(example_path)
    base = check_config.validate_env(env, example_env, env_path, example_path, strict=False)

    errors = list(base.errors)
    warnings = list(base.warnings)
    provider_errors, provider_warnings, provider_lines = validate_provider_fields(env, base.provider_summaries, strict)
    errors.extend(provider_errors)
    warnings.extend(provider_warnings)

    status = "FAIL" if errors else ("WARN" if warnings else "PASS")
    return ProviderCheckResult(
        status=status,
        env_path=env_path,
        example_path=example_path,
        errors=errors,
        warnings=warnings,
        provider_lines=provider_lines,
    )


def format_result(result: ProviderCheckResult) -> str:
    lines = [
        f"Status: {result.status}",
        f"Env file: {result.env_path}",
        f"Example file: {result.example_path}",
        "",
        "Provider checks:",
    ]
    for line in result.provider_lines:
        lines.append(f"  - {line}")

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
        env_path, example_path = resolve_paths(args)
        result = evaluate_providers(env_path, example_path, args.strict)
        print(format_result(result))
        return 1 if result.status == "FAIL" else 0
    except (ProviderToolError, check_config.ConfigToolError) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 2


def main() -> int:
    return run(sys.argv[1:])


if __name__ == "__main__":
    raise SystemExit(main())
