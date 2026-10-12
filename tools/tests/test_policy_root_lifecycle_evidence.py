# SPDX-License-Identifier: BSD-3-Clause
"""Synthetic evidence-consumer regressions, never native proof."""
import re
import json
import unittest
from test_closed_startup_evidence import module
from test_policy_root_review_evidence import fixture as review_fixture
from test_policy_facts_evidence import CHILD as FACT_CHILD, PARENT as FACT_PARENT

CHILD = set('cpu_affinity registered active_successor active_private_record current_boot leases_held nonjournal_unchanged original_observation observation_preserves_history copied_denied decoded_denied cancelled_no_effect closed_successor ledger_unchanged close_retains_leases retired_observation retired_finish release_once locks_released registration_consumed budget_profile startup_caller_budget'.split())
ROOT = {'seed_closed', 'domain_teardown', 'mounts_restored', 'parent_budget'}
for mode in ('first', 'restart'):
    ROOT.update(mode + '_' + name for name in CHILD | {'child_waited', 'child_inventory', 'exact_closed', 'ledger_preserved', 'leases_released'})


STAGES = ['startup', 'observe', 'copied', 'copied-finish', 'decoded', 'decoded-finish', 'finish', 'retired-observe', 'retired-finish', 'registration-consumed']

def stages(mode):
    rows = []
    for stage in STAGES:
        budget = (180 if stage == 'startup' else 90) * 1000000000
        row = dict(mode=mode, stage=stage, elapsed_ns=1000000, budget_ns=budget, deadline_ns=budget,
                   deadline_unchanged=True, operation_error='none' if stage in ('startup', 'observe', 'finish') else 'denied', caller_error='none')
        if stage == 'startup':
            row['owner_present'] = True
        rows.append(row)
    return rows


def fixture():
    text = 'POLICY_ROOT_LIFECYCLE_CPU_PROFILE cpu_count=1\n' + review_fixture() + 'POLICY_ROOT_LIFECYCLE_PHASE lifecycle begin timeout=44m\nPOLICY_ROOT_LIFECYCLE_BUDGET_PROFILE caller_seconds=180 child_seconds=390 parent_work_seconds=1200 phase_minutes=44\nPOLICY_ROOT_LIFECYCLE_AFFINITY phase=lifecycle cpus=0\n'
    for kind in ('UFS', 'ZFS'):
        name = 'TestNativeMultiplePolicyLifecycle' + kind
        text += '=== RUN   ' + name + '\n'
        for mode in ('first', 'restart'):
            for row in stages(mode):
                text += 'POLICY_ROOT_LIFECYCLE_STAGE ' + json.dumps(row, separators=(',', ':')) + '\n'
        for marker, names in [('POLICY_ROOT_LIFECYCLE_' + kind + '_RESULT', ROOT), ('POLICY_FACTS_CHILD_RESULT', FACT_CHILD), ('POLICY_FACTS_PARENT_RESULT', FACT_PARENT)]:
            text += marker + ' ' + ' '.join(n + '=pass' for n in sorted(names)) + '\n'
        text += 'POLICY_ROOT_LIFECYCLE_DOMAIN_TEARDOWN ' + kind.lower() + ' filesystems=pass directories=pass\n--- PASS: ' + name + ' (1.0s)\n'
    return text + 'PASS\nPOLICY_ROOT_LIFECYCLE_PHASE lifecycle done\n'


class RootLifecycleEvidence(unittest.TestCase):
    def test_all_controls_and_independent_cleanup_are_required(self):
        m = module('check-policy-root-lifecycle-evidence')
        self.assertEqual(m.ROOT, ROOT)
        self.assertEqual(len(CHILD), 22)
        self.assertEqual(len(ROOT), 58)
        m.validate(fixture())
        for control in ROOT | FACT_CHILD | FACT_PARENT:
            for replacement in ('', control + '=fail', control + '=pass ' + control + '=pass'):
                with self.subTest(control=control), self.assertRaises(ValueError):
                    m.validate(fixture().replace(control + '=pass', replacement))
        for bad in (fixture().replace('POLICY_ROOT_LIFECYCLE_CPU_PROFILE cpu_count=1\n', ''),
                    fixture().replace('cpu_count=1', 'cpu_count=2'),
                    fixture() + 'POLICY_ROOT_LIFECYCLE_CPU_PROFILE cpu_count=1\n',
                    fixture().replace('POLICY_ROOT_LIFECYCLE_AFFINITY phase=lifecycle cpus=0\n', ''),
                    fixture().replace('cpus=0', 'cpus=0,1'),
                    fixture() + 'POLICY_ROOT_LIFECYCLE_AFFINITY phase=lifecycle cpus=0\n',
                    '', fixture() * 2, fixture().replace('PASS:', 'SKIP:'), fixture().replace('PASS:', 'FAIL:'),
                    fixture().replace('LIFECYCLE_DOMAIN_TEARDOWN ufs', 'LIFECYCLE_DOMAIN_TEARDOWN absent'),
                    fixture() + 'POLICY_ROOT_LIFECYCLE_UFS_RESULT extra=pass\n', fixture() + '=== RUN   TestUnfinished\n',
                    fixture().replace('PASS\n', ''), fixture().replace('lifecycle done', 'lifecycle failed')):
            with self.assertRaises(ValueError):
                m.validate(bad)

    def test_all_retained_phases_and_exact_parent_placement_are_required(self):
        m = module('check-policy-root-lifecycle-evidence')
        text = fixture()
        for kind in ('UFS', 'ZFS'):
            name = 'TestNativeMultiplePolicyLifecycle' + kind
            block = re.search(r'(?ms)^=== RUN   ' + name + r'\n.*?^--- PASS: ' + name + r' \(1.0s\)\n', text)[0]
            for bad in (text.replace(block, ''), text.replace(block, '') + block,
                        text.replace(block, '').replace('POLICY_ROOT_REVIEW_PHASE review done', block + 'POLICY_ROOT_REVIEW_PHASE review done')):
                with self.assertRaises(ValueError):
                    m.validate(bad)
        for bad in (text.replace(review_fixture(), ''), text.replace('review done', 'review failed'), text.replace('facts done', 'facts failed')):
            with self.assertRaises(ValueError):
                m.validate(bad)

    def test_device_checksum_and_host_cleanup_are_independent(self):
        m = module('local-qualification')
        self.assertTrue(m.uses_device_fixture('policy-startup-lifecycle'))
        self.assertEqual(m.PREBUILT['policy-startup-lifecycle'][1], './internal/host/freebsd')
        m.validate_result('policy-startup-lifecycle', fixture())
        m.validate_result('policy-startup-lifecycle', fixture() + 'DEVICE_FIXTURE_CLEANUP pass\n', complete=True)
        for suffix in ('', 'DEVICE_FIXTURE_CLEANUP fail\n', 'DEVICE_FIXTURE_CLEANUP pass\n' * 2):
            with self.assertRaises(ValueError):
                m.validate_result('policy-startup-lifecycle', fixture() + suffix, complete=True)
        _, command = m.suite_command('policy-startup-lifecycle', artifact_digest='a' * 64)
        self.assertIn('HORVIUM_POLICY_ROOT_LIFECYCLE_VM=1', command)
        for digest in (None, '', 'a' * 63, 'G' * 64):
            with self.assertRaises(ValueError):
                m.suite_command('policy-startup-lifecycle', artifact_digest=digest)


class SerialGuestSelection(unittest.TestCase):
    def test_only_lifecycle_selects_one_cpu_and_defaults_are_preserved(self):
        m = module('local-qualification')
        for suite in m.SUITES:
            self.assertEqual(m.cpu_count_for_suite(suite), 1 if suite in ('policy-startup-lifecycle', 'policy-startup-diagnostics', 'policy-startup-trace', 'policy-startup-envelope', 'policy-startup-provenance') else 2)
        with self.assertRaises(ValueError):
            m.cpu_count_for_suite('unknown')
        self.assertEqual(m.qemu_command('qemu', 'base', 'code', 'vars', 1234)[6], '2')
        for count in (1, 2):
            self.assertEqual(m.qemu_command('qemu', 'base', 'code', 'vars', 1234, cpu_count=count)[6], str(count))
        for count in (0, 3, True, '1', None):
            with self.assertRaises(ValueError):
                m.qemu_command('qemu', 'base', 'code', 'vars', 1234, cpu_count=count)


class LifecycleBudgetEvidence(unittest.TestCase):
    def test_profile_and_old_evidence_are_refused(self):
        m = module('check-policy-root-lifecycle-evidence')
        text = fixture()
        m.validate(text)
        marker = 'POLICY_ROOT_LIFECYCLE_BUDGET_PROFILE caller_seconds=180 child_seconds=390 parent_work_seconds=1200 phase_minutes=44\n'
        for bad in (text.replace(marker, ''), text + marker,
                    text.replace(marker, '').replace('POLICY_ROOT_REVIEW_PHASE facts begin', marker + 'POLICY_ROOT_REVIEW_PHASE facts begin'),
                    text.replace('child_seconds=390', 'child_seconds=330'),
                    text.replace('parent_work_seconds=1200', 'parent_work_seconds=1080'),
                    text.replace('phase_minutes=44', 'phase_minutes=40'),
                    text.replace(marker, marker.replace('\n', ' unknown=1\n')),
                    text.replace('lifecycle begin timeout=44m', 'lifecycle begin timeout=18m'),
                    text.replace(marker, '').replace('lifecycle begin timeout=44m', 'lifecycle begin timeout=18m').replace('parent_budget=pass ', '').replace('first_budget_profile=pass ', '').replace('restart_budget_profile=pass ', '')):
            with self.subTest(profile=bad), self.assertRaises(ValueError):
                m.validate(bad)

    def test_host_profile_is_fixed_and_scoped(self):
        m = module('local-qualification')
        for suite in m.SUITES:
            expected = {'caller_seconds': 180, 'child_seconds': 390, 'parent_work_seconds': 1200, 'phase_minutes': 44} if suite == 'policy-startup-lifecycle' else None
            self.assertEqual(m.lifecycle_budget_for_suite(suite), expected)
        with self.assertRaises(ValueError):
            m.lifecycle_budget_for_suite('unknown')
        profile = m.lifecycle_budget_for_suite('policy-startup-lifecycle')
        profile['child_seconds'] = 999
        self.assertEqual(m.lifecycle_budget_for_suite('policy-startup-lifecycle')['child_seconds'], 390)


class LifecycleStageEvidence(unittest.TestCase):
    def test_exact_ordered_scalar_observations(self):
        m = module('check-policy-root-lifecycle-evidence')
        self.assertEqual(len(m.validate(fixture())), 40)
        row = json.dumps(stages('first')[0], separators=(',', ':'))
        line = 'POLICY_ROOT_LIFECYCLE_STAGE ' + row + '\n'
        for bad in (fixture().replace(line, '', 1), fixture()+line,
                    fixture().replace(line, line+line, 1), fixture().replace('POLICY_ROOT_LIFECYCLE_STAGE ', 'RAW ', 1)):
            with self.assertRaises(ValueError):
                m.validate(bad)
        for field, values in {'mode':['restart','unknown'], 'stage':['observe','unknown'],
                'elapsed_ns':[0,-1,True,180000000001,'1'], 'budget_ns':[120000000000,True,0],
                'deadline_ns':[0,179999999999,180000000001], 'deadline_unchanged':[False,1],
                'operation_error':['deadline','cancelled','other','raw /root/private'],
                'caller_error':['deadline','cancelled','unknown'], 'owner_present':[False,1,None]}.items():
            for value in values:
                altered=dict(stages('first')[0]);altered[field]=value
                with self.subTest(field=field,value=value), self.assertRaises(ValueError):
                    m.validate(fixture().replace(row,json.dumps(altered,separators=(',',':')),1))
        for altered in (row[:-1]+',"mode":"first"}',row[:-1]+',"native_acceptance":true}',row.replace('"stage":"startup",','')):
            with self.assertRaises(ValueError):
                m.validate(fixture().replace(row,altered,1))

    def test_all_modes_and_denials_require_current_unexpired_callers(self):
        m=module('check-policy-root-lifecycle-evidence')
        text=fixture()
        for mode in ('first','restart'):
            for item in stages(mode):
                row=json.dumps(item,separators=(',',':'))
                for field,value in [('caller_error','cancelled'),('deadline_unchanged',False),('budget_ns',1),('operation_error','other')]:
                    bad=dict(item);bad[field]=value
                    with self.subTest(mode=mode,stage=item['stage'],field=field),self.assertRaises(ValueError):
                        m.validate(text.replace(row,json.dumps(bad,separators=(',',':')),1))
        first=json.dumps(stages('first')[0],separators=(',',':'));second=json.dumps(stages('first')[1],separators=(',',':'))
        with self.assertRaises(ValueError):
            m.validate(text.replace(first,'PLACEHOLDER',1).replace(second,first,1).replace('PLACEHOLDER',second))

    def test_failed_stage_partial_never_qualifies_release(self):
        m=module('check-policy-root-lifecycle-evidence');text=fixture()
        row=json.dumps(stages('restart')[0],separators=(',',':'));bad=dict(stages('restart')[0])
        bad.update(operation_error='deadline',caller_error='deadline',owner_present=False,elapsed_ns=180010000000)
        text=text.replace(row,json.dumps(bad,separators=(',',':')),1).replace('--- PASS: TestNativeMultiplePolicyLifecycleUFS','--- FAIL: TestNativeMultiplePolicyLifecycleUFS')
        with self.assertRaises(ValueError):m.validate(text)
