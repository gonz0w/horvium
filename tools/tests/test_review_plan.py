# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
from pathlib import Path
import tempfile
import unittest
ROOT = Path(__file__).resolve().parents[2]

class ReviewPlanTests(unittest.TestCase):

    def module(self):
        spec = importlib.util.spec_from_file_location('plan', ROOT / 'tools/review-plan.py')
        m = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(m)
        return m

    def graph(self):
        return {'./internal/leaf': {'directory': 'internal/leaf', 'imports': []}, './internal/consumer': {'directory': 'internal/consumer', 'imports': ['./internal/leaf']}, './cmd/horvium': {'directory': 'cmd/horvium', 'imports': ['./internal/consumer']}, './internal/other': {'directory': 'internal/other', 'imports': []}}

    def test_changed_package_and_transitive_consumers_are_selected(self):
        m = self.module()
        for name in ['internal/leaf/change.go', 'internal/leaf/old_freebsd.go', 'internal/leaf/testdata/control.json', 'internal/leaf/assets/template.html']:
            p = m.select_plan([name], self.graph())
            self.assertEqual(p['mode'], 'focused')
            self.assertEqual(p['packages'], ['./cmd/horvium', './internal/consumer', './internal/leaf'])

    def test_broad_unknown_or_unmapped_changes_escalate(self):
        m = self.module()
        for name in ['go.mod', 'go.sum', 'Makefile', 'tools/review-gate.py', '.githooks/pre-commit', 'AGENTS.md', 'docs/engineering/local-review.md', 'mystery.bin', 'internal/missing/new.go']:
            self.assertEqual(m.select_plan([name], self.graph())['mode'], 'full', name)
        self.assertEqual(m.select_plan(['docs/testing/status.md'], self.graph(), True)['mode'], 'full')

    def test_documentation_keeps_baseline_without_deep_tests(self):
        m = self.module()
        p = m.select_plan(['README.md', 'docs/testing/status.md', 'docs/testing/probe-manifest.json'], {})
        self.assertEqual(p['mode'], 'documentation')
        self.assertEqual(p['packages'], [])
        self.assertEqual(m.select_plan(['docs/example.json'], {})['mode'], 'full')

    def test_native_platform_edges_are_merged(self):
        m = self.module()
        host = [{'ImportPath': 'example/internal/a', 'Dir': '/fixture/internal/a', 'Imports': [], 'TestImports': []}]
        native = [{'ImportPath': 'example/internal/a', 'Dir': '/fixture/internal/a', 'Imports': [], 'TestImports': ['example/internal/b']}, {'ImportPath': 'example/internal/b', 'Dir': '/fixture/internal/b', 'Imports': []}]
        graph = m.merge_graphs(Path('/fixture'), [host, native])
        self.assertEqual(m.select_plan(['internal/b/only_freebsd.go'], graph)['packages'], ['./internal/a', './internal/b'])

    def test_reuse_requires_exact_inputs_success_and_clean_teardown(self):
        m = self.module()
        with tempfile.TemporaryDirectory() as tmp:
            log = Path(tmp) / 'deep.log'
            log.write_text('DEEP_RESULT {}\n')
            expected = {'mode': 'focused', 'packages': ['./internal/leaf'], 'inputs_digest': 'a', 'environment_digest': 'b'}
            report = {'schema': 2, 'qualified': True, 'base_commit': 'base', 'source_commit': 'old', 'deep_key': expected, 'deep_evidence': {'executed': True, 'log_sha256': m.digest(log), 'supervisor': {'completed': True, 'group_cleaned': True, 'worker_returncode': 0}}}
            self.assertTrue(m.reusable(report, expected, 'base', log))
            for key, value in [('qualified', False), ('base_commit', 'other'), ('schema', 1)]:
                bad = dict(report)
                bad[key] = value
                self.assertFalse(m.reusable(bad, expected, 'base', log))
            for key in expected:
                different = dict(expected)
                different[key] = 'changed'
                self.assertFalse(m.reusable(report, different, 'base', log))
            bad = dict(report)
            bad['deep_evidence'] = {'executed': False}
            self.assertFalse(m.reusable(bad, expected, 'base', log))
            report['deep_evidence']['supervisor']['group_cleaned'] = False
            self.assertFalse(m.reusable(report, expected, 'base', log))
            report['deep_evidence']['supervisor']['group_cleaned'] = True
            log.write_text('tampered')
            self.assertFalse(m.reusable(report, expected, 'base', log))

    def test_behavior_fingerprint_ignores_only_documentation(self):
        m = self.module()
        self.assertTrue(m.documentation_path('docs/testing/state.md'))
        for name in ['AGENTS.md', 'CONTRIBUTING.md', 'docs/engineering/local-review.md', 'docs/testing/strategy.md', 'internal/a/probe.go', 'tools/review-plan.py']:
            self.assertFalse(m.documentation_path(name), name)
