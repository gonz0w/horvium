#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Validate the executed focused pool-property collection and owned teardown."""
from pathlib import Path
import re
import sys

CASES={'complete','pool_identity','dataset_identity','property_values','inheritance',
       'snapshot_owner','features','topology','eligibility_denied','wrong_guid_denied',
       'rename_fresh','readonly_denied','volume_denied','preservation','backing_preserved','teardown',
       'policy_bound','policy_missing_denied','policy_change_denied','policy_scope_denied',
       'policy_rename_denied','policy_closed_denied'}

def validate(text):
    if len(text)>1<<20 or re.search(r'(?m)^(?:FAIL|--- (?:FAIL|SKIP):)',text):
        raise ValueError('pool-property evidence failed, skipped or oversized')
    passed=re.findall(r'(?m)^--- PASS: TestNativePoolPropertyCollection \([0-9.]+s\)$',text)
    records=re.findall(r'POOL_PROPERTY_COLLECTION_RESULT ([^\n]+)',text)
    if len(passed)!=1 or len(records)!=1:
        raise ValueError('one actual native collector test/result required')
    fields=records[0].split()
    if len(fields)!=len(CASES) or set(fields)!={case+'=pass' for case in CASES}:
        raise ValueError('missing, duplicate, unknown or failed native collector case')

if __name__=='__main__':
    validate(Path(sys.argv[1]).read_text())
