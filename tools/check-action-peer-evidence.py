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


EXCHANGE_CASES = {'review', 'submit', 'duplicate', 'refused', 'queued-request',
                  'second-operation', 'second-receive', 'second-reply', 'wrong-uid',
                  'root-denied', 'local-uid', 'malformed-request', 'wrong-request-id',
                  'request-extra', 'request-cut', 'request-oversize', 'idle-deadline',
                  'cancellation', 'explicit-close', 'reply-extra', 'reply-missing-eof',
                  'reply-wrong-selection', 'reply-wrong-id', 'overlap',
                  'nonroot-helper', 'fixtures-teardown'}

ADMITTED_CASES = {'review', 'submit', 'refused', 'wrong-uid', 'root-denied',
                  'terminal-slot-reuse', 'malformed-slot-reuse', 'capacity-recovery',
                  'negotiation-deadline', 'phase-deadline', 'earlier-parent-deadline',
                  'shutdown-negotiation', 'shutdown-request', 'overlap-negotiation',
                  'duplicate-lease', 'fixtures-teardown'}


def validate(text):
    if len(text) > 1 << 20 or re.search(r'(?m)^(?:FAIL|\s*--- (?:FAIL|SKIP):)', text):
        raise ValueError('peer evidence failed, skipped or oversized')
    for parent, marker, cases in (
            ('TestNativeActionPeer', 'ACTION_PEER_RESULT', CASES),
            ('TestNativePropertyExchange', 'PROPERTY_EXCHANGE_RESULT', EXCHANGE_CASES),
            ('TestNativeAdmittedPropertyExchange', 'ADMITTED_PROPERTY_RESULT', ADMITTED_CASES)):
        if len(re.findall(r'(?m)^--- PASS: ' + parent + r' \([0-9.]+s\)$', text)) != 1:
            raise ValueError('one actual native parent required: ' + parent)
        children = re.findall(r'(?m)^\s+--- PASS: ' + parent + r'/([^ ]+) \([0-9.]+s\)$', text)
        if len(children) != len(cases) or set(children) != cases:
            raise ValueError('missing, duplicate or unknown executed child: ' + parent)
        results = re.findall(r'(?m)^' + marker + r' ([^\n]+)$', text)
        if len(results) != 1:
            raise ValueError('one complete native result required: ' + marker)
        fields = results[0].split()
        if len(fields) != len(cases) or set(fields) != {case + '=pass' for case in cases}:
            raise ValueError('missing, duplicate, unknown or failed assertion: ' + marker)



if __name__ == '__main__':
    validate(Path(sys.argv[1]).read_text())
