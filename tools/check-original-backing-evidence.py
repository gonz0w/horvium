#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Require original-lease observations and complete owned native teardown."""
from pathlib import Path
import re
import sys

CASES = {'in_call_cancel', 'uncertainty_preserved', 'bounded_context', 'kernel_uid_denial', 'uncertainty_latched', 'close_finishes', 'restored_identity', 'cancelled', 'original_descriptor', 'directory_mode_denial', 'inert_lease', 'lock_hardlink_denial', 'native_domain', 'repeat', 'uncertainty_denial', 'closed_denial', 'final_close', 'reopen_new_origin', 'nil_lease', 'copied_lease', 'observation_finishes', 'filesystem_mismatch', 'invalid_projection', 'nil_reader', 'nil_context', 'same_inode_drift', 'lock_replacement_denial', 'namespace_drift', 'reader_error', 'lock_nonempty_denial', 'child_reaped', 'close_excluded', 'lock_mode_denial'}

def validate(text):
    if len(text) > 1 << 20 or re.search(r'(?m)^(?:FAIL|\s*--- (?:FAIL|SKIP):)', text):
        raise ValueError('failed/skipped/oversized original backing evidence')
    parents = re.findall(r'(?m)^--- PASS: (Test\w+) \([0-9.]+s\)$', text)
    if len(parents) != 2 or set(parents) != {'TestNativeOriginalBackingUFS','TestNativeOriginalBackingZFS'}:
        raise ValueError('both distinct original lease parents required')
    for kind in ('UFS','ZFS'):
        records = re.findall(r'ORIGINAL_BACKING_' + kind + r'_RESULT ([^\n]+)', text)
        if len(records) != 1:
            raise ValueError('one actual native result required')
        fields = records[0].split()
        if len(fields) != len(CASES) or set(fields) != {c+'=pass' for c in CASES}:
            raise ValueError('missing/duplicate/unknown/failed original lease assertion')
    teardown = re.findall(r'PERSISTENT_BACKING_TEARDOWN ([^\n]+)', text)
    if len(teardown) != 2 or set(teardown) != {k+' filesystem=pass device=pass directory=pass' for k in ('ufs','zfs')}:
        raise ValueError('both actual filesystem/device/directory teardowns required')

if __name__ == '__main__':
    validate(Path(sys.argv[1]).read_text())
