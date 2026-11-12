# SPDX-License-Identifier: BSD-3-Clause
"""Portable metadata/limit checks; no product build, VM or retained output."""
import importlib.util
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

TOOLS = Path(__file__).resolve().parents[1]


class StoragePreflightTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        spec = importlib.util.spec_from_file_location('preflight', TOOLS / 'storage-preflight.py')
        cls.tool = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(cls.tool)

    def test_large_host_reserve(self):
        self.assertEqual(self.tool.reserve_bytes(2_000_000_000_000), 200_000_000_000)

    def test_small_guest_reserve(self):
        self.assertEqual(self.tool.reserve_bytes(40_000_000_000), 4_000_000_000)

    def test_peak_exceeds_cache_budget(self):
        with tempfile.TemporaryDirectory() as td:
            with patch.object(self.tool.guard, 'allocated_bytes', return_value=19_000_000_000), patch.object(self.tool.shutil, 'disk_usage', return_value=SimpleNamespace(total=2_000_000_000_000, free=900_000_000_000)):
                self.assertEqual(self.tool.preflight(Path(td)), 1)

    def test_preserves_existing_file(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            (root / '.tools').mkdir()
            marker = root / '.tools' / 'source'
            marker.write_text('keep')
            with patch.object(self.tool.shutil, 'disk_usage', return_value=SimpleNamespace(total=2_000_000_000_000, free=1)):
                self.assertEqual(self.tool.preflight(root), 1)
            self.assertEqual(marker.read_text(), 'keep')

    def test_sufficient_space(self):
        with tempfile.TemporaryDirectory() as td:
            with patch.object(self.tool.shutil, 'disk_usage', return_value=SimpleNamespace(total=40_000_000_000, free=10_000_000_000)):
                self.assertEqual(self.tool.preflight(Path(td)), 0)

    def test_direct_bootstrap_and_source_check_are_wired(self):
        self.assertIn('storage-preflight.py', (TOOLS / 'bootstrap-tools.py').read_text())
        self.assertIn('storage_guard.py', (TOOLS / 'check-repo.py').read_text())

    def test_make_default_is_preserved(self):
        lines = (TOOLS.parent / 'Makefile').read_text().splitlines()
        targets = [line.split(':', 1)[0] for line in lines if ':' in line and not line.startswith(('.', '\t', '#', ' ')) and ':=' not in line]
        self.assertEqual(targets[0], 'tools')
