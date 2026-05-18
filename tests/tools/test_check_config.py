import subprocess
import sys
import tempfile
import textwrap
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "tools" / "check_config.py"


def write_env(path: Path, content: str) -> None:
    path.write_text(textwrap.dedent(content).strip() + "\n", encoding="utf-8")


class CheckConfigCLITest(unittest.TestCase):
    def run_tool(self, env_text: str, example_text: str | None = None, *extra_args: str):
        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            env_path = tmp_path / ".env"
            example_path = tmp_path / ".env.example"
            write_env(env_path, env_text)
            write_env(
                example_path,
                example_text
                or """
                APP_NAME=llm-trace
                APP_ENV=dev
                APP_PORT=10008
                APP_SHUTDOWN_TIMEOUT=10s
                APP_DEBUG=true
                APP_TRUSTED_PROXIES=127.0.0.1
                LOG_HTTP_ACCESS=false
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

    def test_pass_when_config_matches_and_has_no_warnings(self):
        result = self.run_tool(
            """
            APP_NAME=llm-trace
            APP_ENV=dev
            APP_PORT=10008
            APP_SHUTDOWN_TIMEOUT=10s
            APP_DEBUG=true
            APP_TRUSTED_PROXIES=127.0.0.1
            LOG_HTTP_ACCESS=false
            """
        )
        self.assertEqual(result.returncode, 0)
        self.assertIn("Status: PASS", result.stdout)

    def test_fail_for_invalid_port(self):
        result = self.run_tool("APP_PORT=not-a-number")
        self.assertEqual(result.returncode, 1)
        self.assertIn("Status: FAIL", result.stdout)
        self.assertIn("APP_PORT must be an integer", result.stdout)

    def test_fail_for_invalid_duration(self):
        result = self.run_tool("APP_SHUTDOWN_TIMEOUT=0s")
        self.assertEqual(result.returncode, 1)
        self.assertIn("APP_SHUTDOWN_TIMEOUT must be greater than zero", result.stdout)

    def test_fail_for_invalid_provider_bool(self):
        result = self.run_tool("ANTHROPIC_ENABLED=sometimes")
        self.assertEqual(result.returncode, 1)
        self.assertIn("ANTHROPIC_ENABLED must be a Go-compatible boolean", result.stdout)

    def test_provider_is_inferred_enabled_from_model(self):
        result = self.run_tool("ANTHROPIC_MODEL=gpt-5.4-A")
        self.assertEqual(result.returncode, 0)
        self.assertIn("anthropic: enabled (inferred)", result.stdout)

    def test_provider_explicit_disable_wins(self):
        result = self.run_tool(
            """
            ANTHROPIC_ENABLED=false
            ANTHROPIC_BASE_URL=http://localhost:1314
            ANTHROPIC_AUTH_TOKEN=test-token
            ANTHROPIC_MODEL=gpt-5.4-A
            """
        )
        self.assertEqual(result.returncode, 0)
        self.assertIn("anthropic: disabled (explicit)", result.stdout)

    def test_unknown_key_warns_by_default(self):
        result = self.run_tool("UNKNOWN_FLAG=yes")
        self.assertEqual(result.returncode, 0)
        self.assertIn("Status: WARN", result.stdout)
        self.assertIn("unknown key not recognized", result.stdout)

    def test_unknown_key_fails_in_strict_mode(self):
        result = self.run_tool("UNKNOWN_FLAG=yes", None, "--strict")
        self.assertEqual(result.returncode, 1)
        self.assertIn("Status: FAIL", result.stdout)
        self.assertIn("unknown key not recognized", result.stdout)

    def test_empty_value_warns(self):
        result = self.run_tool("APP_NAME=")
        self.assertEqual(result.returncode, 0)
        self.assertIn("APP_NAME is set but empty", result.stdout)


if __name__ == "__main__":
    unittest.main()
