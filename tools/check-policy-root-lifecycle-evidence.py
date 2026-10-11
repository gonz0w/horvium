#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Both original lifecycles and every retained parent must complete normally."""
from pathlib import Path
import re
import sys
import importlib.util

CHILD = set('cpu_affinity registered active_successor active_private_record current_boot leases_held nonjournal_unchanged original_observation observation_preserves_history copied_denied decoded_denied cancelled_no_effect closed_successor ledger_unchanged close_retains_leases retired_observation retired_finish release_once locks_released registration_consumed budget_profile'.split())
ROOT = {'seed_closed', 'domain_teardown', 'mounts_restored', 'parent_budget'}
for mode in ('first', 'restart'):
    ROOT.update(mode + '_' + name for name in CHILD | {'child_waited', 'child_inventory', 'exact_closed', 'ledger_preserved', 'leases_released'})


def load(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    m = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(m)
    return m


def validate(text):
    closed, facts, review = load('check-closed-startup-evidence'), load('check-policy-facts-evidence'), load('check-policy-root-review-evidence')
    found = closed.chunks(text)
    if re.findall(r'(?m)^POLICY_ROOT_LIFECYCLE_CPU_PROFILE ([^\n]+)$', text) != ['cpu_count=1'] or text.index('POLICY_ROOT_LIFECYCLE_CPU_PROFILE ') > text.index('POLICY_ROOT_REVIEW_PHASE facts begin'):
        raise ValueError('verified serial guest CPU profile required before phases')
    phases = [('REVIEW', 'facts begin timeout=18m'), ('REVIEW', 'facts done'), ('REVIEW', 'review begin timeout=18m'), ('REVIEW', 'review done'), ('LIFECYCLE', 'lifecycle begin timeout=40m'), ('LIFECYCLE', 'lifecycle done')]
    if re.findall(r'(?m)^POLICY_ROOT_(REVIEW|LIFECYCLE)_PHASE ([^\n]+)$', text) != phases:
        raise ValueError('three normally completed bounded phases required')
    prefix, body = text.split('POLICY_ROOT_LIFECYCLE_PHASE lifecycle begin timeout=40m\n', 1)
    review.validate(prefix)
    body = body.split('POLICY_ROOT_LIFECYCLE_PHASE lifecycle done\n', 1)[0]
    if re.findall(r'(?m)^POLICY_ROOT_LIFECYCLE_AFFINITY ([^\n]+)$', body) != ['phase=lifecycle cpus=0'] or text.count('POLICY_ROOT_LIFECYCLE_AFFINITY ') != 1:
        raise ValueError('exact lifecycle CPU-affinity profile required inside its phase')
    if re.findall(r'(?m)^POLICY_ROOT_LIFECYCLE_BUDGET_PROFILE ([^\n]+)$', body) != ['child_seconds=330 parent_work_seconds=1080 phase_minutes=40'] or text.count('POLICY_ROOT_LIFECYCLE_BUDGET_PROFILE ') != 1:
        raise ValueError('exact adopted lifecycle budget profile required inside its phase')
    parents = closed.chunks(body)
    wanted = ['TestNativeMultiplePolicyLifecycleUFS', 'TestNativeMultiplePolicyLifecycleZFS']
    if [n for n, _, _, _ in parents] != wanted or re.findall(r'(?m)^=== RUN   (\S+)$', body) != wanted or re.findall(r'(?m)^PASS$', body) != ['PASS']:
        raise ValueError('two normally completed original lifecycle parents required')
    all_wanted = ['TestNativeMultiplePolicyFacts', 'TestNativeMultiplePolicyReviewUFS', 'TestNativeMultiplePolicyReviewZFS'] + wanted
    if [n for n, _, _, _ in found] != all_wanted or re.findall(r'(?m)^=== RUN   (\S+)$', text) != all_wanted or text.count('\nPASS\n') != 3:
        raise ValueError('extra/missing/misplaced parents or incomplete processes')
    for name, row, _, _ in parents:
        kind = 'UFS' if name.endswith('UFS') else 'ZFS'
        closed.controls(row, 'POLICY_ROOT_LIFECYCLE_' + kind + '_RESULT', ROOT)
        closed.controls(row, 'POLICY_FACTS_CHILD_RESULT', facts.CHILD)
        closed.controls(row, 'POLICY_FACTS_PARENT_RESULT', facts.PARENT)
        if re.findall(r'POLICY_ROOT_LIFECYCLE_DOMAIN_TEARDOWN ([^\n]+)', row) != [kind.lower() + ' filesystems=pass directories=pass']:
            raise ValueError('independent lifecycle metadata teardown required')
    for marker, count in [('POLICY_FACTS_CHILD_RESULT ', 5), ('POLICY_FACTS_PARENT_RESULT ', 5), ('POLICY_ROOT_REVIEW_UFS_RESULT ', 1), ('POLICY_ROOT_REVIEW_ZFS_RESULT ', 1), ('POLICY_ROOT_REVIEW_DOMAIN_TEARDOWN ', 2), ('POLICY_ROOT_LIFECYCLE_UFS_RESULT ', 1), ('POLICY_ROOT_LIFECYCLE_ZFS_RESULT ', 1), ('POLICY_ROOT_LIFECYCLE_DOMAIN_TEARDOWN ', 2)]:
        if text.count(marker) != count:
            raise ValueError('extra/missing controls or cleanup outside parents')


if __name__ == '__main__':
    validate(Path(sys.argv[1]).read_text())
