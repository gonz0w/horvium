#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Validate the actual foreground status command and owned native teardown."""
from pathlib import Path
import re
import sys

CASES={'built_process','empty','mixed_complete','queued_held','applying_held','observing_held','terminal_retained','private_baseline_hidden','repeat_preserved','lease_released','active_writer_denied','kernel_uid_denied','unknown_denied','newer_denied','unsafe_file_denied','orphan_denied','cancelled_denied','output_cancelled','native_unavailable_not_consulted','lock_preserved','fixtures_teardown'}

def validate(text):
    if len(text)>1<<20 or re.search(r'(?m)^(?:FAIL|\s*--- (?:FAIL|SKIP):)',text):
        raise ValueError('status evidence failed, skipped or oversized')
    if len(re.findall(r'(?m)^--- PASS: TestNativeStorageJobStatus \([0-9.]+s\)$',text))!=1:
        raise ValueError('one actual native status test required')
    records=re.findall(r'STORAGE_JOB_STATUS_RESULT ([^\n]+)',text)
    if len(records)!=1:
        raise ValueError('one complete native status result required')
    fields=records[0].split()
    if len(fields)!=len(CASES) or set(fields)!={case+'=pass' for case in CASES}:
        raise ValueError('missing, duplicate, unknown or failed status assertion')
    if len(re.findall(r'STORAGE_JOB_STATUS_BACKING filesystem=(?:ufs|zfs) softupdates=(?:on|off) teardown=pass',text))!=1:
        raise ValueError('actual metadata backing and teardown required')

if __name__=='__main__':
    validate(Path(sys.argv[1]).read_text())
