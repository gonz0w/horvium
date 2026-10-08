# SPDX-License-Identifier: BSD-3-Clause
"""Regression coverage for foundation gates, using disposable Git repositories."""
import os
import hashlib
import io
import json
import platform
import shutil
import subprocess
import sys
import tempfile
import tarfile
import unittest
from unittest import mock
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
        # Disposable fixture commits must not outlive their synchronous command.
        # Do not inherit a policy that starts detached maintenance during teardown.
        self.git("config", "maintenance.auto", "false")
        self.git("config", "gc.auto", "0")

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
        for suffix in ("py", "go", "c"):
            (self.repo / f"probe.{suffix}").write_text("value = 1   \n")
        self.commit("test: add whitespace probe")
        self.assertEqual(self.git("diff", "--cached", "--name-only"), "")
        result = self.check()
        self.assertNotEqual(result.returncode, 0)
        for suffix in ("py", "go", "c"):
            self.assertIn(f"probe.{suffix}: line 1: trailing whitespace", result.stderr)

    def test_license_is_required(self):
        (self.repo / "LICENSE").unlink()
        self.assertIn("LICENSE: required file", self.check().stderr)

    def test_markdown_hard_breaks_are_allowed(self):
        (self.repo / "probe.md").write_text("Intentional hard break  \nNext line\n")
        self.git("add", ".")
        result = self.check()
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_actions_workflow_reintroduction_is_rejected(self):
        path = self.repo / '.github/workflows/probe.yml'
        path.parent.mkdir(exist_ok=True)
        path.write_text('name: probe\njobs: {}\n')
        self.assertIn('GitHub Actions are disabled by repository policy', self.check().stderr)

    def test_required_hosted_status_reintroduction_is_rejected(self):
        path = self.repo / '.github/main-ruleset.json'
        data = json.loads(path.read_text())
        data['rules'].append({'type': 'required_status_checks'})
        path.write_text(json.dumps(data) + '\n')
        self.assertIn('local review policy forbids required hosted status checks', self.check().stderr)

    def test_signed_history_protection_cannot_be_removed(self):
        path = self.repo / '.github/main-ruleset.json'
        data = json.loads(path.read_text())
        data['rules'] = [rule for rule in data['rules'] if rule['type'] != 'required_signatures']
        path.write_text(json.dumps(data) + '\n')
        self.assertIn('main protection missing: required_signatures', self.check().stderr)

    def test_duplicate_yaml_security_keys_are_rejected(self):
        (self.repo / 'probe.yml').write_text('permissions: read\npermissions: write\n')
        self.assertIn('duplicate YAML key: permissions', self.check().stderr)

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

    def test_fixture_commits_launch_no_automatic_maintenance(self):
        # An inherited aggressive policy must not create a detached child that
        # writes into the disposable repository while TemporaryDirectory cleans it.
        global_config = Path(self.temp.name) / 'global-config'
        global_config.write_text('[maintenance]\n\tauto = true\n[gc]\n\tauto = 1\n')
        trace = Path(self.temp.name) / 'git-trace.json'
        with mock.patch.dict(os.environ, {
            'GIT_CONFIG_GLOBAL': str(global_config),
            'GIT_TRACE2_EVENT': str(trace),
        }):
            self.commit('test: create maintenance fixture', signoff=True)
            (self.repo / 'probe.md').write_text('Probe\n')
            self.commit('test: update maintenance fixture', signoff=True)
        events = [json.loads(line) for line in trace.read_text().splitlines()]
        self.assertTrue(any(event.get('event') == 'start' for event in events))
        automatic = [event for event in events if event.get('event') == 'child_start'
                     and any(value in ('maintenance', 'gc') for value in event.get('argv', []))]
        self.assertEqual(automatic, [], 'fixture commit launched automatic maintenance')

    def test_commit_hook_rejects_missing_or_wrong_signoff(self):
        message = self.repo / "message"
        for body in ("docs: add probe\n", "docs: add probe\n\nSigned-off-by: Other <other@example.invalid>\n"):
            message.write_text(body)
            self.assertNotEqual(self.command("sh", ".githooks/commit-msg", str(message)).returncode, 0)
        message.write_text("docs: add probe\n\nSigned-off-by: Test Author <author@example.invalid>\n")
        self.assertEqual(self.command("sh", ".githooks/commit-msg", str(message)).returncode, 0)

    def go_archive_fixture(self, bad_checksum=False):
        data = io.BytesIO()
        with tarfile.open(fileobj=data, mode="w:gz") as archive:
            root = tarfile.TarInfo("go")
            root.type = tarfile.DIRTYPE
            archive.addfile(root)
            binary = tarfile.TarInfo("go/bin/go")
            content = b"test SDK binary\n"
            binary.size, binary.mode = len(content), 0o755
            archive.addfile(binary, io.BytesIO(content))
        archive_path = self.repo / "fixture-go.tar.gz"
        archive_path.write_bytes(data.getvalue())
        machine = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64"}[platform.machine()]
        target = f"{platform.system().lower()}_{machine}"
        pin = {"version": "fixture", "artifacts": {target: {
            "url": archive_path.as_uri(), "size": len(data.getvalue()),
            "sha256": "0" * 64 if bad_checksum else hashlib.sha256(data.getvalue()).hexdigest(),
        }}}
        (self.repo / "tools/go-pins.json").write_text(json.dumps(pin))
        previous = self.repo / ".tools/go"
        (previous / "bin").mkdir(parents=True)
        (previous / "bin/go").write_text("old SDK\n")
        (previous / "obsolete-source").write_text("must not survive replacement\n")
        return previous

    def test_go_bootstrap_accepts_root_directory_and_replaces_cleanly(self):
        previous = self.go_archive_fixture()
        result = self.command(sys.executable, "tools/bootstrap-go.py")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((previous / "bin/go").read_bytes(), b"test SDK binary\n")
        self.assertFalse((previous / "obsolete-source").exists())
        self.assertEqual(self.command(sys.executable, "tools/bootstrap-go.py").returncode, 0)

    def test_go_bootstrap_rejects_bad_checksum_before_replacing_sdk(self):
        previous = self.go_archive_fixture(bad_checksum=True)
        result = self.command(sys.executable, "tools/bootstrap-go.py")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("size/checksum mismatch", result.stderr)
        self.assertEqual((previous / "bin/go").read_text(), "old SDK\n")
        self.assertTrue((previous / "obsolete-source").exists())


if __name__ == "__main__":
    unittest.main()
