#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
from pathlib import Path
import re
import sys

CASES = {'default', 'local', 'inherited', 'received', 'shadowed', 'no_limit',
         'rename', 'name_reuse', 'foreign_preservation', 'eligibility_denied', 'teardown',
         'identity_binding', 'complete_filesystems', 'snapshot_owner', 'bookmark_owner',
         'replicated_guid', 'snapshot_rename', 'snapshot_reuse', 'volume_denial', 'stable_enumeration',
         'state_binding', 'mount_sources', 'readonly_denial', 'encrypted_loaded_denial',
         'encrypted_unloaded_denial', 'encryption_root_binding', 'state_drift', 'state_preservation',
         'pool_identity_binding', 'pool_source_metadata', 'pool_bootfs_denial', 'pool_readonly_denial',
         'import_incarnation', 'reimport_drift', 'pool_preservation', 'pool_name_reuse'}

def validate(text):
    if len(text) > 1 << 20 or re.search(r'(?m)^(?:FAIL|--- (?:FAIL|SKIP):)', text):
        raise ValueError('property native evidence failed, skipped or oversized')
    passed = re.findall(r'(?m)^--- PASS: TestNativePropertyObservations \([0-9.]+s\)$', text)
    records = re.findall(r'PROPERTY_OBSERVATION_RESULT ([^\n]+)', text)
    if len(passed) != 1 or len(records) != 1:
        raise ValueError('one executed native property result and test pass required')
    fields = records[0].split()
    expected = {case + '=pass' for case in CASES}
    if len(fields) != len(expected) or set(fields) != expected:
        raise ValueError('missing, duplicate, unknown or failed native property case')

if __name__ == '__main__':
    validate(Path(sys.argv[1]).read_text())
