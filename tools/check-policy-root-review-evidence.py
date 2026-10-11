#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Original root review requires both metadata backings and retained fact proof."""
import importlib.util
from pathlib import Path
import re
import sys

CLEAN = set('registered original_observation both_policies both_leases shared_deadline zero_eligibility copied_denied decoded_denied finish_denied finish_terminal unchanged_journal unchanged_ledger release_once locks_released registration_consumed'.split())
DENIAL = set('denied last_policy both_leases shared_deadline zero_eligibility unchanged_journal unchanged_ledger locks_released registration_consumed'.split())
ROOT = {'domain_teardown', 'mounts_restored'}
for mode, names in [('clean', CLEAN), ('last-member', DENIAL)]:
    ROOT.update(mode + '_' + n for n in names | {'seed_closed', 'child_waited', 'child_inventory', 'exact_history', 'leases_released'})


def load(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    m = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(m)
    return m


def validate(text):
    closed, facts = load('check-closed-startup-evidence'), load('check-policy-facts-evidence')
    found = closed.chunks(text)
    phases = ['facts begin timeout=18m', 'facts done', 'review begin timeout=18m', 'review done']
    if re.findall(r'(?m)^POLICY_ROOT_REVIEW_PHASE ([^\n]+)$', text) != phases:
        raise ValueError('two normally completed bounded phases required')
    fact_body = text.split('POLICY_ROOT_REVIEW_PHASE facts begin timeout=18m\n', 1)[1].split('POLICY_ROOT_REVIEW_PHASE facts done\n', 1)[0]
    facts.validate(fact_body)
    body = text.split('POLICY_ROOT_REVIEW_PHASE review begin timeout=18m\n', 1)[1].split('POLICY_ROOT_REVIEW_PHASE review done\n', 1)[0]
    parents = closed.chunks(body)
    wanted = ['TestNativeMultiplePolicyReviewUFS', 'TestNativeMultiplePolicyReviewZFS']
    if [n for n, _, _, _ in parents] != wanted or re.findall(r'(?m)^=== RUN   (\S+)$', body) != wanted or re.findall(r'(?m)^PASS$', body) != ['PASS']:
        raise ValueError('both normally completed original review parents required')
    if [n for n, _, _, _ in found] != ['TestNativeMultiplePolicyFacts'] + wanted or re.findall(r'(?m)^=== RUN   (\S+)$', text) != ['TestNativeMultiplePolicyFacts'] + wanted:
        raise ValueError('extra/missing/misplaced parents')
    if text.count('\nPASS\n') != 2:
        raise ValueError('both test processes must complete normally')
    for name, row, _, _ in parents:
        kind = 'UFS' if name.endswith('UFS') else 'ZFS'
        closed.controls(row, 'POLICY_ROOT_REVIEW_' + kind + '_RESULT', ROOT)
        closed.controls(row, 'POLICY_FACTS_CHILD_RESULT', facts.CHILD)
        closed.controls(row, 'POLICY_FACTS_PARENT_RESULT', facts.PARENT)
        if re.findall(r'POLICY_ROOT_REVIEW_DOMAIN_TEARDOWN ([^\n]+)', row) != [kind.lower() + ' filesystems=pass directories=pass']:
            raise ValueError('independent metadata teardown required')
    for marker, count in [('POLICY_FACTS_CHILD_RESULT ', 3), ('POLICY_FACTS_PARENT_RESULT ', 3), ('POLICY_ROOT_REVIEW_UFS_RESULT ', 1), ('POLICY_ROOT_REVIEW_ZFS_RESULT ', 1), ('POLICY_ROOT_REVIEW_DOMAIN_TEARDOWN ', 2)]:
        if text.count(marker) != count:
            raise ValueError('extra/missing controls or cleanup outside parents')


if __name__ == '__main__':
    validate(Path(sys.argv[1]).read_text())
