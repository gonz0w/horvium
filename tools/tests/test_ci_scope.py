# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
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


if __name__ == '__main__':
    unittest.main()
