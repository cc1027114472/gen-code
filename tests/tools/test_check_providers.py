import subprocess
import sys
import tempfile
import textwrap
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "tools" / "check_providers.py"


def write_file(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(textwrap.dedent(content).strip() + "\n", encoding="utf-8")


class CheckProvidersCLITest(unittest.TestCase):
    def run_tool(self, env_text: str, example_text: str | None = None, *extra_args: str):
        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            env_path = tmp_path / ".env"
            example_path = tmp_path / ".env.example"
            write_file(env_path, env_text)
            write_file(
                example_path,
                example_text
                or """
                APP_PORT=10008
                APP_SHUTDOWN_TIMEOUT=10s
                ANTHROPIC_ENABLED=true
                ANTHROPIC_MODEL=claude-3-7-sonnet
                """,
            )
            cmd = [
                sys.executable,
                str(SCRIPT_PATH),
                "--env-file",
                str(env_path),
                "--example-file",
                str(example_path),
                *extra_args,
            ]
            return subprocess.run(cmd, capture_output=True, text=True, check=False)

    def test_pass_with_enabled_provider(self):
        result = self.run_tool(
            """
            APP_PORT=10008
            APP_SHUTDOWN_TIMEOUT=10s
            ANTHROPIC_ENABLED=true
            ANTHROPIC_MODEL=claude-3-7-sonnet
            ANTHROPIC_AUTH_TOKEN=token
            """
        )
        self.assertEqual(result.returncode, 0)
        self.assertIn("Status: PASS", result.stdout)

    def test_fail_when_enabled_provider_missing_token(self):
        result = self.run_tool(
            """
            APP_PORT=10008
            APP_SHUTDOWN_TIMEOUT=10s
            ANTHROPIC_ENABLED=true
            ANTHROPIC_MODEL=claude-3-7-sonnet
            """
        )
        self.assertEqual(result.returncode, 1)
        self.assertIn("missing auth token", result.stdout)

    def test_strict_promotes_warning(self):
        result = self.run_tool(
            """
            APP_PORT=10008
            APP_SHUTDOWN_TIMEOUT=10s
            ANTHROPIC_ENABLED=false
            ANTHROPIC_MODEL=claude-3-7-sonnet
            ANTHROPIC_AUTH_TOKEN=token
            """,
            None,
            "--strict",
        )
        self.assertEqual(result.returncode, 1)
        self.assertIn("strict mode", result.stdout)


if __name__ == "__main__":
    unittest.main()
