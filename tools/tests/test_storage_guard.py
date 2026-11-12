# SPDX-License-Identifier: BSD-3-Clause
"""Small synthetic fixtures only; no native workloads or retained images."""
import importlib.util
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('storage_guard', Path(__file__).resolve().parents[1] / 'storage_guard.py')


class StorageGuardTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.guard = importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(cls.guard)

    def test_reject_negative_limits(self):
        with self.assertRaises(ValueError):
            self.guard.check_capacity(1, 100, -1, 10, 20)

    def test_budget_includes_peak(self):
        self.assertTrue(self.guard.check_capacity(15, 100, 6, 20, 10))

    def test_reserve_after_peak(self):
        self.assertTrue(self.guard.check_capacity(1, 25, 6, 100, 20))

    def test_exact_boundary(self):
        self.assertEqual(self.guard.check_capacity(15, 25, 5, 20, 20), [])

    def test_disk_artifact_rejected(self):
        self.assertTrue(self.guard.blob_problem('out/test.qcow2', 1, 1024, set()))

    def test_large_blob_rejected(self):
        self.assertTrue(self.guard.blob_problem('out/evidence.tar.gz', 1025, 1024, set()))

    def test_explicit_asset_exception(self):
        self.assertIsNone(self.guard.blob_problem('assets/icon.img', 1025, 1024, {'assets/icon.img'}))
        self.assertTrue(self.guard.blob_problem('other/icon.img', 1, 1024, {'assets/icon.img'}))

    def test_small_source_allowed(self):
        self.assertIsNone(self.guard.blob_problem('src/main.py', 1024, 1024, set()))

    def test_missing_output_fails(self):
        with tempfile.TemporaryDirectory() as td:
            with self.assertRaises(OSError):
                self.guard.allocated_bytes([Path(td) / 'missing'])

    def test_duplicate_roots_not_counted_twice(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            (root / 'data').write_bytes(b'x')
            self.assertEqual(self.guard.allocated_bytes([root]), self.guard.allocated_bytes([root, root]))

    def test_symlink_root_rejected(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            (root / 'link').symlink_to(root, target_is_directory=True)
            with self.assertRaises(ValueError):
                self.guard.allocated_bytes([root / 'link'])

    def test_staged_uses_index_not_worktree(self):
        import shutil
        import subprocess
        if not shutil.which('git'):
            self.skipTest('Git unavailable; optional index integration not tested')
        with tempfile.TemporaryDirectory() as td:
            def git(*args):
                subprocess.run(['git', '-C', td, *args], check=True, capture_output=True)
            git('init', '-q')
            (Path(td) / 'sub').mkdir()
            path = Path(td) / 'evidence.bin'
            path.write_bytes(b'12345')
            git('add', 'evidence.bin')
            path.write_bytes(b'x')
            self.assertEqual(len(self.guard.check_staged(td, 4, set())), 1)
            self.assertEqual(len(self.guard.check_staged(Path(td) / 'sub', 4, set())), 1)
            self.assertEqual(self.guard.check_staged(td, 5, set()), [])

    def test_cli_invalid_measurement_fails_closed(self):
        with tempfile.TemporaryDirectory() as td:
            self.assertEqual(self.guard.main(['preflight', '--output', td + '/absent',
                '--budget-bytes', '100', '--reserve-bytes', '20', '--peak-bytes', '1']), 2)

    def test_committed_tree_guard(self):
        import shutil
        import subprocess
        if not shutil.which('git'):
            self.skipTest('Git unavailable; optional tree integration not tested')
        with tempfile.TemporaryDirectory() as td:
            def git(*args):
                subprocess.run(['git', '-C', td, *args], check=True, capture_output=True)
            git('init', '-q')
            (Path(td) / 'sub').mkdir()
            (Path(td) / 'test.img').write_bytes(b'x')
            git('add', 'test.img')
            git('-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid',
                '-c', 'commit.gpgsign=false', 'commit', '-qm', 'fixture')
            self.assertEqual(len(self.guard.check_tree(td, 10, set())), 1)
            self.assertEqual(len(self.guard.check_tree(Path(td) / 'sub', 10, set())), 1)
            self.assertEqual(self.guard.check_tree(td, 10, {'test.img'}), [])
