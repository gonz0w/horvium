# SPDX-License-Identifier: BSD-3-Clause
import os
import sys
import importlib.util
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
if os.name != 'posix' or os.getuid() == 0 or sys.version_info[:3] != (3, 14, 7):
    raise RuntimeError('Portable source regressions require permitted unprivileged POSIX and Python 3.14.7')

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

    def test_environment_ignores_derived_scratch_but_tracks_build_flags(self):
        m = self.module()
        def fingerprint(scratch, experiment=''):
            import json
            settings = {'CC': 'cc', 'GOGCCFLAGS': '-fdebug-prefix-map=' + scratch,
                        'GOARM64': 'v8.0', 'GOEXPERIMENT': experiment}
            with patch.object(m.subprocess, 'check_output', side_effect=[json.dumps(settings), b'compiler version']):
                with patch.object(m, 'digest', return_value='compiler-sha'):
                    return m.environment_fingerprint(ROOT)
        self.assertEqual(fingerprint('/tmp/build-1'), fingerprint('/tmp/build-2'))
        self.assertNotEqual(fingerprint('/tmp/build-1'), fingerprint('/tmp/build-1', 'experiment'))

    def test_policy_is_full_alone_or_mixed_and_always_in_fingerprint(self):
        m = self.module()
        for name in ('policy.toml', 'docs/policy/routing.md', 'docs/policy/unknown.md'):
            self.assertFalse(m.documentation_path(name))
            self.assertEqual(m.select_plan([name], {})['mode'], 'full')
            self.assertEqual(m.select_plan([name, '.codex/agents/temper_product.toml'], {})['mode'], 'full')
            raw = b'100644 blob ' + b'a' * 40 + b'\t' + name.encode() + b'\0'
            with patch.object(m.subprocess, 'check_output', return_value=raw):
                self.assertIn(name, m.input_fingerprint(ROOT, 'fixture')[1])

    def test_unadmitted_candidate_inventory_cannot_expand_temper_scope(self):
        from test_temper_source import writable_copy
        m = self.module()
        with tempfile.TemporaryDirectory() as tmp:
            candidate = Path(tmp)
            writable_copy(ROOT, candidate)
            role = candidate / '.codex/agents/fictional.toml'
            role.write_bytes((candidate / '.codex/agents/temper_product.toml').read_bytes())
            self.assertEqual(m.select_plan(['.codex/agents/fictional.toml'], {}, root=candidate)['mode'], 'full')
            self.assertEqual(m.select_plan(['.codex/agents/temper_product.toml'], {}, root=candidate)['mode'], 'full')
            self.assertEqual(m.accepted_temper_paths(candidate), frozenset())
