#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Both original lifecycles and every retained parent must complete normally."""
from pathlib import Path
import re
import json
import sys
import importlib.util

CHILD = set('cpu_affinity registered active_successor active_private_record current_boot leases_held nonjournal_unchanged original_observation observation_preserves_history copied_denied decoded_denied cancelled_no_effect closed_successor ledger_unchanged close_retains_leases retired_observation retired_finish release_once locks_released registration_consumed budget_profile startup_caller_budget'.split())
ROOT = {'seed_closed', 'domain_teardown', 'mounts_restored', 'parent_budget'}
for mode in ('first', 'restart'):
    ROOT.update(mode + '_' + name for name in CHILD | {'child_waited', 'child_inventory', 'exact_closed', 'ledger_preserved', 'leases_released'})


def load(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    m = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(m)
    return m



STAGES = ('startup', 'observe', 'copied', 'copied-finish', 'decoded', 'decoded-finish', 'finish', 'retired-observe', 'retired-finish', 'registration-consumed')

def stage_observations(text):
    raw = re.findall(r'(?m)^POLICY_ROOT_LIFECYCLE_STAGE ([^\n]+)$', text)
    if len(raw) != 20 or text.count('POLICY_ROOT_LIFECYCLE_STAGE ') != 20:
        raise ValueError('twenty ordered original lifecycle scalars required')
    def unique(items):
        result = {}
        for k, v in items:
            if k in result:
                raise ValueError('duplicate scalar key')
            result[k] = v
        return result
    observations = []
    for data, (mode, stage) in zip(raw, ((mode, stage) for mode in ('first', 'restart') for stage in STAGES)):
        try:
            row = json.loads(data, object_pairs_hook=unique)
        except (json.JSONDecodeError, TypeError) as e:
            raise ValueError('invalid scalar JSON') from e
        names = {'mode', 'stage', 'elapsed_ns', 'budget_ns', 'deadline_ns', 'deadline_unchanged', 'operation_error', 'caller_error'}
        if stage == 'startup':
            names.add('owner_present')
        budget = (180 if stage == 'startup' else 90) * 1000000000
        operation = 'none' if stage in ('startup', 'observe', 'finish') else 'denied'
        if type(row) is not dict or set(row) != names or row['mode'] != mode or row['stage'] != stage:
            raise ValueError('exact ordered mode/stage scalar shape required')
        if any(type(row[k]) is not int for k in ('elapsed_ns', 'budget_ns', 'deadline_ns')) or not 0 < row['elapsed_ns'] <= budget or row['budget_ns'] != budget or row['deadline_ns'] != budget:
            raise ValueError('bounded original scalar deadline/time required')
        if row['deadline_unchanged'] is not True or row['caller_error'] != 'none' or row['operation_error'] != operation or (stage == 'startup' and row['owner_present'] is not True):
            raise ValueError('original successful/denied pre-cancel scalar results required')
        observations.append(row)
    return observations

def validate(text):
    closed, facts, review = load('check-closed-startup-evidence'), load('check-policy-facts-evidence'), load('check-policy-root-review-evidence')
    found = closed.chunks(text)
    if re.findall(r'(?m)^POLICY_ROOT_LIFECYCLE_CPU_PROFILE ([^\n]+)$', text) != ['cpu_count=1'] or text.index('POLICY_ROOT_LIFECYCLE_CPU_PROFILE ') > text.index('POLICY_ROOT_REVIEW_PHASE facts begin'):
        raise ValueError('verified serial guest CPU profile required before phases')
    phases = [('REVIEW', 'facts begin timeout=18m'), ('REVIEW', 'facts done'), ('REVIEW', 'review begin timeout=18m'), ('REVIEW', 'review done'), ('LIFECYCLE', 'lifecycle begin timeout=44m'), ('LIFECYCLE', 'lifecycle done')]
    if re.findall(r'(?m)^POLICY_ROOT_(REVIEW|LIFECYCLE)_PHASE ([^\n]+)$', text) != phases:
        raise ValueError('three normally completed bounded phases required')
    prefix, body = text.split('POLICY_ROOT_LIFECYCLE_PHASE lifecycle begin timeout=44m\n', 1)
    review.validate(prefix)
    body = body.split('POLICY_ROOT_LIFECYCLE_PHASE lifecycle done\n', 1)[0]
    if re.findall(r'(?m)^POLICY_ROOT_LIFECYCLE_AFFINITY ([^\n]+)$', body) != ['phase=lifecycle cpus=0'] or text.count('POLICY_ROOT_LIFECYCLE_AFFINITY ') != 1:
        raise ValueError('exact lifecycle CPU-affinity profile required inside its phase')
    if re.findall(r'(?m)^POLICY_ROOT_LIFECYCLE_BUDGET_PROFILE ([^\n]+)$', body) != ['caller_seconds=180 child_seconds=390 parent_work_seconds=1200 phase_minutes=44'] or text.count('POLICY_ROOT_LIFECYCLE_BUDGET_PROFILE ') != 1:
        raise ValueError('exact adopted lifecycle budget profile required inside its phase')
    parents = closed.chunks(body)
    wanted = ['TestNativeMultiplePolicyLifecycleUFS', 'TestNativeMultiplePolicyLifecycleZFS']
    if [n for n, _, _, _ in parents] != wanted or re.findall(r'(?m)^=== RUN   (\S+)$', body) != wanted or re.findall(r'(?m)^PASS$', body) != ['PASS']:
        raise ValueError('two normally completed original lifecycle parents required')
    all_wanted = ['TestNativeMultiplePolicyFacts', 'TestNativeMultiplePolicyReviewUFS', 'TestNativeMultiplePolicyReviewZFS'] + wanted
    if [n for n, _, _, _ in found] != all_wanted or re.findall(r'(?m)^=== RUN   (\S+)$', text) != all_wanted or text.count('\nPASS\n') != 3:
        raise ValueError('extra/missing/misplaced parents or incomplete processes')
    observations = []
    for name, row, _, _ in parents:
        kind = 'UFS' if name.endswith('UFS') else 'ZFS'
        observations.extend(stage_observations(row))
        closed.controls(row, 'POLICY_ROOT_LIFECYCLE_' + kind + '_RESULT', ROOT)
        closed.controls(row, 'POLICY_FACTS_CHILD_RESULT', facts.CHILD)
        closed.controls(row, 'POLICY_FACTS_PARENT_RESULT', facts.PARENT)
        if re.findall(r'POLICY_ROOT_LIFECYCLE_DOMAIN_TEARDOWN ([^\n]+)', row) != [kind.lower() + ' filesystems=pass directories=pass']:
            raise ValueError('independent lifecycle metadata teardown required')
    if text.count('POLICY_ROOT_LIFECYCLE_STAGE ') != 40:
        raise ValueError('extra/missing scalar records outside parents')
    for marker, count in [('POLICY_FACTS_CHILD_RESULT ', 5), ('POLICY_FACTS_PARENT_RESULT ', 5), ('POLICY_ROOT_REVIEW_UFS_RESULT ', 1), ('POLICY_ROOT_REVIEW_ZFS_RESULT ', 1), ('POLICY_ROOT_REVIEW_DOMAIN_TEARDOWN ', 2), ('POLICY_ROOT_LIFECYCLE_UFS_RESULT ', 1), ('POLICY_ROOT_LIFECYCLE_ZFS_RESULT ', 1), ('POLICY_ROOT_LIFECYCLE_DOMAIN_TEARDOWN ', 2)]:
        if text.count(marker) != count:
            raise ValueError('extra/missing controls or cleanup outside parents')

    return observations


if __name__ == '__main__':
    validate(Path(sys.argv[1]).read_text())
