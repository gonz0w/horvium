#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
from pathlib import Path
import re
import sys

CASES={'ancestor_denial', 'lock_owner_denial', 'lock_fifo_denial', 'mount_teardown', 'lease_close', 'directory_mode_denial', 'process_denial', 'close_release', 'final_close', 'fixture_teardown', 'directory_symlink_denial', 'second_writer_denial', 'lock_directory_denial', 'process_reaped', 'directory_owner_denial', 'lock_hardlink_denial', 'reopen', 'private_lock', 'root_directory', 'directory_sync', 'process_kill', 'death_release', 'path_denial', 'process_lock', 'death_inode_preservation', 'no_directory_creation', 'stable_inode', 'first_writer_race', 'cloexec', 'unsupported_fs_denial', 'lock_nonempty_denial', 'lock_symlink_denial', 'kernel_uid_denial', 'foreign_preservation', 'lock_mode_denial'}

READ_CASES={'missing_denial','selector_denial','valid_read','bytes_preserved','independent_policy','inode_preserved','kernel_uid_denial','mode_denial','owner_denial','hardlink_denial','symlink_denial','fifo_denial','directory_denial','empty_denial','oversize_denial','damage_denial','newer_denial','wrong_pool_denial','controls_preserved','directory_anchor','directory_drift_denial','lock_drift_denial','lock_replacement_denial','lease_close','closed_denial','inert_denial','foreign_preservation','fixture_teardown'}
TESTS={'TestNativeStateDirectory','TestNativeOwnershipRead','TestOwnershipRecordRoundTripAndCanonicalBytes','TestOwnershipChecksumDoesNotGrantInvalidScope','TestOwnershipRecordStrictDenials','TestOwnershipRecordRejectsInvalidPolicies','TestOwnershipRecordCapacityAndRevision','TestOwnershipReadMetadataDrift'}
CODEC_SUBTEST_COUNTS={'TestOwnershipChecksumDoesNotGrantInvalidScope':10,'TestOwnershipRecordStrictDenials':32,'TestOwnershipRecordRejectsInvalidPolicies':15}

def validate(text):
    if len(text)>1<<20 or re.search(r'(?m)^(?:FAIL|[ \t]*--- (?:FAIL|SKIP):)',text):
        raise ValueError('state native evidence failed, skipped or oversized')
    passed=re.findall(r'(?m)^--- PASS: (Test\w+) \([0-9.]+s\)$',text)
    if len(passed)!=len(TESTS) or set(passed)!=TESTS:
        raise ValueError('complete executed lease/reader/codec tests required')
    for marker,cases in [('STATE_DIRECTORY_RESULT',CASES),('OWNERSHIP_READ_RESULT',READ_CASES)]:
        records=re.findall(marker+r' ([^\n]+)',text)
        if len(records)!=1:raise ValueError('one native result per boundary required')
        fields=records[0].split();expected={case+'=pass' for case in cases}
        if len(fields)!=len(expected) or set(fields)!=expected:
            raise ValueError('missing, duplicate, unknown or failed state assertion')
    children=re.findall(r'(?m)^[ \t]+--- PASS: (TestOwnership\w+)/(\S+) \([0-9.]+s\)$',text)
    if len(children)!=sum(CODEC_SUBTEST_COUNTS.values()) or len(set(children))!=len(children):
        raise ValueError('complete distinct codec subtests required')
    for name,count in CODEC_SUBTEST_COUNTS.items():
        if sum(parent==name for parent,_ in children)!=count:raise ValueError('incomplete codec test family')

if __name__=='__main__':
    validate(Path(sys.argv[1]).read_text())
