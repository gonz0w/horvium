# SPDX-License-Identifier: BSD-3-Clause
"""Regression coverage for foundation gates, using disposable Git repositories."""
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SOURCE = Path(__file__).resolve().parents[2]


class FoundationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="horvium-foundation-")
        self.addCleanup(self.temp.cleanup)
        self.repo = Path(self.temp.name) / "repo"
        shutil.copytree(SOURCE, self.repo, ignore=shutil.ignore_patterns(".git", ".tools", "__pycache__"))
        self.git("init", "-b", "main")
        self.git("config", "user.name", "Test Author")
        self.git("config", "user.email", "author@example.invalid")
        self.git("config", "commit.gpgsign", "false")
        self.git("config", "core.hooksPath", "/dev/null")

    def command(self, *args):
        return subprocess.run(args, cwd=self.repo, capture_output=True, text=True,
                              env={**os.environ, "GIT_CONFIG_NOSYSTEM": "1"})

    def git(self, *args):
        result = self.command("git", *args)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        return result.stdout.strip()

    def check(self):
        return self.command(sys.executable, "tools/check-repo.py")

    def commit(self, message, signoff=False):
        self.git("add", ".")
        args = ["commit", "-m", message]
        if signoff:
            args.append("-s")
        self.git(*args)
        return self.git("rev-parse", "HEAD")

    def test_committed_whitespace_is_rejected_without_staged_changes(self):
        (self.repo / "probe.py").write_text("value = 1   \n")
        self.commit("test: add whitespace probe")
        self.assertEqual(self.git("diff", "--cached", "--name-only"), "")
        result = self.check()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("probe.py: line 1: trailing whitespace", result.stderr)

    def test_license_is_required(self):
        (self.repo / "LICENSE").unlink()
        self.assertIn("LICENSE: required file", self.check().stderr)

    def test_markdown_hard_breaks_are_allowed(self):
        (self.repo / "probe.md").write_text("Intentional hard break  \nNext line\n")
        self.git("add", ".")
        result = self.check()
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_mutable_action_reference_is_rejected(self):
        path = self.repo / ".github/workflows/repository.yml"
        path.write_text(path.read_text().replace("@3d3c42e5aac5ba805825da76410c181273ba90b1", "@main"))
        self.assertIn("external action must use a full commit SHA", self.check().stderr)

    def test_duplicate_yaml_security_keys_are_rejected(self):
        path = self.repo / ".github/workflows/repository.yml"
        path.write_text(path.read_text() + "\npermissions:\n  contents: write\n")
        self.assertIn("duplicate YAML key: permissions", self.check().stderr)

    def test_secret_filename_with_spaces_is_rejected(self):
        (self.repo / "key with spaces.key").write_text("not a real key\n")
        self.assertIn("key with spaces.key: private-key", self.check().stderr)

    def test_missing_local_link_is_rejected(self):
        (self.repo / "probe.md").write_text("[missing](missing-document.md)\n")
        self.assertIn("missing local link target", self.check().stderr)

    def test_dco_requires_author_trailer_on_every_commit(self):
        base = self.commit("chore: create fixture", signoff=True)
        (self.repo / "probe.md").write_text("Probe\n")
        head = self.commit("docs: add probe")
        result = self.command(sys.executable, "tools/check-dco.py", base, head)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Missing author DCO sign-off", result.stderr)
        self.git("commit", "--amend", "--no-edit", "-s")
        head = self.git("rev-parse", "HEAD")
        result = self.command(sys.executable, "tools/check-dco.py", base, head)
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_commit_hook_rejects_missing_or_wrong_signoff(self):
        message = self.repo / "message"
        for body in ("docs: add probe\n", "docs: add probe\n\nSigned-off-by: Other <other@example.invalid>\n"):
            message.write_text(body)
            self.assertNotEqual(self.command("sh", ".githooks/commit-msg", str(message)).returncode, 0)
        message.write_text("docs: add probe\n\nSigned-off-by: Test Author <author@example.invalid>\n")
        self.assertEqual(self.command("sh", ".githooks/commit-msg", str(message)).returncode, 0)


if __name__ == "__main__":
    unittest.main()
