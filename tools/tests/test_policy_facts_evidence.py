# SPDX-License-Identifier: BSD-3-Clause
"""Synthetic evidence-parser cases do not qualify native collectors."""
import unittest
from test_closed_startup_evidence import module

CHILD = set('native_baselines distinct_pools distinct_members first_pass second_pass last_member_denied last_scope_denied zero_eligibility no_policy_mutation records_preserved bounded_context cancelled_zero'.split())
PARENT = set('child_waited child_inventory policy_bytes sentinel_first sentinel_second pools_removed mounts_restored swap_restored inputs_removed'.split())


def fixture():
    return ('=== RUN   TestNativeMultiplePolicyFacts\n'
            'POLICY_FACTS_CHILD_RESULT ' + ' '.join(c + '=pass' for c in sorted(CHILD)) + '\n'
            'POLICY_FACTS_PARENT_RESULT ' + ' '.join(c + '=pass' for c in sorted(PARENT)) + '\n'
            '--- PASS: TestNativeMultiplePolicyFacts (1.0s)\nPASS\n')


class PolicyFactsEvidence(unittest.TestCase):
    def test_complete_independent_inventory(self):
        m = module('check-policy-facts-evidence')
        self.assertEqual(m.CHILD, CHILD)
        self.assertEqual(m.PARENT, PARENT)
        m.validate(fixture())
        for control in CHILD | PARENT:
            for value in ('', control + '=fail', control + '=pass ' + control + '=pass'):
                with self.subTest(control=control, value=value), self.assertRaises(ValueError):
                    m.validate(fixture().replace(control + '=pass', value))

    def test_missing_extra_skipped_and_unfinished_parents_deny(self):
        m = module('check-policy-facts-evidence')
        good = fixture()
        for bad in ('', good * 2, good.replace('PASS:', 'SKIP:'), good.replace('PASS:', 'FAIL:'),
                    good.replace('POLICY_FACTS_CHILD_RESULT', 'missing'),
                    good + 'POLICY_FACTS_PARENT_RESULT extra=pass\n',
                    good + '=== RUN   TestUnfinished\n',
                    good + '--- PASS: TestUnknown (1.0s)\n',
                    good.replace('TestNativeMultiplePolicyFacts (', 'TestWrong ('),
                    good.replace('PASS\n', ''), 'x' * (2 << 20) + good):
            with self.subTest(bad=bad[:80]), self.assertRaises(ValueError):
                m.validate(bad)

    def test_host_cleanup_and_checksum_are_independent_requirements(self):
        m = module('local-qualification')
        self.assertTrue(m.uses_device_fixture('policy-startup-facts'))
        self.assertEqual(m.PREBUILT['policy-startup-facts'][1], './internal/host/freebsd')
        m.validate_result('policy-startup-facts', fixture())
        m.validate_result('policy-startup-facts', fixture() + 'DEVICE_FIXTURE_CLEANUP pass\n', complete=True)
        for suffix in ('', 'DEVICE_FIXTURE_CLEANUP fail\n', 'DEVICE_FIXTURE_CLEANUP pass\n' * 2):
            with self.assertRaises(ValueError):
                m.validate_result('policy-startup-facts', fixture() + suffix, complete=True)
        _, command = m.suite_command('policy-startup-facts', artifact_digest='a' * 64)
        self.assertIn('HORVIUM_POLICY_FACTS_VM=1', command)
        self.assertIn('HORVIUM_TEST_BINARY_SHA256=' + 'a' * 64, command)
        for digest in (None, '', 'a' * 63, 'G' * 64):
            with self.assertRaises(ValueError):
                m.suite_command('policy-startup-facts', artifact_digest=digest)
