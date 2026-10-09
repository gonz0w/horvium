#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Require actual focused persistent backing probes and disposable teardown."""
from pathlib import Path
import re
import sys

CASES = {
    'UFS': {'native_id','actual_kernel_correspondence','repeat','record_change','record_identity_stable','cancelled','nil_context','invalid_fd','unsafe_mode','mode_denied','inherited_uid_denial','child_reaped','close_before_remount','remount_identity_stable','close','closed_denied'},
    'ZFS': {'native_id','exact_dataset','exact_pool','repeat','rename_identity_stable','close_before_remount','remount_identity_stable','distinct_dataset','other_close','close','name_reuse_new_identity','replacement_close'},
}

def validate(text):
    if len(text) > 1 << 20 or re.search(r'(?m)^(?:FAIL|\s*--- (?:FAIL|SKIP):)', text):
        raise ValueError('failed/skipped/oversized persistent backing evidence')
    parents = re.findall(r'(?m)^--- PASS: (Test\w+) \([0-9.]+s\)$', text)
    if len(parents) != 2 or set(parents) != {'TestNativePersistentBackingUFS','TestNativePersistentBackingZFS'}:
        raise ValueError('both distinct native filesystem parents required')
    for kind, cases in CASES.items():
        records = re.findall(r'PERSISTENT_BACKING_' + kind + r'_RESULT ([^\n]+)', text)
        if len(records) != 1:
            raise ValueError('one actual native result required')
        fields = records[0].split()
        if len(fields) != len(cases) or set(fields) != {c+'=pass' for c in cases}:
            raise ValueError('missing/duplicate/unknown/failed native assertion')
    teardown = re.findall(r'PERSISTENT_BACKING_TEARDOWN ([^\n]+)', text)
    if len(teardown) != 2 or set(teardown) != {k+' filesystem=pass device=pass directory=pass' for k in ('ufs','zfs')}:
        raise ValueError('both actual filesystem/device/directory teardowns required')

if __name__ == '__main__':
    validate(Path(sys.argv[1]).read_text())
