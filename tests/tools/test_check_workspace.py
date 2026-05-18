import subprocess
import sys
import tempfile
import textwrap
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "tools" / "check_workspace.py"


def write_file(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(textwrap.dedent(content).strip() + "\n", encoding="utf-8")


class CheckWorkspaceCLITest(unittest.TestCase):
    def create_workspace(self) -> Path:
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        root = Path(tmp.name)
        write_file(root / "go.mod", "module example\n\ngo 1.25.0\n")
        write_file(root / ".env.example", "APP_PORT=10008\n")
        write_file(root / "tools" / "check_config.py", "#!/usr/bin/env python3\n")
        write_file(root / "tools" / "check_runtime.py", "#!/usr/bin/env python3\n")
        write_file(root / "tools" / "check_workspace.py", "#!/usr/bin/env python3\n")
        write_file(root / "internal" / "core" / "skill" / "catalog" / "cc" / "planning-with-files" / "scripts" / "session-catchup.py", "print('ok')\n")
        write_file(root / "internal" / "core" / "skill" / "catalog" / "cc" / "planning-with-files" / "skill.tools.json", "{}\n")
        write_file(root / "internal" / "core" / "skill" / "catalog" / "cc" / "react-vite-expert" / "scripts" / "analyze_bundle.py", "print('ok')\n")
        write_file(root / "internal" / "core" / "skill" / "catalog" / "cc" / "react-vite-expert" / "scripts" / "create_component.py", "print('ok')\n")
        write_file(root / "internal" / "core" / "skill" / "catalog" / "cc" / "react-vite-expert" / "scripts" / "create_hook.py", "print('ok')\n")
        write_file(root / "internal" / "core" / "skill" / "catalog" / "cc" / "react-vite-expert" / "skill.tools.json", "{}\n")
        write_file(root / "internal" / "core" / "skill" / "catalog" / "codex" / "plugin-creator" / "scripts" / "create_basic_plugin.py", "print('ok')\n")
        write_file(root / "internal" / "core" / "skill" / "catalog" / "codex" / "plugin-creator" / "skill.tools.json", "{}\n")
        write_file(root / "internal" / "core" / "skill" / "catalog" / "codex" / "skill-creator" / "scripts" / "quick_validate.py", "print('ok')\n")
        write_file(root / "internal" / "core" / "skill" / "catalog" / "codex" / "skill-creator" / "skill.tools.json", "{}\n")
        (root / "tests" / "tools").mkdir(parents=True, exist_ok=True)
        return root

    def run_tool(self, workspace: Path, *extra_args: str):
        cmd = [sys.executable, str(SCRIPT_PATH), "--workspace", str(workspace), *extra_args]
        return subprocess.run(cmd, capture_output=True, text=True, check=False)

    def test_cli_passes_for_complete_workspace(self):
        workspace = self.create_workspace()
        result = self.run_tool(workspace)
        self.assertEqual(result.returncode, 0)
        self.assertIn("Status: PASS", result.stdout)

    def test_cli_warns_when_manifest_missing(self):
        workspace = self.create_workspace()
        (workspace / "internal" / "core" / "skill" / "catalog" / "cc" / "react-vite-expert" / "skill.tools.json").unlink()
        result = self.run_tool(workspace)
        self.assertEqual(result.returncode, 0)
        self.assertIn("Status: WARN", result.stdout)
        self.assertIn("missing skill tool manifests", result.stdout)


if __name__ == "__main__":
    unittest.main()
