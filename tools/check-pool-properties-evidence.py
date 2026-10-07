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
       'policy_rename_denied','policy_closed_denied','system_root_backing','system_pool_backing',
       'safety_bound','safety_system_mount_denied','safety_fresh_restored','system_swap_observed',
       'safety_swap_disjoint','system_mount_preserved','system_fixture_teardown','unsupported_md_denied','providers_correlated','provider_geometry','providers_duplicate_denied',
       'providers_missing_denied','provider_fixtures_teardown','enrollment_bound','enrollment_legacy_denied','enrollment_member_drift_denied','enrollment_newer_denied','enrollment_restored','enrollment_catalog','enrollment_legacy_read','enrollment_publication','enrollment_publication_preserved','state_backing_gate','state_composed_gate','state_policy_drift_denied','state_unknown_denied','state_job_hold_denied','state_terminal_hold_denied','state_orphan_denied','state_uncertainty_latched','state_preserved','state_missing_policy_denied','enrollment_review_bound','enrollment_review_expired','enrollment_review_context_denied','enrollment_review_policy_drift','enrollment_review_native_drift','enrollment_review_missing_scope','enrollment_review_immutable','enrollment_review_preserved','enrollment_state_bound','enrollment_state_absent','enrollment_state_predecessor_denied','enrollment_state_record_drift','enrollment_state_unrelated_drift','enrollment_state_unknown_denied','enrollment_state_job_denied','enrollment_state_orphan_denied','enrollment_state_expired_denied','enrollment_state_context_denied','enrollment_state_preserved','enrollment_state_fixture_teardown','enrollment_source_review','enrollment_source_revalidate','enrollment_source_policy_denied','enrollment_source_member_denied','enrollment_source_cancelled','enrollment_source_context_denied','enrollment_source_store_uncertain','enrollment_source_preserved'}

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
