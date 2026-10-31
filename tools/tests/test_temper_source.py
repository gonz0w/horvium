# SPDX-License-Identifier: BSD-3-Clause
"""Mutate retained metadata to check inventory, provenance and routing refusal."""
import hashlib
import importlib.util
import os
import sys
import platform
import shutil
import socket
import tempfile
import unittest
from pathlib import Path
from unittest import mock


if os.name != 'posix' or os.getuid() == 0 or sys.version_info[:3] != (3, 14, 7):
    raise RuntimeError('Portable source regressions require permitted unprivileged POSIX and Python 3.14.7')

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location("temper_source", ROOT / "tools/check-temper-source.py")
TEMPER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(TEMPER)


def writable_copy(source, destination):
    """Copy immutable input bytes; normalize only this newly owned fixture."""
    destination.mkdir(parents=True, exist_ok=True)
    for name in (".agent-temper", ".codex"):
        shutil.copytree(source / name, destination / name, copy_function=shutil.copyfile)
    for name in ("PROJECT.md", "AGENTS.md", "policy.toml"):
        shutil.copyfile(source / name, destination / name)
    for module in TEMPER.selection((source / 'policy.toml').read_bytes()):
        name = module['path']
        if not (destination / name).exists():
            (destination / name).parent.mkdir(parents=True, exist_ok=True)
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
        with self.assertRaisesRegex(ValueError, "immutable upstream"):
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
        with self.assertRaisesRegex(ValueError, "admission"):
            TEMPER.validate(self.root)

    def test_source_revision_cannot_be_changed_by_rehash(self):
        path = self.root / ".agent-temper/SOURCE.md"
        path.write_text(path.read_text().replace('cf928f4120b687f34f1b1f974603a4979b5b874c', "1" * 40))
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

    def test_rehashed_license_and_missing_or_extra_roles_cannot_readmit(self):
        license_path = self.root / '.agent-temper/LICENSE'
        license_path.write_text('Fictional incomplete license\n')
        self.rehash()
        with self.assertRaisesRegex(ValueError, 'admission'):
            TEMPER.validate(self.root)
        writable_copy(ROOT, self.root / 'fresh')
        fresh = self.root / 'fresh'
        role = fresh / '.codex/agents/temper_product.toml'
        raw = role.read_bytes()
        role.unlink()
        with self.assertRaisesRegex(ValueError, 'admission'):
            TEMPER.validate(fresh)
        role.write_bytes(raw)
        (fresh / '.codex/agents/fictional.toml').write_bytes(raw)
        with self.assertRaisesRegex(ValueError, 'admission'):
            TEMPER.validate(fresh)

    def test_policy_prose_changes_need_reviewed_readmission(self):
        for name in ('policy.toml', 'docs/policy/routing.md', 'AGENTS.md'):
            path = self.root / name
            original = path.read_bytes()
            path.write_bytes(original + b'\n# Fictional tiny prose change\n')
            with self.assertRaisesRegex(ValueError, 'admission'):
                TEMPER.validate(self.root)
            path.write_bytes(original)

    def test_candidate_digest_environment_or_policy_override_cannot_admit(self):
        path = self.root / '.codex/agents/temper_product.toml'
        path.write_text(path.read_text().replace('gpt-6.1-terra', 'fictional-model'))
        digest = TEMPER.admission_input(self.root)[0]
        with mock.patch.dict(os.environ, {'REVIEWED_ADMISSION_SHA256': digest,
                                         'TEMPER_ADMISSION_SHA256': digest}):
            with self.assertRaisesRegex(ValueError, 'admission'):
                TEMPER.validate(self.root)
        policy = self.root / 'policy.toml'
        policy.write_text('reviewed_admission_sha256 = "' + digest + '"\n' + policy.read_text())
        with self.assertRaisesRegex(ValueError, 'policy declaration'):
            TEMPER.validate(self.root)

    def test_selected_module_identity_prerequisites_and_traversal_fail(self):
        path = self.root / 'policy.toml'
        original = path.read_text()
        for text in (original.replace('id = "work"', 'id = "project"'),
                     original.replace('requires = []', 'requires = ["missing"]'),
                     original.replace('requires = []', 'requires = ["temper-core"]'),
                     original.replace('docs/policy/work.md', '../outside.md'),
                     original.replace('docs/policy/work.md', '/tmp/outside.md'),
                     original.replace('docs/policy/work.md', 'docs//policy/work.md'),
                     original.replace('docs/policy/work.md', 'docs/./policy/work.md')):
            with self.subTest(declaration=text):
                path.write_text(text)
                with self.assertRaises(ValueError):
                    TEMPER.validate(self.root)
        path.write_text(original.replace('id = "work"', 'id = "fictional-work"').replace('"work",', '"fictional-work",'))
        with self.assertRaisesRegex(ValueError, 'admission'):
            TEMPER.validate(self.root)

    def test_selected_module_and_ancestor_links_and_fifo_refused_before_read(self):
        path = self.root / 'docs/policy/routing.md'
        raw = path.read_bytes()
        path.unlink()
        os.mkfifo(path)
        original = Path.read_bytes
        def read_bytes(candidate):
            if candidate == path:
                raise AssertionError('selected FIFO content read')
            return original(candidate)
        with mock.patch.object(Path, 'read_bytes', read_bytes):
            with self.assertRaisesRegex(ValueError, 'type'):
                TEMPER.validate(self.root)
        path.unlink()
        path.symlink_to(ROOT / 'docs/policy/routing.md')
        with self.assertRaisesRegex(ValueError, 'symlink'):
            TEMPER.validate(self.root)
        path.unlink()
        path.write_bytes(raw)
        directory = self.root / 'docs/policy'
        directory.rename(self.root / 'saved-policy')
        directory.symlink_to(self.root / 'saved-policy', target_is_directory=True)
        with self.assertRaisesRegex(ValueError, 'symlink'):
            TEMPER.validate(self.root)

    def test_root_policy_fifo_is_refused_before_any_content_read(self):
        path = self.root / 'policy.toml'
        path.unlink()
        os.mkfifo(path)
        self.assert_inventory_refused_without_reads()


if __name__ == "__main__":
    unittest.main()
