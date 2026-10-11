# SPDX-License-Identifier: BSD-3-Clause
"""Diagnostic receipts are not lifecycle or performance acceptance."""
import subprocess
import sys
import unittest
from pathlib import Path
from test_closed_startup_evidence import module
from test_policy_facts_evidence import CHILD, PARENT

STAGES = ['direct-launch', 'guarded-launch', 'system', 'first-properties', 'first-providers', 'second-properties', 'second-providers', 'ufs-backing', 'zfs-backing']


def fixture():
    text = 'POLICY_STARTUP_COST_PROFILE cpu_count=1\n=== RUN   TestNativePolicyStartupCosts\n'
    for stage, reads in zip(STAGES, [8, 8, 28, 16, 16, 16, 16, 0, 0]):
        text += f'POLICY_STARTUP_COST stage={stage} reads={reads} elapsed_ns=100000000 read_ns={0 if reads == 0 else 90000000}\n'
    text += 'POLICY_STARTUP_COST_CLEANUP ufs filesystems=pass directories=pass mounts=pass\nPOLICY_STARTUP_COST_CLEANUP zfs filesystems=pass directories=pass mounts=pass\n'
    for marker, controls in [('POLICY_FACTS_CHILD_RESULT', CHILD), ('POLICY_FACTS_PARENT_RESULT', PARENT)]:
        text += marker + ' ' + ' '.join(c + '=pass' for c in sorted(controls)) + '\n'
    return text + '--- PASS: TestNativePolicyStartupCosts (1.0s)\nPASS\n'


class StartupCosts(unittest.TestCase):
    def test_ordered_complete_diagnostics_and_retained_controls(self):
        m = module('check-policy-startup-costs')
        self.assertEqual(m.STAGES, STAGES)
        m.validate(fixture())
        for line in fixture().splitlines(True):
            if 'COST' in line or 'RESULT' in line or 'PASS' in line:
                with self.subTest(line=line), self.assertRaises(ValueError):
                    m.validate(fixture().replace(line, ''))
                with self.assertRaises(ValueError):
                    m.validate(fixture() + line)
        for control in CHILD | PARENT:
            for replacement in ('', control + '=fail', control + '=pass ' + control + '=pass'):
                with self.assertRaises(ValueError):
                    m.validate(fixture().replace(control + '=pass', replacement))
        for bad in ('', fixture().replace('cpu_count=1', 'cpu_count=2'),
                    fixture().replace('PASS:', 'SKIP:'), fixture().replace('PASS:', 'FAIL:'),
                    fixture().replace('reads=28', 'reads=0'), fixture().replace('reads=16', 'reads=15'),
                    fixture().replace('elapsed_ns=100000000', 'elapsed_ns=-1'),
                    fixture().replace('read_ns=90000000', 'read_ns=100000001'),
                    fixture().replace('reads=0 elapsed_ns=100000000 read_ns=0', 'reads=1 elapsed_ns=100000000 read_ns=1'),
                    fixture().replace('stage=system', 'stage=unknown'), fixture() * 2, fixture() + '=== RUN   Unfinished\n'):
            with self.assertRaises(ValueError):
                m.validate(bad)

    def test_registered_checksum_cleanup_and_cpu_profile(self):
        m = module('local-qualification')
        self.assertTrue(m.uses_device_fixture('policy-startup-diagnostics'))
        self.assertEqual(m.cpu_count_for_suite('policy-startup-diagnostics'), 1)
        self.assertEqual(m.PREBUILT['policy-startup-diagnostics'][1], './internal/host/freebsd')
        m.validate_result('policy-startup-diagnostics', fixture() + 'DEVICE_FIXTURE_CLEANUP pass\n', complete=True)
        for suffix in ('', 'DEVICE_FIXTURE_CLEANUP fail\n', 'DEVICE_FIXTURE_CLEANUP pass\n' * 2):
            with self.assertRaises(ValueError):
                m.validate_result('policy-startup-diagnostics', fixture() + suffix, complete=True)
        for digest in (None, '', 'a' * 63, 'G' * 64):
            with self.assertRaises(ValueError):
                m.suite_command('policy-startup-diagnostics', artifact_digest=digest)
        _, command = m.suite_command('policy-startup-diagnostics', artifact_digest='a' * 64)
        self.assertIn('HORVIUM_POLICY_STARTUP_DIAGNOSTICS_VM=1', command)

    def test_dry_run_reports_selected_cpu_count(self):
        root = Path(__file__).resolve().parents[2]
        for suite, count in [('policy-startup-lifecycle', 1), ('policy-startup-diagnostics', 1), ('policy-startup-facts', 2)]:
            result = subprocess.run([sys.executable, str(root / 'tools/local-qualification.py'), suite, '15.1'], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0)
            self.assertIn(f'{count} CPUs', result.stdout)
