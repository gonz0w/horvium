# SPDX-License-Identifier: BSD-3-Clause
"""Synthetic transcript coverage only; no native trace or lifecycle claim."""
import copy
import json
import unittest
from test_closed_startup_evidence import module
from test_policy_startup_envelope import fixture as envelope_fixture, summary

SOURCE = 'a' * 40
COMPILER = 'b' * 64
ARTIFACT = 'c' * 64
EXTRA_CHILD = set('trace_enabled error_preserved snapshots_before_cancel trace_inventory original_contexts'.split())
EXTRA_PARENT = set('trace_extracted trace_sanitized source_isolated'.split())


def frame(site='activation', phase='final-inspection', **changes):
    row = dict(site=site, operation='deadline', phase=phase, branch='0', allowance_ns=90000000000,
               child_present=True, child_deadline=True, child_deadline_ns=90000000000,
               parent_deadline=True, parent_deadline_ns=180000000000,
               child_before='deadline', child_after='deadline', parent_before='none', parent_after='none',
               begin_ns=121000000000, end_ns=121000000001, constraint='local-expiry')
    row.update(changes)
    return row


def trace():
    return {'status': 'complete', 'frames': [frame()]}


def fixture(record=None, operation=None):
    s = summary()
    s.update(return_ns=122000000000, caller_error='none')
    if record and record.get('frames'):
        s['return_ns'] = max(s['return_ns'], max((f.get('end_ns', 0) for f in record['frames'] if type(f) is dict and type(f.get('end_ns')) is int), default=0) + 1)
        if s['return_ns'] >= 180000000000:
            s['caller_error'] = 'deadline'
    if operation:
        s.update(startup_error=operation, owner_present=operation == 'none')
    text = envelope_fixture(s).replace('POLICY_STARTUP_ENVELOPE_', 'POLICY_STARTUP_PROVENANCE_').replace('TestNativeMultiplePolicyStartupEnvelopeUFS', 'TestNativeMultiplePolicyStartupProvenanceUFS')
    row = next(x for x in text.splitlines() if x.startswith('POLICY_STARTUP_PROVENANCE_UFS_RESULT '))
    text = text.replace(row, row + ' ' + ' '.join(x + '=pass' for x in sorted(EXTRA_PARENT | {'child_' + c for c in EXTRA_CHILD})))
    identity = dict(schema=1, profile='deadline-provenance-v1', source_commit=SOURCE, compiler_sha256=COMPILER, artifact_sha256=ARTIFACT)
    return ('POLICY_STARTUP_PROVENANCE_PREBUILT_SHA256 ' + ARTIFACT + '\n' +
            'POLICY_STARTUP_PROVENANCE_SOURCE ' + json.dumps(identity) + '\n' +
            text.replace('POLICY_STARTUP_PROVENANCE_SUMMARY ', 'POLICY_STARTUP_PROVENANCE_TRACE ' + json.dumps(record if record is not None else trace(), separators=(',', ':')) + '\nPOLICY_STARTUP_PROVENANCE_SUMMARY '))


class PolicyStartupProvenance(unittest.TestCase):
    def setUp(self):
        self.m = module('check-policy-startup-provenance')

    def validate(self, text, **expected):
        return self.m.validate(text, **(dict(source_commit=SOURCE, compiler_sha256=COMPILER, artifact_sha256=ARTIFACT) | expected))

    def reject_trace(self, record):
        with self.assertRaises(ValueError):
            self.validate(fixture(record))

    def test_exact_inventory_and_complete_error_is_diagnostic_only(self):
        old = module('check-policy-startup-envelope')
        facts = module('check-policy-facts-evidence')
        self.assertEqual(self.m.CHILD, old.CHILD | EXTRA_CHILD)
        self.assertEqual(self.m.ROOT, old.ROOT | EXTRA_PARENT | {'child_' + c for c in EXTRA_CHILD})
        self.assertEqual((len(self.m.CHILD), len(self.m.ROOT), len(facts.CHILD) + len(facts.PARENT)), (13, 29, 21))
        result = self.validate(fixture())
        self.assertTrue(result['diagnostic_complete'])
        self.assertFalse(result['native_acceptance'])
        self.assertEqual(result['trace'], trace())

    def test_success_nondeadline_and_incomplete_states_stay_distinct(self):
        for status, error, complete in [('success', 'none', True), ('not-deadline', 'denied', True),
                                         ('unattributed', 'deadline', False), ('ambiguous', 'deadline', False),
                                         ('invalid', 'deadline', False), ('overflow', 'deadline', False)]:
            with self.subTest(status=status):
                result = self.validate(fixture({'status': status, 'frames': []}, error))
                self.assertEqual(result['diagnostic_complete'], complete)
                self.assertFalse(result['native_acceptance'])
        for status in ('success', 'not-deadline', 'complete', 'disabled', 'unknown'):
            self.reject_trace({'status': status, 'frames': []})
        for status in ('success', 'not-deadline', 'unattributed'):
            self.reject_trace({'status': status, 'frames': [frame()]})
        with self.assertRaises(ValueError):
            self.validate(fixture(trace(), 'none'))

    def test_every_control_and_whole_parent_is_required(self):
        log = fixture()
        for marker, controls in [('POLICY_STARTUP_PROVENANCE_UFS_RESULT', self.m.ROOT),
                                  ('POLICY_FACTS_CHILD_RESULT', self.m.facts.CHILD),
                                  ('POLICY_FACTS_PARENT_RESULT', self.m.facts.PARENT)]:
            row = next(x for x in log.splitlines() if x.startswith(marker + ' '))
            for control in controls:
                for replacement in ('', control + '=fail', control + '=pass ' + control + '=pass'):
                    with self.subTest(marker=marker, control=control, replacement=replacement), self.assertRaises(ValueError):
                        self.validate(log.replace(row, row.replace(control + '=pass', replacement)))
            with self.assertRaises(ValueError):
                self.validate(log.replace(row, row + ' unknown=pass'))
        for bad in ('', log * 2, log.replace('--- PASS:', '--- FAIL:'), log.replace('PASS\n', 'SKIP\n'),
                    log.replace('child_seconds=210', 'child_seconds=150'), log.replace('cpu_count=1', 'cpu_count=2'),
                    log.replace('DOMAIN_TEARDOWN', 'MISSING'), log + '=== RUN   TestOther\n'):
            with self.assertRaises(ValueError):
                self.validate(bad)

    def test_expected_candidate_identity_is_external_and_exact(self):
        for key, value in [('source_commit', 'd' * 40), ('compiler_sha256', 'd' * 64), ('artifact_sha256', 'd' * 64),
                           ('source_commit', True), ('compiler_sha256', 'private')]:
            with self.subTest(key=key, value=value), self.assertRaises(ValueError):
                self.validate(fixture(), **{key: value})
        for marker in ('SOURCE', 'PREBUILT_SHA256', 'PROFILE', 'TRACE'):
            log = fixture()
            row = next(x for x in log.splitlines() if x.startswith('POLICY_STARTUP_PROVENANCE_' + marker + ' '))
            for bad in (log.replace(row + '\n', ''), log + row + '\n'):
                with self.assertRaises(ValueError):
                    self.validate(bad)
        for old in (envelope_fixture(), fixture().replace('PROVENANCE', 'TRACE')):
            with self.assertRaises(ValueError):
                self.validate(old)
        log = fixture()
        for bad in (log.replace('"schema": 1', '"schema": true'), log.replace('deadline-provenance-v1', 'point-only'),
                    log.replace('"source_commit":', '"unknown":'), log.replace('"schema": 1', '"schema": 1, "schema": 1'),
                    log.replace('"status":"complete"', '"status":"complete","status":"complete"')):
            with self.assertRaises(ValueError):
                self.validate(bad)

    def test_shapes_bounds_types_and_raw_output(self):
        for key, value in [('extra', 'secret'), ('frames', None), ('status', True)]:
            r = trace(); r[key] = value
            self.reject_trace(r)
        for key, value in [('site', 'raw command'), ('phase', 'unknown'), ('operation', 'none'), ('branch', '/private'),
                           ('begin_ns', True), ('begin_ns', -1), ('end_ns', 210000000001), ('allowance_ns', 10000000000),
                           ('child_present', 1), ('parent_before', 'absent'), ('child_before', 'unknown'),
                           ('child_deadline_ns', 180000000001), ('parent_deadline_ns', -1), ('extra', 'inventory')]:
            self.reject_trace({'status': 'complete', 'frames': [frame(**{key: value})]})
        r = trace(); del r['frames'][0]['branch']; self.reject_trace(r)
        r = trace(); r['frames'] *= 13; self.reject_trace(r)
        self.reject_trace({'status': 'complete', 'frames': [frame(), frame()]})
        for bad in [fixture() + 'private raw error\n', fixture() + 'goroutine 9 [syscall]:\n',
                    fixture().replace('POLICY_STARTUP_PROVENANCE_TRACE ', 'raw error\nPOLICY_STARTUP_PROVENANCE_TRACE '),
                    fixture().replace('"status":"complete"', '"raw":"' + 'x' * 8192 + '","status":"complete"')]:
            with self.assertRaises(ValueError):
                self.validate(bad)

    def test_constraint_semantics_and_original_path(self):
        collector = frame('system-collector', 'collection', allowance_ns=45000000000, child_deadline_ns=90000000000,
                          parent_deadline_ns=90000000000, child_before='deadline', child_after='deadline',
                          parent_before='deadline', parent_after='deadline', constraint='both-expired',
                          begin_ns=120000000000, end_ns=120000000001)
        command = frame('system-command', 'capture', allowance_ns=10000000000, child_deadline_ns=90000000000,
                        parent_deadline_ns=90000000000, child_before='deadline', child_after='deadline',
                        parent_before='deadline', parent_after='deadline', constraint='both-expired',
                        begin_ns=119000000000, end_ns=119000000001)
        r = {'status': 'complete', 'frames': [command, collector, frame()]}
        self.assertTrue(self.validate(fixture(r))['diagnostic_complete'])
        for mutate in [lambda x: x['frames'].reverse(), lambda x: x['frames'][0].update(constraint='local-expiry'),
                       lambda x: x['frames'][1].update(site='backing-collector'),
                       lambda x: x['frames'][1].update(child_deadline_ns=89999999999),
                       lambda x: x['frames'][0].update(branch='0.1'),
                       lambda x: x['frames'][2].update(parent_deadline_ns=179000000000)]:
            bad = copy.deepcopy(r); mutate(bad); self.reject_trace(bad)
        guard = frame('review', 'guard', allowance_ns=0, child_present=False, child_deadline=False,
                      child_deadline_ns=0, child_before='absent', child_after='absent', parent_before='deadline',
                      parent_after='deadline', begin_ns=180000000000, end_ns=180000000001, constraint='parent-only')
        self.assertTrue(self.validate(fixture({'status': 'complete', 'frames': [guard]}))['diagnostic_complete'])
        for changes in [dict(child_deadline=True), dict(allowance_ns=90000000000), dict(child_after='deadline')]:
            self.reject_trace({'status': 'complete', 'frames': [guard | changes]})
        overlap = frame(child_before='none', constraint='overlap')
        self.reject_trace({'status': 'complete', 'frames': [overlap]})
        self.assertFalse(self.validate(fixture({'status': 'ambiguous', 'frames': [overlap]}))['diagnostic_complete'])

    def test_original_samples_journal_and_teardown_still_required(self):
        log = fixture()
        for bad in [log.replace('"bytes": 100', '"bytes": 65536'), log.replace('"line": 26', '"line": 0'),
                    log.replace('"unchanged": true', '"unchanged": false'), log.replace('"generation_before": 2', '"generation_before": true'),
                    log.replace('"generation_after": 2', '"generation_after": false')]:
            with self.assertRaises(ValueError):
                self.validate(bad)

    def test_scalar_recursion_nonfinite_and_parent_duration_rejected(self):
        log = fixture()
        for bad in [log.replace('"bytes": 100', '"bytes": NaN'),
                    log.replace('"bytes": 100', '"bytes": Infinity'),
                    log.replace('"bytes": 100', '"bytes": ' + '[' * 1100 + '0' + ']' * 1100),
                    log.replace('(200.0s)', '(1.0s)'), log.replace('(200.0s)', '(1081.0s)')]:
            with self.assertRaises(ValueError):
                self.validate(bad)
        for value in [None, 'private', [], {'unknown': True}]:
            self.reject_trace({'status': 'complete', 'frames': [value]})

    def test_reused_provider_scope_and_inherited_unresolved_canceled_frames(self):
        # Original command errors can propagate while their containing collector
        # and activation contexts are still live; attribution is observation only.
        root = frame(child_before='none', child_after='none', constraint='unresolved')
        provider = frame('provider-collector', 'collection', allowance_ns=45000000000,
                         child_deadline_ns=45000000000, parent_deadline_ns=90000000000,
                         child_before='none', child_after='none', constraint='unresolved',
                         begin_ns=120000000000, end_ns=120000000001)
        command = frame('system-command', 'capture', allowance_ns=10000000000,
                        child_deadline_ns=10000000000, parent_deadline_ns=45000000000,
                        begin_ns=119000000000, end_ns=119000000001)
        self.assertTrue(self.validate(fixture({'status': 'complete', 'frames': [command, provider, root]}))['diagnostic_complete'])
        for changes in [dict(child_deadline_ns=180000000000, child_before='none', child_after='none', constraint='inherited'),
                        dict(child_before='canceled', child_after='canceled', constraint='canceled'),
                        dict(child_before='none', child_after='none', constraint='unresolved')]:
            self.assertTrue(self.validate(fixture({'status': 'complete', 'frames': [frame(**changes)]}))['diagnostic_complete'])
        branches = [frame(branch='0.0'), frame(branch='0.1')]
        result = self.validate(fixture({'status': 'ambiguous', 'frames': branches}))
        self.assertEqual(result['trace']['frames'], branches)
        self.assertFalse(result['diagnostic_complete'])

    def test_cli_records_incomplete_result_without_success_exit(self):
        import subprocess
        import sys
        import tempfile
        from pathlib import Path
        with tempfile.TemporaryDirectory() as name:
            path = Path(name) / 'synthetic.txt'
            command = [sys.executable, self.m.__file__, str(path), '--source-commit', SOURCE,
                       '--compiler-sha256', COMPILER, '--artifact-sha256', ARTIFACT]
            for text, code, complete in [(fixture(), 0, True),
                                         (fixture({'status': 'unattributed', 'frames': []}), 1, False),
                                         ('raw inventory', 2, None)]:
                path.write_text(text)
                result = subprocess.run(command, capture_output=True, text=True, timeout=10)
                self.assertEqual(result.returncode, code)
                if complete is not None:
                    record = json.loads(result.stdout)
                    self.assertEqual(record['diagnostic_complete'], complete)
                    self.assertFalse(record['native_acceptance'])
                else:
                    self.assertEqual(result.stdout, '')
                    self.assertNotIn('raw inventory', result.stderr)

    def test_guard_requires_an_actual_guard_site_and_expired_parent(self):
        guard = frame('review', 'guard', allowance_ns=0, child_present=False, child_deadline=False,
                      child_deadline_ns=0, child_before='absent', child_after='absent',
                      constraint='parent-only')
        # A parent-only deadline guard can only return its original expired Err.
        with self.subTest(case='live parent'):
            self.reject_trace({'status': 'complete', 'frames': [guard]})
        # Collectors always create their original child before their error check;
        # only command captures and review/activation have pre-child guards.
        guard.update(site='backing-collector', parent_deadline_ns=90000000000,
                     parent_before='deadline', parent_after='deadline', begin_ns=120000000000, end_ns=120000000001)
        with self.subTest(case='invented collector guard'):
            self.reject_trace({'status': 'complete', 'frames': [guard, frame()]})
