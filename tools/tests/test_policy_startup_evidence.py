# SPDX-License-Identifier: BSD-3-Clause
"""Synthetic parser tests are never native qualification evidence."""
import unittest
import test_closed_startup_evidence as retained

module = retained.module

LIFECYCLE_MODES = ('clean', 'clean-again')
REVOCATION_MODES = ('late', 'uid')
DENIAL_MODES = ('legacy', 'member', 'scope', 'missing-pool')


class PolicyStartupEvidence(unittest.TestCase):
    def test_multiple_policy_lifecycle_inventory_is_required(self):
        checker = module('check-policy-startup-evidence')
        multi = {'multi_' + result for result in ('catalog', 'reviewed', 'activated', 'observed', 'closed', 'lineage', 'ledger', 'all-policies', 'leases', 'shared-deadline', 'retired', 'reopened')}
        for stage in ('review', 'activate', 'observe', 'finish', 'post-activate', 'post-finish'):
            multi.update('last-' + stage + '_' + result for result in ('denied', 'last-policy', 'leases', 'deadline', 'preserved', 'sticky', 'released', 'diagnostic'))
        self.assertTrue(multi <= checker.STORAGE)
        self.assertEqual(len(checker.STORAGE), 163 + 60)
        text = self.fixture('publication')
        checker.validate(text, 'publication')
        for control in multi:
            for replacement in ('', control + '=fail', control + '=pass ' + control + '=pass'):
                with self.subTest(control=control), self.assertRaises(ValueError):
                    checker.validate(text.replace(control + '=pass', replacement), 'publication')

    def test_complete_native_history_inventory_is_required(self):
        checker = module('check-policy-startup-evidence')
        history = {f'history-{relation}_{result}'
                   for relation in ('same-pool', 'other-pool', 'mixed')
                   for result in ('catalog', 'leases-held', 'denied', 'preserved', 'holds', 'released', 'reopened', 'retry')}
        self.assertTrue(history <= checker.STORAGE)
        self.assertEqual(len(checker.STORAGE), 139 + 24 + 60)
        text = self.fixture('publication')
        checker.validate(text, 'publication')
        for control in history:
            for replacement in ('', control + '=fail', control + '=pass ' + control + '=pass'):
                with self.subTest(control=control), self.assertRaises(ValueError):
                    checker.validate(text.replace(control + '=pass', replacement), 'publication')
        # Historical single-job receipts remain valid for their original checker,
        # but cannot satisfy this new complete mixed-namespace qualification.
        import re
        old = re.sub(r'history-(?:same-pool|other-pool|mixed)_[a-z-]+=pass ?', '', text)
        with self.assertRaises(ValueError):
            checker.validate(old, 'publication')

    def fixture(self, scope):
        baseline = retained.ClosedStartupEvidenceTests()
        baseline.setUp()
        old = baseline.publication() if scope == 'publication' else baseline.owner()
        text = old if scope == 'publication' else ''
        checker = module('check-policy-startup-evidence')
        groups = ('Lifecycle', 'Revocation', 'Denial') if scope == 'owner' else ('',)
        for group, kind in ((group, kind) for group in groups for kind in ('UFS', 'ZFS')):
            name = 'TestNativePolicyStartup' + (group + 'Owner' if scope == 'owner' else '') + kind
            marker = 'POLICY_STARTUP_' + (group.upper() + '_OWNER_' if scope == 'owner' else 'STORAGE_') + kind + '_RESULT'
            modes = {'Lifecycle': LIFECYCLE_MODES, 'Revocation': REVOCATION_MODES, 'Denial': DENIAL_MODES}.get(group, ())
            cases = ({c for c in checker.OWNER if c in ('data_sentinel_preserved', 'domain_teardown', 'mounts_restored') or any(c.startswith(mode + '_') for mode in modes)} if scope == 'owner' else checker.STORAGE)
            text += '=== RUN   ' + name + '\n' + marker + ' ' + ' '.join(c + '=pass' for c in sorted(cases)) + '\n'
            if scope == 'publication':
                text += 'STATE_BACKING_RESULT ' + name + ' filesystem=' + kind.lower() + ' softupdates=off teardown=pass\n'
            else:
                text += 'POLICY_STARTUP_DOMAIN_TEARDOWN ' + group.lower() + ' ' + kind.lower() + ' filesystems=pass directories=pass\nPOLICY_STARTUP_DATA_TEARDOWN ' + group.lower() + ' ' + kind.lower() + ' restored=pass\n'
            text += '--- PASS: ' + name + ' (1.0s)\n'
        if scope == 'owner':
            text = 'POLICY_STARTUP_OWNER_PHASE policy begin timeout=34m\n' + text + 'POLICY_STARTUP_OWNER_PHASE policy done\nPOLICY_STARTUP_OWNER_PHASE retained begin timeout=34m\n' + old + 'POLICY_STARTUP_OWNER_PHASE retained done\n'
        return text

    def test_independent_groups_preserve_all_cases_and_cleanup(self):
        import re
        checker = module('check-policy-startup-evidence')
        text = self.fixture('owner')
        checker.validate(text, 'owner')
        self.assertEqual(len(checker.OWNER_GROUPS['Lifecycle']), 31)
        self.assertEqual(len(checker.OWNER_GROUPS['Revocation']), 23)
        self.assertEqual(len(checker.OWNER_GROUPS['Denial']), 39)
        self.assertEqual(set.union(*checker.OWNER_GROUPS.values()), checker.OWNER)
        self.assertEqual(set.intersection(*checker.OWNER_GROUPS.values()), {'data_sentinel_preserved', 'domain_teardown', 'mounts_restored'})
        for group in ('Lifecycle', 'Revocation', 'Denial'):
            for kind in ('UFS', 'ZFS'):
                name = 'TestNativePolicyStartup' + group + 'Owner' + kind
                block = re.search(r'(?ms)^=== RUN   ' + name + r'\n.*?^--- PASS: ' + name + r' \(1.0s\)\n', text).group(0)
                for bad in (text.replace(block, ''), text + block, text.replace('DOMAIN_TEARDOWN ' + group.lower() + ' ' + kind.lower(), 'DOMAIN_TEARDOWN missing ' + kind.lower())):
                    with self.assertRaises(ValueError):
                        checker.validate(bad, 'owner')
        # Earlier corpora cannot stand in for all six independently owned parents.
        with self.assertRaises(ValueError):
            checker.validate(text.replace('LifecycleOwner', 'Owner').replace('RevocationOwner', 'Owner').replace('DenialOwner', 'Owner'), 'owner')

    def test_both_bounded_phases_are_required(self):
        import re
        from pathlib import Path
        checker = module('check-policy-startup-evidence')
        text = self.fixture('owner')
        checker.validate(text, 'owner')
        markers = re.findall(r'(?m)^POLICY_STARTUP_OWNER_PHASE [^\n]+\n', text)
        for marker in markers:
            for bad in (text.replace(marker, ''), text.replace(marker, marker * 2), text.replace(marker, marker.replace('34m', '68m').replace('done', 'failed'))):
                with self.assertRaises(ValueError):
                    checker.validate(bad, 'owner')
        with self.assertRaises(ValueError):
            checker.validate(text.replace(markers[0], '<swap>').replace(markers[2], markers[0]).replace('<swap>', markers[2]), 'owner')
        name = 'TestNativePolicyStartupRevocationOwnerUFS'
        block = re.search(r'(?ms)^=== RUN   ' + name + r'\n.*?^--- PASS: ' + name + r' \(1.0s\)\n', text).group(0)
        with self.assertRaises(ValueError):
            checker.validate(text.replace(block, '').replace(markers[2], markers[2] + block), 'owner')
        script = Path(checker.__file__).with_name('bsd-policy-startup-owner-prebuilt.sh').read_text()
        selectors = re.findall(r"-test.run='([^']+)'", script)
        self.assertEqual(len(selectors), 2)
        self.assertEqual(script.count('-test.timeout=34m'), 2)
        parents = {'TestNativePolicyStartup' + group + 'Owner' + kind for group in ('Lifecycle', 'Revocation', 'Denial') for kind in ('UFS', 'ZFS')} | {'TestNative' + family + kind for family in ('ExecutionStartup', 'ClosedStartupOwner') for kind in ('UFS', 'ZFS')}
        selected = [{name for name in parents if re.fullmatch(selector, name)} for selector in selectors]
        self.assertEqual([len(group) for group in selected], [6, 4])
        self.assertFalse(selected[0] & selected[1])
        self.assertEqual(selected[0] | selected[1], parents)

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
