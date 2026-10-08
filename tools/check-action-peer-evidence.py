#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Require actual kernel peer cases and owned connection/fixture teardown."""
from pathlib import Path
import re
import sys

CASES = {'allowed', 'asserted-context', 'wrong-uid', 'root-denied', 'forged-uid',
         'unknown-operation', 'old-version', 'truncated', 'idle-deadline',
         'cancelled-before', 'local-uid-denied', 'live-cancellation', 'live-deadline',
         'disconnect', 'extra-frame', 'closed', 'nonroot-helper', 'fixtures-teardown'}


def validate(text):
    if len(text) > 1 << 20 or re.search(r'(?m)^(?:FAIL|\s*--- (?:FAIL|SKIP):)', text):
        raise ValueError('peer evidence failed, skipped or oversized')
    if len(re.findall(r'(?m)^--- PASS: TestNativeActionPeer \([0-9.]+s\)$', text)) != 1:
        raise ValueError('one actual native peer parent required')
    children = re.findall(r'(?m)^\s+--- PASS: TestNativeActionPeer/([^ ]+) \([0-9.]+s\)$', text)
    if len(children) != len(CASES) or set(children) != CASES:
        raise ValueError('missing, duplicate or unknown executed native peer child')
    results = re.findall(r'(?m)^ACTION_PEER_RESULT ([^\n]+)$', text)
    if len(results) != 1:
        raise ValueError('one complete native peer result required')
    fields = results[0].split()
    if len(fields) != len(CASES) or set(fields) != {case + '=pass' for case in CASES}:
        raise ValueError('missing, duplicate, unknown or failed peer assertion')


if __name__ == '__main__':
    validate(Path(sys.argv[1]).read_text())
