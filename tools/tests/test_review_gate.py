# SPDX-License-Identifier: BSD-3-Clause
"""Local review must bind a clean, signed contributor range and valid title."""
import importlib.util
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]


class ReviewGateTests(unittest.TestCase):
    def setUp(self):
        spec = importlib.util.spec_from_file_location('review', ROOT / 'tools/review-gate.py')
        self.tool = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.tool)
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.repo = Path(self.temp.name)
        self.git('init', '-b', 'main')
        self.git('config', 'user.name', 'Fixture')
        self.git('config', 'user.email', 'fixture@example.invalid')
        self.git('config', 'commit.gpgsign', 'false')
        self.git('config', 'core.hooksPath', '/dev/null')
        (self.repo / 'probe').write_text('base\n')
        self.git('add', '.')
        self.git('commit', '-s', '-m', 'test: create fixture')
        self.base = self.git('rev-parse', 'HEAD')

    def git(self, *args):
        return subprocess.check_output(['git', *args], cwd=self.repo, text=True, stderr=subprocess.DEVNULL).strip()

    def test_invalid_title_cannot_pass(self):
        for title in ('Update everything', 'ci: x', 'ci: checks\nforged', 'Merge branch'):
            with self.assertRaises(ValueError):
                self.tool.validate_title(title)
        self.tool.validate_title('ci: move qualification to local hosts')

    def test_dirty_source_cannot_be_reported_as_committed_evidence(self):
        (self.repo / 'probe').write_text('dirty\n')
        with self.assertRaisesRegex(ValueError, 'clean worktree'):
            self.tool.identity(self.repo, self.base, 'test: review fixture')

    def test_unsigned_contributor_is_rejected(self):
        (self.repo / 'probe').write_text('head\n')
        self.git('add', '.')
        self.git('commit', '-s', '-m', 'test: modify fixture')
        with self.assertRaisesRegex(ValueError, 'signature'):
            self.tool.identity(self.repo, self.base, 'test: review fixture')

    def test_empty_range_is_not_a_contributor_review(self):
        with self.assertRaisesRegex(ValueError, 'contributor commits'):
            self.tool.identity(self.repo, self.base, 'test: review fixture')
