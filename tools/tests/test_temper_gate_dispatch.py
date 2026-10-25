# SPDX-License-Identifier: BSD-3-Clause
"""Gate dispatch contract; source preparation is not an execution result.

These main-level fixtures exercise command selection and receipt failures with
mocked execution. They do not admit tools, run hooks or qualify the real process
supervisor. The metadata baseline is a minimum: separate exact bundle/role
verification remains required before a reduced route can be accepted.
"""
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
    raise RuntimeError('This pending LAB contract requires an authorized BSD')

ROOT = Path(__file__).resolve().parents[2]
ROLE = '.codex/agents/temper_workflow.toml'


class TemperGateDispatchTests(unittest.TestCase):
    def setUp(self):
        spec = importlib.util.spec_from_file_location('gate_under_test', ROOT / 'tools/review-gate.py')
        self.gate = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.gate)
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.record = {'source_commit': 'head', 'base_commit': 'base',
                       'source_tree': 'tree', 'title': 'test: review fixture',
                       'signed_commits': ['head']}
        self.commands = []
        self.snapshots = []

    def run_gate(self, paths=None, full=False, failure=None, final_identity=None,
                 fingerprints=None, forbid_go=False, dco_failure=None):
        gate = self.gate

        def write_record(_path, record):
            self.snapshots.append(json.loads(json.dumps(record)))

        def run_step(command, log, record, directory):
            self.assertFalse(record['qualified'])
            self.commands.append(command)
            log.write_text('DEEP_RESULT {}\n' if log.name == 'deep.log' else 'checks\n')
            if failure:
                raise failure
            return {'completed': True, 'group_cleaned': True, 'worker_returncode': 0}

        with contextlib.ExitStack() as stack:
            stack.enter_context(patch.object(gate, 'ROOT', self.root))
            stack.enter_context(patch.object(sys, 'argv', ['review-gate.py'] + (['--full'] if full else [])))
            stack.enter_context(patch.dict(gate.os.environ, {'PR_TITLE': self.record['title']}))
            identity = stack.enter_context(patch.object(gate, 'identity', side_effect=[self.record.copy(), final_identity or self.record.copy()]))
            dco = stack.enter_context(patch.object(gate.subprocess, 'run', side_effect=dco_failure))
            stack.enter_context(patch.object(gate.PLAN, 'changed_paths', return_value=paths or [ROLE]))
            graph = stack.enter_context(patch.object(gate.PLAN, 'package_graph', side_effect=AssertionError('unexpected Go package query')))
            stack.enter_context(patch.object(gate.PLAN, 'input_fingerprint', side_effect=fingerprints or [('inputs', {})] * 2))
            stack.enter_context(patch.object(gate, 'metadata_fingerprint', return_value=('metadata', {}), create=True))
            go = stack.enter_context(patch.object(gate.PLAN, 'go_environment', return_value={}, side_effect=AssertionError('unexpected Go environment') if forbid_go else None))
            environment = stack.enter_context(patch.object(gate.PLAN, 'environment_fingerprint', return_value='environment', side_effect=AssertionError('unexpected Go/compiler fingerprint') if forbid_go else None))
            reuse = stack.enter_context(patch.object(gate, 'previous_evidence', return_value=None))
            stack.enter_context(patch.object(gate.OWNED, 'write_record', side_effect=write_record))
            stack.enter_context(patch.object(gate, 'run_step', side_effect=run_step))
            stack.enter_context(patch.object(gate.signal, 'signal'))
            gate.main()
            dco.assert_called_once_with([sys.executable, str(self.root / 'tools/check-dco.py'), 'base', 'head'], cwd=self.root, check=True)
            self.assertEqual(identity.call_count, 2)
            graph.assert_not_called()
            return go, environment, reuse

    def test_temper_dispatches_metadata_baseline_without_deep_or_reuse(self):
        _go, _environment, reuse = self.run_gate()
        self.assertEqual(self.commands, [['make', 'check', 'secrets'],
                                       [sys.executable, str(self.root / 'tools/check-temper-source.py')]])
        reuse.assert_not_called()
        self.assertTrue(self.snapshots[-1]['qualified'])
        self.assertFalse(self.snapshots[-1]['deep_evidence']['executed'])

    def test_temper_never_queries_go_or_compiler_metadata(self):
        go, environment, _reuse = self.run_gate(forbid_go=True)
        go.assert_not_called()
        environment.assert_not_called()

    def test_temper_command_failure_stays_unqualified(self):
        with self.assertRaisesRegex(ValueError, 'local checks or teardown failed'):
            self.run_gate(failure=ValueError('local checks or teardown failed; inspect checks.log'))
        self.assertFalse(self.snapshots[-1]['qualified'])
        self.assertIn('failure', self.snapshots[-1])

    def test_dco_failure_stops_before_dispatch_or_qualification(self):
        with self.assertRaisesRegex(ValueError, 'DCO rejected'):
            self.run_gate(dco_failure=ValueError('DCO rejected'))
        self.assertEqual(self.commands, [])
        self.assertFalse(any(record.get('qualified') for record in self.snapshots))

    def test_temper_final_source_change_stays_unqualified(self):
        changed = dict(self.record, source_tree='changed')
        with self.assertRaisesRegex(ValueError, 'source changed during review'):
            self.run_gate(final_identity=changed)
        self.assertFalse(self.snapshots[-1]['qualified'])

    def test_temper_final_input_change_stays_unqualified(self):
        with self.assertRaisesRegex(ValueError, 'review inputs or toolchain changed'):
            self.run_gate(fingerprints=[('before', {}), ('after', {})])
        self.assertFalse(self.snapshots[-1]['qualified'])

    def test_explicit_full_keeps_product_baseline_and_complete_deep(self):
        self.run_gate(full=True)
        self.assertEqual(self.commands, [['make', 'check', 'lint', 'test', 'secrets'],
                                       [sys.executable, str(self.root / 'tools/deep-check.py'), '--all']])
        self.assertEqual(self.snapshots[-1]['plan']['mode'], 'full')

    def test_mixed_unknown_policy_and_tooling_keep_full_dispatch(self):
        for path in ('unknown.file', 'PROJECT.md', 'tools/review-plan.py',
                     'tools/review-gate.py', 'tools/tests/test_temper_gate_dispatch.py',
                     '.githooks/pre-commit', 'Makefile', 'go.mod'):
            with self.subTest(path=path):
                self.commands = []
                self.run_gate(paths=[ROLE, path])
                self.assertEqual(self.commands, [['make', 'check', 'lint', 'test', 'secrets'],
                                               [sys.executable, str(self.root / 'tools/deep-check.py'), '--all']])


class TemperGateSupervisorFailureTests(unittest.TestCase):
    def test_divergent_review_base_is_rejected_before_dispatch(self):
        spec = importlib.util.spec_from_file_location('gate_identity_under_test', ROOT / 'tools/review-gate.py')
        gate = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(gate)
        with patch.object(gate, 'git', side_effect=['', 'head', 'base']):
            with patch.object(gate.subprocess, 'run') as command:
                command.return_value.returncode = 1
                with self.assertRaisesRegex(ValueError, 'review base must be an ancestor'):
                    gate.identity(ROOT, 'base', 'test: review fixture')
                command.assert_called_once_with(['git', 'merge-base', '--is-ancestor', 'base', 'head'], cwd=ROOT)

    def test_run_step_rejects_worker_failure_and_incomplete_cleanup(self):
        spec = importlib.util.spec_from_file_location('gate_supervisor_under_test', ROOT / 'tools/review-gate.py')
        gate = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(gate)
        for returncode, cleaned in ((1, True), (0, False)):
            with self.subTest(returncode=returncode, group_cleaned=cleaned):
                with tempfile.TemporaryDirectory() as tmp:
                    directory = Path(tmp)
                    with patch.object(gate.OWNED, 'OwnedProcess') as owned:
                        process = owned.return_value
                        process.directory = directory
                        process.wait.return_value = returncode
                        process.record.return_value = {'completed': True, 'group_cleaned': cleaned,
                                                       'worker_returncode': returncode}
                        with patch.object(gate.OWNED, 'write_record'):
                            with self.assertRaisesRegex(ValueError, 'local checks or teardown failed'):
                                gate.run_step(['make', 'check', 'secrets'], directory / 'checks.log',
                                              {'qualified': False}, directory)
                        self.assertGreaterEqual(process.stop.call_count, 1)
