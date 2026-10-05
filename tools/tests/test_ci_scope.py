# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location('ci_scope', ROOT / 'tools/ci-scope.py')


class CIScopeTests(unittest.TestCase):
    def setUp(self):
        self.module = importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(self.module)

    def test_only_plain_documentation_avoids_product_checks(self):
        self.assertFalse(self.module.needs_go(['README.md', 'docs/testing/M2/properties.md']))

    def test_unknown_empty_and_behavior_changes_fail_closed(self):
        for paths in ([], ['internal/storage/model.go'], ['go.sum'], ['Makefile'],
                      ['tools/ci-scope.py'], ['.github/workflows/repository.yml'],
                      ['docs/ui/console.html'], ['docs/../internal/code.md'],
                      ['docs/notes.md\ninternal/code.go'], ['internal/design.md']):
            with self.subTest(paths=paths):
                self.assertTrue(self.module.needs_go(paths))

    def test_mixed_change_cannot_hide_deleted_or_renamed_code(self):
        self.assertTrue(self.module.needs_go(['docs/change.md', 'cmd/horvium/main.go']))


    def test_git_rename_to_documentation_cannot_hide_deleted_code(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            def git(*args):
                return subprocess.check_output(['git', *args], cwd=root, text=True).strip()
            git('init', '-b', 'main')
            git('config', 'user.name', 'Fixture')
            git('config', 'user.email', 'fixture@example.invalid')
            git('config', 'commit.gpgsign', 'false')
            git('config', 'core.hooksPath', '/dev/null')
            (root / 'internal').mkdir()
            (root / 'internal/code.go').write_text('package example\n')
            git('add', '.')
            git('commit', '-m', 'test: initial source')
            base = git('rev-parse', 'HEAD')
            (root / 'docs').mkdir()
            git('mv', 'internal/code.go', 'docs/code.md')
            git('commit', '-m', 'docs: rename source')
            head = git('rev-parse', 'HEAD')
            output = root / 'outputs'
            result = subprocess.run([sys.executable, str(ROOT / 'tools/ci-scope.py')],
                                    cwd=root, capture_output=True, text=True,
                                    env={**os.environ, 'BASE_SHA': base, 'HEAD_SHA': head,
                                         'GITHUB_OUTPUT': str(output)})
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(output.read_text(), 'go=true\n')


if __name__ == '__main__':
    unittest.main()
