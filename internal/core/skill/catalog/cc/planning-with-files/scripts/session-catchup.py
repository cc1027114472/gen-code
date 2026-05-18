#!/usr/bin/env python3
"""
planning-with-files 的会话补齐脚本

分析上一次会话，找出在最近一次规划文件更新之后尚未同步的上下文。
设计目标是在 SessionStart 时运行。

用法：python3 session-catchup.py [project-path]
"""

import json
import sys
import os
from pathlib import Path
from typing import List, Dict, Optional, Tuple

PLANNING_FILES = ['task_plan.md', 'progress.md', 'findings.md']


def normalize_path(project_path: str) -> str:
    """规范化项目路径，使其匹配 Claude Code 的内部表示。

    Claude Code 使用 Windows 原生路径保存会话目录
    （例如 C:\\Users\\...），并将分隔符替换为短横线。
    Git Bash 传入的是 /c/Users/...，会生成不同的清洗结果。
    这个函数会先把 Git Bash 路径转换成 Windows 路径。
    """
    p = project_path

    # Git Bash / MSYS2: /c/Users/... -> C:/Users/...
    if len(p) >= 3 and p[0] == '/' and p[2] == '/':
        p = p[1].upper() + ':' + p[2:]

    # 解析为绝对路径，以处理相对路径和符号链接
    try:
        resolved = str(Path(p).resolve())
        # 在 Windows 上，resolve() 会返回 C:\Users\...，这正是我们需要的格式
        if os.name == 'nt' or '\\' in resolved:
            p = resolved
    except (OSError, ValueError):
        pass

    return p


def get_project_dir(project_path: str) -> Tuple[Optional[Path], Optional[str]]:
    """为当前运行时变体解析会话存储路径。"""
    normalized = normalize_path(project_path)

    # Claude Code 的清洗规则：将路径分隔符和 : 替换为 -
    sanitized = normalized.replace('\\', '-').replace('/', '-').replace(':', '-')
    sanitized = sanitized.replace('_', '-')
    # 如果存在前导短横线则去掉（Unix 绝对路径以 / 开头）
    if sanitized.startswith('-'):
        sanitized = sanitized[1:]

    claude_path = Path.home() / '.claude' / 'projects' / sanitized

    # Codex 将会话保存在 ~/.codex/sessions，并使用不同格式。
    # 从 Codex skill 目录运行时，避免悄悄去扫描 Claude 的路径。
    script_path = Path(__file__).as_posix().lower()
    is_codex_variant = '/.codex/' in script_path
    codex_sessions_dir = Path.home() / '.codex' / 'sessions'
    if is_codex_variant and codex_sessions_dir.exists() and not claude_path.exists():
        return None, (
            "[planning-with-files] 已跳过会话补齐：Codex 将会话保存在 ~/.codex/sessions，"
            "当前尚未实现原生 Codex 会话解析。"
        )

    return claude_path, None


def get_sessions_sorted(project_dir: Path) -> List[Path]:
    """获取所有会话文件，并按修改时间排序（最新的在前）。"""
    sessions = list(project_dir.glob('*.jsonl'))
    main_sessions = [s for s in sessions if not s.name.startswith('agent-')]
    return sorted(main_sessions, key=lambda p: p.stat().st_mtime, reverse=True)


def parse_session_messages(session_file: Path) -> List[Dict]:
    """解析会话文件中的所有消息，并保持原始顺序。"""
    messages = []
    with open(session_file, 'r', encoding='utf-8', errors='replace') as f:
        for line_num, line in enumerate(f):
            try:
                data = json.loads(line)
                data['_line_num'] = line_num
                messages.append(data)
            except json.JSONDecodeError:
                pass
    return messages


def find_last_planning_update(messages: List[Dict]) -> Tuple[int, Optional[str]]:
    """
    找出最近一次写入/编辑规划文件的时刻。
    如果找到则返回 (line_number, filename)，否则返回 (-1, None)。
    """
    last_update_line = -1
    last_update_file = None

    for msg in messages:
        msg_type = msg.get('type')

        if msg_type == 'assistant':
            content = msg.get('message', {}).get('content', [])
            if isinstance(content, list):
                for item in content:
                    if item.get('type') == 'tool_use':
                        tool_name = item.get('name', '')
                        tool_input = item.get('input', {})

                        if tool_name in ('Write', 'Edit'):
                            file_path = tool_input.get('file_path', '')
                            for pf in PLANNING_FILES:
                                if file_path.endswith(pf):
                                    last_update_line = msg['_line_num']
                                    last_update_file = pf

    return last_update_line, last_update_file


def extract_messages_after(messages: List[Dict], after_line: int) -> List[Dict]:
    """提取某一行号之后的对话消息。"""
    result = []
    for msg in messages:
        if msg['_line_num'] <= after_line:
            continue

        msg_type = msg.get('type')
        is_meta = msg.get('isMeta', False)

        if msg_type == 'user' and not is_meta:
            content = msg.get('message', {}).get('content', '')
            if isinstance(content, list):
                for item in content:
                    if isinstance(item, dict) and item.get('type') == 'text':
                        content = item.get('text', '')
                        break
                else:
                    content = ''

            if content and isinstance(content, str):
                if content.startswith(('<local-command', '<command-', '<task-notification')):
                    continue
                if len(content) > 20:
                    result.append({'role': 'user', 'content': content, 'line': msg['_line_num']})

        elif msg_type == 'assistant':
            msg_content = msg.get('message', {}).get('content', '')
            text_content = ''
            tool_uses = []

            if isinstance(msg_content, str):
                text_content = msg_content
            elif isinstance(msg_content, list):
                for item in msg_content:
                    if item.get('type') == 'text':
                        text_content = item.get('text', '')
                    elif item.get('type') == 'tool_use':
                        tool_name = item.get('name', '')
                        tool_input = item.get('input', {})
                        if tool_name == 'Edit':
                            tool_uses.append(f"Edit: {tool_input.get('file_path', 'unknown')}")
                        elif tool_name == 'Write':
                            tool_uses.append(f"Write: {tool_input.get('file_path', 'unknown')}")
                        elif tool_name == 'Bash':
                            cmd = tool_input.get('command', '')[:80]
                            tool_uses.append(f"Bash: {cmd}")
                        else:
                            tool_uses.append(f"{tool_name}")

            if text_content or tool_uses:
                result.append({
                    'role': 'assistant',
                    'content': text_content[:600] if text_content else '',
                    'tools': tool_uses,
                    'line': msg['_line_num']
                })

    return result


def main():
    project_path = sys.argv[1] if len(sys.argv) > 1 else os.getcwd()

    # 检查规划文件是否存在（存在则说明当前任务已启用该模式）
    has_planning_files = any(
        Path(project_path, f).exists() for f in PLANNING_FILES
    )
    if not has_planning_files:
        # 该项目没有规划文件；跳过补齐以避免产生噪音。
        return

    project_dir, skip_reason = get_project_dir(project_path)
    if skip_reason:
        print(skip_reason)
        return

    if not project_dir.exists():
        # 没有历史会话，无需补齐
        return

    sessions = get_sessions_sorted(project_dir)
    if len(sessions) < 1:
        return

    # 找一个内容足够丰富的历史会话
    target_session = None
    for session in sessions:
        if session.stat().st_size > 5000:
            target_session = session
            break

    if not target_session:
        return

    messages = parse_session_messages(target_session)
    last_update_line, last_update_file = find_last_planning_update(messages)

    # 目标会话中没有规划更新，跳过补齐输出。
    if last_update_line < 0:
        return

    # 只有在存在未同步内容时才输出
    messages_after = extract_messages_after(messages, last_update_line)

    if not messages_after:
        return

    # 输出补齐报告
    print("\n[planning-with-files] 检测到会话补齐信息")
    print(f"上一会话：{target_session.stem}")

    print(f"最近一次规划更新：{last_update_file}，消息编号 #{last_update_line}")
    print(f"未同步消息数：{len(messages_after)}")

    print("\n--- 未同步上下文 ---")
    for msg in messages_after[-15:]:  # Last 15 messages
        if msg['role'] == 'user':
            print(f"用户：{msg['content'][:300]}")
        else:
            if msg.get('content'):
                print(f"Claude：{msg['content'][:300]}")
            if msg.get('tools'):
                print(f"  工具：{', '.join(msg['tools'][:4])}")

    print("\n--- 建议操作 ---")
    print("1. 运行：git diff --stat")
    print("2. 阅读：task_plan.md、progress.md、findings.md")
    print("3. 根据以上上下文更新规划文件")
    print("4. 继续当前任务")


if __name__ == '__main__':
    main()
