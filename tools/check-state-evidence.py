#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
from pathlib import Path
import re
import sys

CASES={'ancestor_denial', 'lock_owner_denial', 'lock_fifo_denial', 'mount_teardown', 'lease_close', 'directory_mode_denial', 'process_denial', 'close_release', 'final_close', 'fixture_teardown', 'directory_symlink_denial', 'second_writer_denial', 'lock_directory_denial', 'process_reaped', 'directory_owner_denial', 'lock_hardlink_denial', 'reopen', 'private_lock', 'root_directory', 'directory_sync', 'process_kill', 'death_release', 'path_denial', 'process_lock', 'death_inode_preservation', 'no_directory_creation', 'stable_inode', 'first_writer_race', 'cloexec', 'unsupported_fs_denial', 'lock_nonempty_denial', 'lock_symlink_denial', 'kernel_uid_denial', 'foreign_preservation', 'lock_mode_denial'}

def validate(text):
    if len(text)>1<<20 or re.search(r'(?m)^(?:FAIL|--- (?:FAIL|SKIP):)',text):
        raise ValueError('state native evidence failed, skipped or oversized')
    passed=re.findall(r'(?m)^--- PASS: TestNativeStateDirectory \([0-9.]+s\)$',text)
    records=re.findall(r'STATE_DIRECTORY_RESULT ([^\n]+)',text)
    if len(passed)!=1 or len(records)!=1:
        raise ValueError('one executed native state result and test pass required')
    fields=records[0].split();expected={case+'=pass' for case in CASES}
    if len(fields)!=len(expected) or set(fields)!=expected:
        raise ValueError('missing, duplicate, unknown or failed state assertion')

if __name__=='__main__':
    validate(Path(sys.argv[1]).read_text())
