import subprocess
import sys
import tempfile
import textwrap
import unittest
from pathlib import Path
from unittest import mock


REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "tools" / "check_runtime.py"


def write_file(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(textwrap.dedent(content).strip() + "\n", encoding="utf-8")


class CheckRuntimeModuleTest(unittest.TestCase):
    def setUp(self) -> None:
        from tools import check_runtime

        self.module = check_runtime

    def create_workspace(self) -> Path:
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        root = Path(tmp.name)
        write_file(root / "go.mod", "module example\n\ngo 1.25.0\n")
        write_file(root / ".env.example", "APP_PORT=10008\n")
        write_file(root / "tools" / "check_config.py", "#!/usr/bin/env python3\n")
        (root / "internal" / "core").mkdir(parents=True, exist_ok=True)
        return root

    def test_go_requirement_parser(self):
        workspace = self.create_workspace()
        version = self.module.parse_go_requirement(workspace / "go.mod")
        self.assertEqual((1, 25, 0), version)

    def test_missing_env_warns_by_default(self):
        workspace = self.create_workspace()
        with mock.patch.object(
            self.module,
            "run_command",
            side_effect=[
                (0, "go version go1.25.1 windows/amd64"),
                (0, "v22.14.0"),
            ],
        ):
            result = self.module.evaluate_runtime(workspace, require_env=False, strict=False)
        self.assertEqual("WARN", result.status)
        self.assertTrue(any(item.name == ".env" and item.status == "warn" for item in result.checks))

    def test_missing_env_fails_when_required(self):
        workspace = self.create_workspace()
        with mock.patch.object(
            self.module,
            "run_command",
            side_effect=[
                (0, "go version go1.25.1 windows/amd64"),
                (0, "v22.14.0"),
            ],
        ):
            result = self.module.evaluate_runtime(workspace, require_env=True, strict=False)
        self.assertEqual("FAIL", result.status)
        self.assertTrue(any(".env is missing" in item for item in result.errors))

    def test_missing_check_config_fails(self):
        workspace = self.create_workspace()
        (workspace / "tools" / "check_config.py").unlink()
        with mock.patch.object(
            self.module,
            "run_command",
            side_effect=[
                (0, "go version go1.25.1 windows/amd64"),
                (0, "v22.14.0"),
            ],
        ):
            result = self.module.evaluate_runtime(workspace, require_env=False, strict=False)
        self.assertEqual("FAIL", result.status)
        self.assertTrue(any("tools/check_config.py is missing" in item for item in result.errors))

    def test_go_version_below_requirement_fails(self):
        workspace = self.create_workspace()
        write_file(workspace / ".env", "APP_PORT=10008\n")
        with mock.patch.object(
            self.module,
            "run_command",
            side_effect=[
                (0, "go version go1.24.0 windows/amd64"),
                (0, "v22.14.0"),
            ],
        ):
            result = self.module.evaluate_runtime(workspace, require_env=False, strict=False)
        self.assertEqual("FAIL", result.status)
        self.assertTrue(any("Go version is below the version required by go.mod" in item for item in result.errors))

    def test_missing_node_warns(self):
        workspace = self.create_workspace()
        write_file(workspace / ".env", "APP_PORT=10008\n")
        with mock.patch.object(
            self.module,
            "run_command",
            side_effect=[
                (0, "go version go1.25.1 windows/amd64"),
                self.module.RuntimeToolError("node missing"),
            ],
        ):
            result = self.module.evaluate_runtime(workspace, require_env=False, strict=False)
        self.assertEqual("WARN", result.status)
        self.assertTrue(any("node executable is not available on PATH" in item for item in result.warnings))

    def test_missing_node_fails_in_strict_mode(self):
        workspace = self.create_workspace()
        write_file(workspace / ".env", "APP_PORT=10008\n")
        with mock.patch.object(
            self.module,
            "run_command",
            side_effect=[
                (0, "go version go1.25.1 windows/amd64"),
                self.module.RuntimeToolError("node missing"),
            ],
        ):
            result = self.module.evaluate_runtime(workspace, require_env=False, strict=True)
        self.assertEqual("FAIL", result.status)
        self.assertTrue(any("node executable is not available on PATH" in item for item in result.errors))


class CheckRuntimeCLITest(unittest.TestCase):
    def create_workspace(self) -> Path:
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        root = Path(tmp.name)
        write_file(root / "go.mod", "module example\n\ngo 1.24.0\n")
        write_file(root / ".env.example", "APP_PORT=10008\n")
        write_file(root / "tools" / "check_config.py", "#!/usr/bin/env python3\n")
        write_file(root / ".env", "APP_PORT=10008\n")
        (root / "internal" / "core").mkdir(parents=True, exist_ok=True)
        return root

    def run_tool(self, workspace: Path, *extra_args: str):
        cmd = [
            sys.executable,
            str(SCRIPT_PATH),
            "--workspace",
            str(workspace),
            *extra_args,
        ]
        return subprocess.run(cmd, capture_output=True, text=True, check=False)

    def test_cli_passes_in_healthy_workspace(self):
        workspace = self.create_workspace()
        result = self.run_tool(workspace)
        self.assertEqual(result.returncode, 0)
        self.assertIn("Status: PASS", result.stdout)
        self.assertIn("go: ok", result.stdout)

    def test_cli_fails_for_missing_workspace(self):
        missing = Path(tempfile.gettempdir()) / "missing-workspace-check-runtime"
        result = self.run_tool(missing)
        self.assertEqual(result.returncode, 2)
        self.assertIn("ERROR:", result.stderr)


if __name__ == "__main__":
    unittest.main()
