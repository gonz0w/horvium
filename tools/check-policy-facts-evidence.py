#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Require the complete native two-pool prerequisite; no root lifecycle claim."""
import importlib.util
from pathlib import Path
import re
import sys

CHILD = set('native_baselines distinct_pools distinct_members first_pass second_pass last_member_denied last_scope_denied zero_eligibility no_policy_mutation records_preserved bounded_context cancelled_zero'.split())
PARENT = set('child_waited child_inventory policy_bytes sentinel_first sentinel_second pools_removed mounts_restored swap_restored inputs_removed'.split())


def validate(text):
    spec = importlib.util.spec_from_file_location('closed', Path(__file__).with_name('check-closed-startup-evidence.py'))
    closed = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(closed)
    found = closed.chunks(text)
    wanted = ['TestNativeMultiplePolicyFacts']
    if [n for n, _, _, _ in found] != wanted or re.findall(r'(?m)^=== RUN   (\S+)$', text) != wanted:
        raise ValueError('exact completed two-pool parent required')
    if re.findall(r'(?m)^PASS$', text) != ['PASS']:
        raise ValueError('normally completed test process required')
    for marker, cases in [('POLICY_FACTS_CHILD_RESULT', CHILD), ('POLICY_FACTS_PARENT_RESULT', PARENT)]:
        closed.controls(found[0][1], marker, cases)
        if text.count(marker + ' ') != 1:
            raise ValueError('extra fact controls outside completed parent')


if __name__ == '__main__':
    validate(Path(sys.argv[1]).read_text())
