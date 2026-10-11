# SPDX-License-Identifier: BSD-3-Clause
"""Synthetic evidence-consumer regressions, never native proof."""
import unittest
from test_closed_startup_evidence import module
from test_policy_facts_evidence import fixture as facts_fixture, CHILD, PARENT

CLEAN = set('registered original_observation both_policies both_leases shared_deadline zero_eligibility copied_denied decoded_denied finish_denied finish_terminal unchanged_journal unchanged_ledger release_once locks_released registration_consumed'.split())
DENIAL = set('denied last_policy both_leases shared_deadline zero_eligibility unchanged_journal unchanged_ledger locks_released registration_consumed'.split())
ROOT = {'domain_teardown', 'mounts_restored'}
for mode, names in [('clean', CLEAN), ('last-member', DENIAL)]:
    ROOT.update(mode + '_' + n for n in names | {'seed_closed', 'child_waited', 'child_inventory', 'exact_history', 'leases_released'})


def fixture():
    text = 'POLICY_ROOT_REVIEW_PHASE facts begin timeout=18m\n' + facts_fixture() + 'POLICY_ROOT_REVIEW_PHASE facts done\nPOLICY_ROOT_REVIEW_PHASE review begin timeout=18m\n'
    for kind in ('UFS', 'ZFS'):
        name = 'TestNativeMultiplePolicyReview' + kind
        text += '=== RUN   ' + name + '\n'
        for marker, names in [('POLICY_FACTS_CHILD_RESULT', CHILD), ('POLICY_FACTS_PARENT_RESULT', PARENT), ('POLICY_ROOT_REVIEW_' + kind + '_RESULT', ROOT)]:
            text += marker + ' ' + ' '.join(n + '=pass' for n in sorted(names)) + '\n'
        text += 'POLICY_ROOT_REVIEW_DOMAIN_TEARDOWN ' + kind.lower() + ' filesystems=pass directories=pass\n--- PASS: ' + name + ' (1.0s)\n'
    return text + 'PASS\nPOLICY_ROOT_REVIEW_PHASE review done\n'


class RootReviewEvidence(unittest.TestCase):
    def test_all_independent_controls_and_cleanup_are_required(self):
        m = module('check-policy-root-review-evidence')
        self.assertEqual(m.ROOT, ROOT)
        self.assertEqual(len(ROOT), 36)
        m.validate(fixture())
        for control in ROOT | CHILD | PARENT:
            for replacement in ('', control + '=fail', control + '=pass ' + control + '=pass'):
                with self.subTest(control=control), self.assertRaises(ValueError):
                    m.validate(fixture().replace(control + '=pass', replacement))
        for bad in ('', fixture() * 2, fixture().replace('PASS:', 'SKIP:'), fixture().replace('PASS:', 'FAIL:'),
                    fixture().replace('DOMAIN_TEARDOWN ufs', 'DOMAIN_TEARDOWN absent'),
                    fixture() + 'POLICY_ROOT_REVIEW_UFS_RESULT extra=pass\n',
                    fixture() + '=== RUN   TestUnfinished\n', fixture().replace('PASS\n', ''),
                    fixture().replace('review done', 'review failed')):
            with self.assertRaises(ValueError):
                m.validate(bad)

    def test_retained_fact_phase_and_exact_parent_placement_are_required(self):
        import re
        m = module('check-policy-root-review-evidence')
        text = fixture()
        for kind in ('UFS', 'ZFS'):
            name = 'TestNativeMultiplePolicyReview' + kind
            block = re.search(r'(?ms)^=== RUN   ' + name + r'\n.*?^--- PASS: ' + name + r' \(1.0s\)\n', text)[0]
            for bad in (text.replace(block, ''), text.replace(block, '') + block,
                        text.replace(block, '').replace('POLICY_ROOT_REVIEW_PHASE facts done', block + 'POLICY_ROOT_REVIEW_PHASE facts done')):
                with self.assertRaises(ValueError):
                    m.validate(bad)
        with self.assertRaises(ValueError):
            m.validate(text.replace(facts_fixture(), ''))

    def test_owned_devices_checksum_and_host_teardown_are_independent(self):
        m = module('local-qualification')
        self.assertTrue(m.uses_device_fixture('policy-startup-review'))
        self.assertEqual(m.PREBUILT['policy-startup-review'][1], './internal/host/freebsd')
        m.validate_result('policy-startup-review', fixture())
        m.validate_result('policy-startup-review', fixture() + 'DEVICE_FIXTURE_CLEANUP pass\n', complete=True)
        for suffix in ('', 'DEVICE_FIXTURE_CLEANUP fail\n', 'DEVICE_FIXTURE_CLEANUP pass\n' * 2):
            with self.assertRaises(ValueError):
                m.validate_result('policy-startup-review', fixture() + suffix, complete=True)
        _, command = m.suite_command('policy-startup-review', artifact_digest='a' * 64)
        self.assertIn('HORVIUM_POLICY_ROOT_REVIEW_VM=1', command)
        for digest in (None, '', 'a' * 63):
            with self.assertRaises(ValueError):
                m.suite_command('policy-startup-review', artifact_digest=digest)
