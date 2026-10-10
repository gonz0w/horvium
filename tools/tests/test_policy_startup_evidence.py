# SPDX-License-Identifier: BSD-3-Clause
"""Synthetic parser tests are never native qualification evidence."""
import unittest
import test_closed_startup_evidence as retained

module = retained.module

LIFECYCLE_MODES = ('clean', 'clean-again', 'late', 'uid')
DENIAL_MODES = ('legacy', 'member', 'scope', 'missing-pool')


class PolicyStartupEvidence(unittest.TestCase):
    def fixture(self, scope):
        baseline = retained.ClosedStartupEvidenceTests()
        baseline.setUp()
        text = baseline.publication() if scope == 'publication' else baseline.owner()
        checker = module('check-policy-startup-evidence')
        groups = ('Lifecycle', 'Denial') if scope == 'owner' else ('',)
        for group, kind in ((group, kind) for group in groups for kind in ('UFS', 'ZFS')):
            name = 'TestNativePolicyStartup' + (group + 'Owner' if scope == 'owner' else '') + kind
            marker = 'POLICY_STARTUP_' + (group.upper() + '_OWNER_' if scope == 'owner' else 'STORAGE_') + kind + '_RESULT'
            modes = LIFECYCLE_MODES if group == 'Lifecycle' else DENIAL_MODES
            cases = ({c for c in checker.OWNER if c in ('data_sentinel_preserved', 'domain_teardown', 'mounts_restored') or any(c.startswith(mode + '_') for mode in modes)} if scope == 'owner' else checker.STORAGE)
            text += '=== RUN   ' + name + '\n' + marker + ' ' + ' '.join(c + '=pass' for c in sorted(cases)) + '\n'
            if scope == 'publication':
                text += 'STATE_BACKING_RESULT ' + name + ' filesystem=' + kind.lower() + ' softupdates=off teardown=pass\n'
            else:
                text += 'POLICY_STARTUP_DOMAIN_TEARDOWN ' + group.lower() + ' ' + kind.lower() + ' filesystems=pass directories=pass\nPOLICY_STARTUP_DATA_TEARDOWN ' + group.lower() + ' ' + kind.lower() + ' restored=pass\n'
            text += '--- PASS: ' + name + ' (1.0s)\n'
        return text

    def test_independent_groups_preserve_all_cases_and_cleanup(self):
        import re
        checker = module('check-policy-startup-evidence')
        text = self.fixture('owner')
        checker.validate(text, 'owner')
        self.assertEqual(len(checker.OWNER_GROUPS['Lifecycle']), 51)
        self.assertEqual(len(checker.OWNER_GROUPS['Denial']), 39)
        self.assertEqual(checker.OWNER_GROUPS['Lifecycle'] | checker.OWNER_GROUPS['Denial'], checker.OWNER)
        self.assertEqual(checker.OWNER_GROUPS['Lifecycle'] & checker.OWNER_GROUPS['Denial'], {'data_sentinel_preserved', 'domain_teardown', 'mounts_restored'})
        for group in ('Lifecycle', 'Denial'):
            for kind in ('UFS', 'ZFS'):
                name = 'TestNativePolicyStartup' + group + 'Owner' + kind
                block = re.search(r'(?ms)^=== RUN   ' + name + r'\n.*?^--- PASS: ' + name + r' \(1.0s\)\n', text).group(0)
                for bad in (text.replace(block, ''), text + block, text.replace('DOMAIN_TEARDOWN ' + group.lower() + ' ' + kind.lower(), 'DOMAIN_TEARDOWN missing ' + kind.lower())):
                    with self.assertRaises(ValueError):
                        checker.validate(bad, 'owner')
        # The earlier two-parent corpus must not stand in for these four parents.
        with self.assertRaises(ValueError):
            checker.validate(text.replace('LifecycleOwner', 'Owner').replace('DenialOwner', 'Owner'), 'owner')

    def test_each_new_and_retained_control_is_required(self):
        checker = module('check-policy-startup-evidence')
        for scope in ('publication', 'owner'):
            text = self.fixture(scope)
            checker.validate(text, scope)
            for c in checker.STORAGE if scope == 'publication' else checker.OWNER:
                for replacement in ('', c + '=fail', c + '=pass ' + c + '=pass'):
                    with self.subTest(scope=scope, control=c), self.assertRaises(ValueError):
                        checker.validate(text.replace(c + '=pass', replacement), scope)
            for bad in (text * 2, text.replace('PASS:', 'SKIP:'), text.replace('PASS:', 'FAIL:'), text.replace('teardown=pass', 'teardown=fail'), text.replace('restored=pass', 'restored=fail') if scope == 'owner' else text.replace('softupdates=off', 'softupdates=on'), text.replace('POLICY_STARTUP_', 'MISSING_')):
                with self.assertRaises(ValueError):
                    checker.validate(bad, scope)
            with self.assertRaises(ValueError):
                checker.validate(text[text.index('=== RUN   TestNativePolicyStartup'):], scope)

    def test_runner_checks_artifact_and_complete_cleanup(self):
        runner = module('local-qualification')
        for scope in ('publication', 'owner'):
            suite = 'policy-startup-' + scope
            text = self.fixture(scope)
            runner.validate_result(suite, text)
            if scope == 'owner':
                with self.assertRaises(ValueError):
                    runner.validate_result(suite, text, complete=True)
                text += 'DEVICE_FIXTURE_CLEANUP pass\n'
            runner.validate_result(suite, text, complete=True)
            runner.validate_prebuilt('POLICY_STARTUP_' + scope.upper() + '_PREBUILT_SHA256 ' + 'a' * 64 + '\n', 'a' * 64, suite)
            with self.assertRaises(ValueError):
                runner.validate_prebuilt('', 'a' * 64, suite)
            self.assertEqual(runner.uses_device_fixture(suite), scope == 'owner')
