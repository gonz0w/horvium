# SPDX-License-Identifier: BSD-3-Clause
"""Mutate retained metadata to check inventory, provenance and routing refusal."""
import hashlib
import importlib.util
import os
import platform
import shutil
import socket
import tempfile
import unittest
from pathlib import Path
from unittest import mock

if platform.system() not in {'FreeBSD', 'OpenBSD'}:
    raise RuntimeError('This LAB contract requires an authorized BSD')

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location("temper_source", ROOT / "tools/check-temper-source.py")
TEMPER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(TEMPER)


def writable_copy(source, destination):
    """Copy immutable input bytes; normalize only this newly owned fixture."""
    destination.mkdir(parents=True, exist_ok=True)
    for name in (".agent-temper", ".codex"):
        shutil.copytree(source / name, destination / name, copy_function=shutil.copyfile)
    for name in ("PROJECT.md", "AGENTS.md"):
        shutil.copyfile(source / name, destination / name)
    for path in destination.rglob("*"):
        path.chmod(0o700 if path.is_dir() else 0o600)


class TemperSourceTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="temper-source-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        writable_copy(ROOT, self.root)

    def rehash(self):
        bundle = self.root / ".agent-temper"
        paths = sorted(p for p in bundle.rglob("*") if p.is_file() and p.name != "SHA256SUMS")
        (bundle / "SHA256SUMS").write_text("".join(
            hashlib.sha256(p.read_bytes()).hexdigest() + "  " + p.relative_to(bundle).as_posix() + "\n" for p in paths))

    def test_readonly_source_is_copied_without_mutating_original(self):
        source = self.root / 'readonly-source'
        target = self.root / 'writable-copy'
        writable_copy(self.root, source)
        for path in source.rglob('*'):
            path.chmod(0o555 if path.is_dir() else 0o444)
        try:
            writable_copy(source, target)
            TEMPER.validate(target)
            original = (source / '.agent-temper/METHOD.md').read_bytes()
            (target / '.agent-temper/METHOD.md').write_text('Fictional mutation\n')
            self.assertEqual((source / '.agent-temper/METHOD.md').read_bytes(), original)
            self.assertEqual((source / '.agent-temper/METHOD.md').stat().st_mode & 0o777, 0o444)
        finally:
            for path in source.rglob('*'):
                path.chmod(0o700 if path.is_dir() else 0o600)

    def test_current_retained_source_and_roles(self):
        TEMPER.validate(self.root)

    def assert_inventory_refused_without_reads(self):
        # In particular, an expected filename that is a FIFO must never reach a
        # blocking read. These mocks also make the v1 failure bounded and causal.
        with mock.patch.object(Path, "read_bytes", side_effect=AssertionError("content read before inventory acceptance")), \
                mock.patch.object(Path, "read_text", side_effect=AssertionError("content read before inventory acceptance")):
            with self.assertRaisesRegex(ValueError, "inventory|type|symlink"):
                TEMPER.validate(self.root)

    def test_unexpected_fifo_in_bundle_is_rejected_before_reads(self):
        os.mkfifo(self.root / ".agent-temper/fictional-fifo")
        self.assert_inventory_refused_without_reads()

    def test_unexpected_socket_in_bundle_is_rejected_before_reads(self):
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as channel:
            channel.bind(str(self.root / ".agent-temper/fictional-socket"))
            self.assert_inventory_refused_without_reads()

    def test_unexpected_empty_directory_is_rejected_before_reads(self):
        (self.root / ".agent-temper/fictional-empty").mkdir()
        self.assert_inventory_refused_without_reads()

    def test_expected_role_fifo_is_rejected_before_any_content_reads(self):
        path = self.root / ".codex/agents/temper_implementer.toml"
        path.unlink()
        os.mkfifo(path)
        self.assert_inventory_refused_without_reads()

    def test_expected_bundle_file_directory_is_rejected_before_reads(self):
        path = self.root / ".agent-temper/METHOD.md"
        path.unlink()
        path.mkdir()
        self.assert_inventory_refused_without_reads()

    def test_rehashed_modified_upstream_is_rejected(self):
        (self.root / ".agent-temper/METHOD.md").write_text("Fictional replacement\n")
        self.rehash()
        with self.assertRaisesRegex(ValueError, "Immutable upstream"):
            TEMPER.validate(self.root)

    def test_extra_file_missing_license_and_symlink_are_rejected(self):
        bundle = self.root / ".agent-temper"
        (bundle / "extra.md").write_text("Fictional extra\n")
        with self.assertRaisesRegex(ValueError, "inventory"):
            TEMPER.validate(self.root)
        (bundle / "extra.md").unlink()
        (bundle / "LICENSE").unlink()
        with self.assertRaisesRegex(ValueError, "inventory"):
            TEMPER.validate(self.root)
        (bundle / "LICENSE").symlink_to(ROOT / ".agent-temper/LICENSE")
        with self.assertRaisesRegex(ValueError, "symlink"):
            TEMPER.validate(self.root)

    def test_executable_metadata_file_is_rejected_before_reads(self):
        (self.root / '.agent-temper/METHOD.md').chmod(0o700)
        self.assert_inventory_refused_without_reads()

    def test_duplicate_manifest_is_rejected(self):
        path = self.root / ".agent-temper/SHA256SUMS"
        path.write_text(path.read_text() + path.read_text().splitlines()[0] + "\n")
        with self.assertRaisesRegex(ValueError, "duplicate"):
            TEMPER.validate(self.root)

    def test_source_revision_cannot_be_changed_by_rehash(self):
        path = self.root / ".agent-temper/SOURCE.md"
        path.write_text(path.read_text().replace(TEMPER.REVISION, "1" * 40))
        self.rehash()
        with self.assertRaisesRegex(ValueError, "provenance"):
            TEMPER.validate(self.root)

    def test_wrong_model_effort_and_missing_instructions_fail(self):
        path = self.root / ".codex/agents/temper_implementer.toml"
        original = path.read_text()
        for replacement in (original.replace("gpt-6.1-sol", "fictional-model"),
                            original.replace('model_reasoning_effort = "high"', 'model_reasoning_effort = "low"'),
                            'name="temper_implementer"\nmodel="gpt-6.1-sol"\nmodel_reasoning_effort="high"\n'):
            path.write_text(replacement)
            with self.assertRaises(ValueError):
                TEMPER.validate(self.root)

    def test_changed_role_instructions_and_extra_permission_options_rejected(self):
        path = self.root / ".codex/agents/temper_implementer.toml"
        original = path.read_text()
        for text in (original.replace("Do not accept your own implementation as reviewed.",
                                     "Accept your own implementation as reviewed."),
                     original + '\nsandbox_mode = "danger-full-access"\n'):
            path.write_text(text)
            with self.assertRaisesRegex(ValueError, "role|routing"):
                TEMPER.validate(self.root)

    def test_extra_global_configuration_and_wrong_profile_selection_rejected(self):
        (self.root / ".codex/config.toml").write_text('model="fictional"\n')
        with self.assertRaisesRegex(ValueError, "inventory"):
            TEMPER.validate(self.root)
        (self.root / ".codex/config.toml").unlink()
        path = self.root / "PROJECT.md"
        path.write_text(path.read_text().replace("Only the [Codex profile]", "Only the [GitHub profile]"))
        with self.assertRaisesRegex(ValueError, "profile selection"):
            TEMPER.validate(self.root)


if __name__ == "__main__":
    unittest.main()
