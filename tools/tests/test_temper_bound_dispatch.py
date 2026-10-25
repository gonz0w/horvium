# SPDX-License-Identifier: BSD-3-Clause
"""Fixed affected checks for source-bound validation tooling, never product Go."""
import contextlib
import importlib.util
import json
import platform
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

if platform.system() not in {'FreeBSD', 'OpenBSD'}:
    raise RuntimeError('This LAB contract requires an authorized BSD')
ROOT = Path(__file__).resolve().parents[2]
MODULES = ('test_review_plan', 'test_review_gate', 'test_temper_review_scope',
           'test_temper_gate_dispatch', 'test_temper_scope_binding',
           'test_temper_bound_dispatch', 'test_temper_source')


class BoundDispatchTests(unittest.TestCase):
    def setUp(self):
        spec = importlib.util.spec_from_file_location('bound_gate', ROOT / 'tools/review-gate.py')
        self.gate = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.gate)
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.identity = dict(source_commit='head', base_commit='base', source_tree='tree',
                             title='test: reviewed scope', signed_commits=['head'])
        self.commands, self.snapshots = [], []

    def run_gate(self, full=False, bindings=None, environments=None, failure_at=None, regression=None,
                 final_identity=None):
        gate = self.gate
        binding = {'route': 'temper-validation-tooling-v1', 'sha256': 'scope',
                   'review_reference': 'https://github.com/gonz0w/horvium/issues/424#issuecomment-123'}

        def step(command, log, record, directory):
            self.assertFalse(record['qualified'])
            self.commands.append(command)
            log.write_text('DEEP_RESULT {}\n' if log.name == 'deep.log' else 'checks\n')
            if len(self.commands) == failure_at:
                raise ValueError('local checks or teardown failed')
            if log.name == 'regressions.log':
                log.write_text('REGRESSION_RESULT ' + json.dumps(regression or
                    {'modules': list(MODULES), 'tests_run': 1, 'failures': 0, 'errors': 0, 'skipped': 0,
                     'successful': True}) + '\n')
            return {'completed': True, 'group_cleaned': True, 'worker_returncode': 0}

        with contextlib.ExitStack() as stack:
            stack.enter_context(patch.object(gate, 'ROOT', self.root))
            stack.enter_context(patch.object(sys, 'argv', ['review-gate.py'] + (['--full'] if full else [])))
            stack.enter_context(patch.dict(gate.os.environ, {'PR_TITLE': self.identity['title'], 'REVIEW_SCOPE': '/fictional/external.json'}))
            # Production identity() returns a fresh dictionary on each call.
            # The gate enriches its first result into the mutable review record.
            identity = stack.enter_context(patch.object(gate, 'identity',
                side_effect=[self.identity.copy(), (final_identity or self.identity).copy()]))
            stack.enter_context(patch.object(gate.subprocess, 'run'))
            stack.enter_context(patch.object(gate.PLAN, 'changed_paths', return_value=['PROJECT.md', 'tools/review-gate.py']))
            stack.enter_context(patch.object(gate, 'scope_binding', side_effect=bindings or [binding, binding], create=True))
            stack.enter_context(patch.object(gate.PLAN, 'input_fingerprint', return_value=('inputs', {})))
            metadata = stack.enter_context(patch.object(gate, 'metadata_fingerprint', side_effect=environments or [('metadata', {})] * 2, create=True))
            go = stack.enter_context(patch.object(gate.PLAN, 'go_environment', return_value={}, side_effect=None if full else AssertionError('eager Go')))
            environment = stack.enter_context(patch.object(gate.PLAN, 'environment_fingerprint', return_value='go-environment', side_effect=None if full else AssertionError('Go/compiler fingerprint')))
            stack.enter_context(patch.object(gate.PLAN, 'package_graph', side_effect=AssertionError('Go graph')))
            reuse = stack.enter_context(patch.object(gate, 'previous_evidence', return_value=None))
            stack.enter_context(patch.object(gate.OWNED, 'write_record', side_effect=lambda p, r: self.snapshots.append(json.loads(json.dumps(r)))))
            stack.enter_context(patch.object(gate, 'run_step', side_effect=step))
            stack.enter_context(patch.object(gate.signal, 'signal'))
            gate.main()
            self.assertEqual(identity.call_count, 2)
            return go, environment, reuse, metadata

    def test_bound_route_runs_exact_fixed_commands_and_never_reuses_deep(self):
        go, environment, reuse, metadata = self.run_gate()
        self.assertEqual(self.commands, [['make', 'check', 'secrets'],
                         [sys.executable, str(self.root / 'tools/check-temper-source.py')],
                         [sys.executable, str(self.root / 'tools/temper-regressions.py')]])
        go.assert_not_called()
        environment.assert_not_called()
        reuse.assert_not_called()
        self.assertEqual(metadata.call_count, 2)
        self.assertTrue(self.snapshots[-1]['qualified'])
        self.assertEqual(self.snapshots[-1]['qualification_scope'], 'affected validation-tooling review only')
        self.assertFalse(self.snapshots[-1]['deep_evidence']['executed'])

    def test_bound_final_source_identity_drift_rejects_qualification(self):
        changed = dict(self.identity, source_tree='changed')
        with self.assertRaisesRegex(ValueError, 'source changed during review; rerun on new commit'):
            self.run_gate(final_identity=changed)
        self.assertEqual(len(self.commands), 3)
        self.assertFalse(self.snapshots[-1]['qualified'])

    def test_scope_or_metadata_drift_rejects_qualification(self):
        for kwargs, message in (({'bindings': [{'route': 'temper-validation-tooling-v1', 'sha256': 'scope'}, None]},
                                 'review scope changed during review'),
                                ({'environments': [('before', {}), ('after', {})]},
                                 'review inputs or toolchain changed during review')):
            with self.subTest(kwargs=kwargs):
                with self.assertRaisesRegex(ValueError, message):
                    self.run_gate(**kwargs)
                self.assertFalse(self.snapshots[-1]['qualified'])

    def test_each_required_command_failure_prevents_qualification(self):
        for index in (1, 2, 3):
            with self.subTest(index=index):
                self.commands = []
                with self.assertRaisesRegex(ValueError, 'checks or teardown failed'):
                    self.run_gate(failure_at=index)
                self.assertFalse(self.snapshots[-1]['qualified'])
                self.assertEqual(len(self.commands), index)

    def test_missing_skipped_wrong_inventory_or_failed_regressions_rejected(self):
        for change in ({'modules': []}, {'tests_run': 0}, {'skipped': 1}, {'successful': False},
                       {'failures': 1}, {'errors': 1}):
            result = dict(modules=list(MODULES), tests_run=1, skipped=0, successful=True, failures=0, errors=0)
            result.update(change)
            with self.subTest(change=change):
                with self.assertRaisesRegex(ValueError, 'complete affected regression result'):
                    self.run_gate(regression=result)
                self.assertFalse(self.snapshots[-1]['qualified'])

    def test_explicit_full_overrides_valid_binding(self):
        _go, _environment, _reuse, metadata = self.run_gate(full=True)
        metadata.assert_not_called()
        self.assertEqual(self.commands, [['make', 'check', 'lint', 'test', 'secrets'],
                         [sys.executable, str(self.root / 'tools/deep-check.py'), '--all']])
        self.assertEqual(self.snapshots[-1]['plan']['mode'], 'full')


class RegressionInventoryTests(unittest.TestCase):
    def runner(self):
        spec = importlib.util.spec_from_file_location('fixed_regressions', ROOT / 'tools/temper-regressions.py')
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        return module

    def test_runner_inventory_is_fixed_and_missing_modules_fail(self):
        module = self.runner()
        self.assertEqual(module.MODULES, MODULES)
        loader = unittest.TestLoader()
        with self.assertRaisesRegex(ValueError, 'missing affected regression'):
            module.run_modules(loader=loader, names=('fictional_missing_regression',))

    def test_runner_refuses_skipped_and_empty_suites(self):
        module = self.runner()

        class Skipped(unittest.TestCase):
            @unittest.skip('synthetic skip must not qualify')
            def test_skipped(self):
                pass

        for suite in (unittest.TestSuite(), unittest.defaultTestLoader.loadTestsFromTestCase(Skipped)):
            class Loader:
                errors = []
                def loadTestsFromName(self, name):
                    return suite
            with self.assertRaisesRegex(ValueError, 'missing affected regression|affected regressions failed'):
                module.run_modules(loader=Loader(), names=('fictional_fixture',))


if __name__ == '__main__':
    unittest.main()
