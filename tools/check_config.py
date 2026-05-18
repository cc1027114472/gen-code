#!/usr/bin/env python3
"""Validate .env configuration against gen-code's Go runtime rules."""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable


ROOT_DIR = Path(__file__).resolve().parent.parent
DEFAULT_ENV_BASENAME = ".env"
DEFAULT_EXAMPLE_BASENAME = ".env.example"

BOOL_TRUE = {"1", "t", "T", "true", "TRUE", "True"}
BOOL_FALSE = {"0", "f", "F", "false", "FALSE", "False"}
BOOL_VALUES = BOOL_TRUE | BOOL_FALSE

PROVIDER_SPECS = {
    "anthropic": {
        "enabled_keys": ["ANTHROPIC_ENABLED"],
        "base_url_keys": ["ANTHROPIC_BASE_URL"],
        "auth_token_keys": ["ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY"],
        "model_keys": [
            "ANTHROPIC_MODEL",
            "ANTHROPIC_DEFAULT_HAIKU_MODEL",
            "ANTHROPIC_DEFAULT_SONNET_MODEL",
            "ANTHROPIC_DEFAULT_OPUS_MODEL",
        ],
    },
    "openai": {
        "enabled_keys": ["OPENAI_ENABLED"],
        "base_url_keys": ["OPENAI_BASE_URL"],
        "auth_token_keys": ["OPENAI_AUTH_TOKEN", "OPENAI_API_KEY"],
        "model_keys": [
            "OPENAI_MODEL",
            "OPENAI_DEFAULT_MINI_MODEL",
            "OPENAI_DEFAULT_MODEL",
            "OPENAI_DEFAULT_REASONING_MODEL",
        ],
    },
    "gemini": {
        "enabled_keys": ["GEMINI_ENABLED"],
        "base_url_keys": ["GEMINI_BASE_URL"],
        "auth_token_keys": ["GEMINI_AUTH_TOKEN", "GEMINI_API_KEY"],
        "model_keys": [
            "GEMINI_MODEL",
            "GEMINI_DEFAULT_FLASH_MODEL",
            "GEMINI_DEFAULT_PRO_MODEL",
            "GEMINI_DEFAULT_ULTRA_MODEL",
        ],
    },
}

RESERVED_KEYS = {
    "SOFFICE_PATH",
    "LLM_LOG_BODY_MAX",
    "LLM_LOG_ERR_SNIPPET",
    "MULTI_COMPANY_RZ_RESUME_ON_START",
}

DURATION_PATTERN = re.compile(r"([+-]?(?:\d+(?:\.\d*)?|\.\d+))(ns|us|µs|μs|ms|s|m|h)")
DURATION_MULTIPLIERS = {
    "ns": 1e-9,
    "us": 1e-6,
    "µs": 1e-6,
    "μs": 1e-6,
    "ms": 1e-3,
    "s": 1.0,
    "m": 60.0,
    "h": 3600.0,
}


class ConfigToolError(Exception):
    """Raised when the tool cannot complete due to input or file issues."""


@dataclass(frozen=True)
class ProviderSummary:
    name: str
    enabled: bool
    source: str


@dataclass
class ValidationResult:
    status: str
    env_path: Path
    example_path: Path
    errors: list[str]
    warnings: list[str]
    provider_summaries: list[ProviderSummary]


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate gen-code .env configuration.")
    parser.add_argument("--env-file", help="Path to the .env file to validate.")
    parser.add_argument("--example-file", help="Path to the .env.example baseline file.")
    parser.add_argument(
        "--strict",
        action="store_true",
        help="Treat missing or unknown .env.example keys as failures.",
    )
    return parser.parse_args(argv)


def parse_env_file(path: Path) -> dict[str, str]:
    try:
        content = path.read_text(encoding="utf-8")
    except OSError as exc:
        raise ConfigToolError(f"failed to read {path}: {exc}") from exc

    env: dict[str, str] = {}
    for raw_line in content.splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        if "=" not in line:
            continue
        key, value = line.split("=", 1)
        key = key.strip()
        value = value.strip()
        if not key:
            continue
        env[key] = value
    return env


def find_dotenv_path(start_dir: Path) -> Path | None:
    current = start_dir.resolve()
    while True:
        candidate = current / DEFAULT_ENV_BASENAME
        if candidate.is_file():
            return candidate
        if current.parent == current:
            return None
        current = current.parent


def default_env_path() -> Path:
    found = find_dotenv_path(Path.cwd())
    if found is None:
        raise ConfigToolError("could not find .env from the current working directory upward")
    return found


def bool_from_string(key: str, value: str) -> bool:
    if value in BOOL_TRUE:
        return True
    if value in BOOL_FALSE:
        return False
    raise ValueError(f"{key} must be a Go-compatible boolean, got {value!r}")


def parse_int(key: str, value: str) -> int:
    try:
        return int(value, 10)
    except ValueError as exc:
        raise ValueError(f"{key} must be an integer, got {value!r}") from exc


def parse_positive_duration(key: str, value: str) -> float:
    if not value:
        raise ValueError(f"{key} must be a duration, got an empty value")
    pos = 0
    total = 0.0
    matched = False
    while pos < len(value):
        match = DURATION_PATTERN.match(value, pos)
        if not match:
            raise ValueError(f"{key} must be a Go duration, got {value!r}")
        matched = True
        amount = float(match.group(1))
        unit = match.group(2)
        total += amount * DURATION_MULTIPLIERS[unit]
        pos = match.end()
    if not matched:
        raise ValueError(f"{key} must be a Go duration, got {value!r}")
    if total <= 0:
        raise ValueError(f"{key} must be greater than zero")
    return total


def parse_csv(value: str) -> list[str]:
    return [item.strip() for item in value.split(",") if item.strip()]


def first_non_empty(env: dict[str, str], keys: Iterable[str]) -> str:
    for key in keys:
        value = env.get(key, "")
        if value != "":
            return value
    return ""


def first_present_non_empty(env: dict[str, str], keys: Iterable[str]) -> tuple[bool, str]:
    for key in keys:
        if key not in env:
            continue
        value = env[key]
        if value == "":
            continue
        return True, value
    return False, ""


def infer_provider_state(name: str, env: dict[str, str]) -> ProviderSummary:
    spec = PROVIDER_SPECS[name]
    present, enabled_value = first_present_non_empty(env, spec["enabled_keys"])
    if present:
        enabled = bool_from_string(spec["enabled_keys"][0], enabled_value)
        return ProviderSummary(name=name, enabled=enabled, source="explicit")

    enabled = any(
        first_non_empty(env, keys)
        for keys in (
            spec["base_url_keys"],
            spec["auth_token_keys"],
            spec["model_keys"],
        )
    )
    return ProviderSummary(name=name, enabled=enabled, source="inferred")


def known_keys(example_env: dict[str, str]) -> set[str]:
    keys = set(example_env)
    keys.update({"MODEL_PROVIDER", "LLM_PROVIDER", "PROVIDER_DEFAULT"})
    for spec in PROVIDER_SPECS.values():
        keys.update(spec["enabled_keys"])
        keys.update(spec["base_url_keys"])
        keys.update(spec["auth_token_keys"])
        keys.update(spec["model_keys"])
    keys.update(RESERVED_KEYS)
    return keys


def validate_env(
    env: dict[str, str],
    example_env: dict[str, str],
    env_path: Path,
    example_path: Path,
    strict: bool,
) -> ValidationResult:
    errors: list[str] = []
    warnings: list[str] = []

    if not env_path.is_file():
        raise ConfigToolError(f".env file not found: {env_path}")
    if not example_path.is_file():
        raise ConfigToolError(f".env.example file not found: {example_path}")

    if not env:
        errors.append(f"{env_path} does not contain any usable KEY=VALUE entries")

    example_keys = set(example_env)
    env_keys = set(env)
    missing_keys = sorted(example_keys - env_keys)
    unknown = sorted(env_keys - known_keys(example_env))

    for key in missing_keys:
        message = f"missing example key: {key}"
        if strict:
            errors.append(message)
        else:
            warnings.append(message)

    for key in unknown:
        message = f"unknown key not recognized by the current baseline: {key}"
        if strict:
            errors.append(message)
        else:
            warnings.append(message)

    for key, value in env.items():
        if value == "":
            warnings.append(f"{key} is set but empty; Go will treat it as unset")

    parse_int_if_present("APP_PORT", env, errors)
    parse_bool_if_present("APP_DEBUG", env, errors)
    parse_bool_if_present("LOG_HTTP_ACCESS", env, errors)
    parse_duration_if_present("APP_SHUTDOWN_TIMEOUT", env, errors)

    if "APP_TRUSTED_PROXIES" in env and env["APP_TRUSTED_PROXIES"] != "":
        proxies = parse_csv(env["APP_TRUSTED_PROXIES"])
        if not proxies:
            warnings.append("APP_TRUSTED_PROXIES resolves to an empty list after trimming")

    parse_provider_bools(env, errors)
    provider_summaries = collect_provider_summaries(env, errors)

    status = "FAIL" if errors else ("WARN" if warnings else "PASS")
    return ValidationResult(
        status=status,
        env_path=env_path,
        example_path=example_path,
        errors=errors,
        warnings=warnings,
        provider_summaries=provider_summaries,
    )


def parse_int_if_present(key: str, env: dict[str, str], errors: list[str]) -> None:
    value = env.get(key, "")
    if value == "":
        return
    try:
        parse_int(key, value)
    except ValueError as exc:
        errors.append(str(exc))


def parse_bool_if_present(key: str, env: dict[str, str], errors: list[str]) -> None:
    value = env.get(key, "")
    if value == "":
        return
    try:
        bool_from_string(key, value)
    except ValueError as exc:
        errors.append(str(exc))


def parse_duration_if_present(key: str, env: dict[str, str], errors: list[str]) -> None:
    value = env.get(key, "")
    if value == "":
        return
    try:
        parse_positive_duration(key, value)
    except ValueError as exc:
        errors.append(str(exc))


def parse_provider_bools(env: dict[str, str], errors: list[str]) -> None:
    for spec in PROVIDER_SPECS.values():
        for key in spec["enabled_keys"]:
            parse_bool_if_present(key, env, errors)


def collect_provider_summaries(env: dict[str, str], errors: list[str]) -> list[ProviderSummary]:
    summaries: list[ProviderSummary] = []
    for name in PROVIDER_SPECS:
        try:
            summaries.append(infer_provider_state(name, env))
        except ValueError as exc:
            errors.append(str(exc))
    return summaries


def format_result(result: ValidationResult) -> str:
    lines = [
        f"Status: {result.status}",
        f"Env file: {result.env_path}",
        f"Example file: {result.example_path}",
        "",
        "Provider summary:",
    ]
    for summary in result.provider_summaries:
        enabled_text = "enabled" if summary.enabled else "disabled"
        lines.append(f"  - {summary.name}: {enabled_text} ({summary.source})")

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


def resolve_paths(args: argparse.Namespace) -> tuple[Path, Path]:
    env_path = Path(args.env_file).resolve() if args.env_file else default_env_path()
    example_path = (
        Path(args.example_file).resolve()
        if args.example_file
        else ROOT_DIR / DEFAULT_EXAMPLE_BASENAME
    )
    return env_path, example_path


def run(argv: list[str]) -> int:
    try:
        args = parse_args(argv)
        env_path, example_path = resolve_paths(args)
        env = parse_env_file(env_path)
        example_env = parse_env_file(example_path)
        result = validate_env(env, example_env, env_path, example_path, args.strict)
        print(format_result(result))
        return 1 if result.status == "FAIL" else 0
    except ConfigToolError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 2


def main() -> int:
    return run(sys.argv[1:])


if __name__ == "__main__":
    raise SystemExit(main())
