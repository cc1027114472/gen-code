#!/usr/bin/env python3
"""Project-local UI/UX design search helper.

Supports three main flows used by SKILL.md:
- --design-system
- --domain <domain>
- --stack <stack>
"""

from __future__ import annotations

import argparse
import csv
import json
from collections import Counter
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parent.parent
DATA_ROOT = ROOT / "data"
DOMAINS_PATH = DATA_ROOT / "domains.json"
STACKS_PATH = DATA_ROOT / "stacks.json"
REASONING_PATH = DATA_ROOT / "ui-reasoning.csv"


def load_json(path: Path) -> Any:
    with path.open("r", encoding="utf-8") as handle:
        return json.load(handle)


def load_reasoning_rules() -> list[dict[str, str]]:
    rows: list[dict[str, str]] = []
    with REASONING_PATH.open("r", encoding="utf-8-sig", newline="") as handle:
        reader = csv.DictReader(handle)
        for row in reader:
            rows.append({key: (value or "").strip() for key, value in row.items()})
    return rows


def tokenize(text: str) -> list[str]:
    normalized = []
    for raw in text.lower().replace("/", " ").replace("-", " ").split():
        token = raw.strip(",.()[]{}:;!?\"'")
        if token:
            normalized.append(token)
    return normalized


def score_entry(query_tokens: list[str], entry: dict[str, Any]) -> int:
    haystack = " ".join(
        str(entry.get(key, ""))
        for key in ("name", "summary", "tags", "use_for", "anti_patterns", "recommended_for", "keywords")
    ).lower()
    score = 0
    for token in query_tokens:
        if token in haystack:
            score += 3
    tags = [str(tag).lower() for tag in entry.get("tags", [])]
    for token in query_tokens:
        if token in tags:
            score += 2
    if not query_tokens:
        score += 1
    return score


def top_matches(entries: list[dict[str, Any]], query: str, limit: int) -> list[dict[str, Any]]:
    query_tokens = tokenize(query)
    ranked = sorted(
        entries,
        key=lambda item: (score_entry(query_tokens, item), item.get("priority", 0), item.get("name", "")),
        reverse=True,
    )
    filtered = [item for item in ranked if score_entry(query_tokens, item) > 0]
    if filtered:
        return filtered[:limit]
    return ranked[:limit]


def build_design_system(query: str, project_name: str | None) -> dict[str, Any]:
    domains = load_json(DOMAINS_PATH)
    rules = load_reasoning_rules()
    selected: dict[str, dict[str, Any]] = {}
    for domain in ("product", "style", "color", "landing", "typography"):
        selected[domain] = top_matches(domains[domain], query, 1)[0]

    tokens = tokenize(query)
    matched_rules = []
    for rule in rules:
        match_terms = tokenize(rule.get("match_terms", ""))
        overlap = len(set(match_terms) & set(tokens))
        if overlap > 0:
            matched_rules.append((overlap, rule))
    matched_rules.sort(key=lambda item: item[0], reverse=True)

    anti_patterns: list[str] = []
    rationale: list[str] = []
    for _, rule in matched_rules[:4]:
        if rule.get("rationale"):
            rationale.append(rule["rationale"])
        if rule.get("anti_patterns"):
            anti_patterns.extend([item.strip() for item in rule["anti_patterns"].split("|") if item.strip()])

    anti_patterns.extend(selected["style"].get("anti_patterns", []))
    anti_patterns.extend(selected["ux"].get("anti_patterns", []) if "ux" in selected else [])

    return {
        "project_name": project_name or "Unnamed Project",
        "query": query,
        "pattern": selected["product"]["name"],
        "style": selected["style"]["name"],
        "colors": selected["color"]["palette"],
        "typography": selected["typography"]["pairing"],
        "layout": selected["landing"]["structure"],
        "effects": selected["style"].get("effects", []),
        "rationale": rationale or [
            selected["product"]["summary"],
            selected["style"]["summary"],
            selected["color"]["summary"],
        ],
        "anti_patterns": list(dict.fromkeys(anti_patterns))[:8],
        "selected": selected,
    }


def render_design_system_ascii(system: dict[str, Any]) -> str:
    lines = [
        f"+ 项目: {system['project_name']}",
        f"+ 查询: {system['query']}",
        "",
        f"+ 产品模式: {system['pattern']}",
        f"+ 视觉风格: {system['style']}",
        f"+ 配色: {system['colors']}",
        f"+ 字体: {system['typography']}",
        f"+ 页面结构: {system['layout']}",
        f"+ 视觉效果: {', '.join(system['effects']) or '保持克制，优先信息层级'}",
        "",
        "理由:",
    ]
    for item in system["rationale"]:
        lines.append(f"- {item}")
    lines.append("")
    lines.append("避免事项:")
    for item in system["anti_patterns"]:
        lines.append(f"- {item}")
    return "\n".join(lines)


def render_design_system_markdown(system: dict[str, Any]) -> str:
    lines = [
        f"# {system['project_name']} Design System",
        "",
        f"- 查询：`{system['query']}`",
        f"- 产品模式：**{system['pattern']}**",
        f"- 视觉风格：**{system['style']}**",
        f"- 配色：**{system['colors']}**",
        f"- 字体：**{system['typography']}**",
        f"- 页面结构：**{system['layout']}**",
        f"- 视觉效果：{', '.join(system['effects']) or '保持克制，优先信息层级'}",
        "",
        "## 推荐理由",
    ]
    for item in system["rationale"]:
        lines.append(f"- {item}")
    lines.append("")
    lines.append("## 反模式")
    for item in system["anti_patterns"]:
        lines.append(f"- {item}")
    return "\n".join(lines)


def render_domain_results(domain: str, query: str, entries: list[dict[str, Any]]) -> str:
    lines = [f"# Domain: {domain}", f"", f"- 查询：`{query}`", ""]
    for index, entry in enumerate(entries, start=1):
        lines.append(f"## {index}. {entry['name']}")
        if entry.get("summary"):
            lines.append(entry["summary"])
        if entry.get("recommended_for"):
            lines.append(f"- 适合：{', '.join(entry['recommended_for'])}")
        if entry.get("tags"):
            lines.append(f"- 标签：{', '.join(entry['tags'])}")
        if entry.get("anti_patterns"):
            lines.append(f"- 避免：{', '.join(entry['anti_patterns'])}")
        if entry.get("details"):
            for detail in entry["details"]:
                lines.append(f"- {detail}")
        lines.append("")
    return "\n".join(lines).rstrip()


def render_stack_results(stack: str, query: str, entries: list[dict[str, Any]]) -> str:
    lines = [f"# Stack: {stack}", "", f"- 查询：`{query}`", ""]
    for index, entry in enumerate(entries, start=1):
        lines.append(f"## {index}. {entry['name']}")
        lines.append(entry["summary"])
        if entry.get("details"):
            for detail in entry["details"]:
                lines.append(f"- {detail}")
        lines.append("")
    return "\n".join(lines).rstrip()


def persist_design_system(system: dict[str, Any], page: str | None) -> list[Path]:
    design_root = ROOT / "design-system"
    pages_root = design_root / "pages"
    pages_root.mkdir(parents=True, exist_ok=True)
    master_path = design_root / "MASTER.md"
    master_path.write_text(render_design_system_markdown(system) + "\n", encoding="utf-8")
    written = [master_path]
    if page:
        page_path = pages_root / f"{page}.md"
        page_lines = [
            f"# {page} 页面覆盖规则",
            "",
            f"- 来源查询：`{system['query']}`",
            f"- 继承主风格：**{system['style']}**",
            "- 仅在该页面需要更高信息密度或不同模块优先级时覆盖 MASTER.md。",
            "- 若本页无额外覆盖，请回退到 MASTER.md 的全局规则。",
        ]
        page_path.write_text("\n".join(page_lines) + "\n", encoding="utf-8")
        written.append(page_path)
    return written


def main() -> int:
    parser = argparse.ArgumentParser(description="搜索 project-local UI/UX 设计情报库。")
    parser.add_argument("query", help="搜索关键词或组合描述")
    parser.add_argument("--design-system", action="store_true", help="生成一整套设计系统建议")
    parser.add_argument("--domain", choices=[
        "product", "style", "typography", "color", "landing", "chart", "ux", "google-fonts", "react", "web", "prompt"
    ], help="按领域检索")
    parser.add_argument("--stack", choices=["react-native"], help="按技术栈检索")
    parser.add_argument("-n", "--max-results", type=int, default=5, help="最多返回多少条结果")
    parser.add_argument("-p", "--project-name", help="项目名，用于 design-system 输出")
    parser.add_argument("--persist", action="store_true", help="把 design system 写入 design-system/ 目录")
    parser.add_argument("--page", help="可选页面名，配合 --persist 写页面覆盖文件")
    parser.add_argument("-f", "--format", choices=["ascii", "markdown"], default="ascii", help="design-system 输出格式")
    args = parser.parse_args()

    if not args.design_system and not args.domain and not args.stack:
        parser.error("必须至少提供 --design-system、--domain 或 --stack 之一。")

    if args.design_system:
        system = build_design_system(args.query, args.project_name)
        output = render_design_system_markdown(system) if args.format == "markdown" else render_design_system_ascii(system)
        print(output)
        if args.persist:
            written = persist_design_system(system, args.page)
            print("")
            print("已写入：")
            for path in written:
                print(f"- {path.relative_to(ROOT).as_posix()}")

    if args.domain:
        domains = load_json(DOMAINS_PATH)
        entries = top_matches(domains[args.domain], args.query, args.max_results)
        if args.design_system:
            print("")
        print(render_domain_results(args.domain, args.query, entries))

    if args.stack:
        stacks = load_json(STACKS_PATH)
        entries = top_matches(stacks[args.stack], args.query, args.max_results)
        if args.design_system or args.domain:
            print("")
        print(render_stack_results(args.stack, args.query, entries))

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
