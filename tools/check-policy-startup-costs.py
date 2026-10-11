#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Strict diagnostic completeness; no factory or performance qualification."""
import importlib.util
from pathlib import Path
import re
import sys

STAGES = ['direct-launch', 'guarded-launch', 'system', 'first-properties', 'first-providers', 'second-properties', 'second-providers', 'ufs-backing', 'zfs-backing']
READS = [8, 8, 28, 16, 16, 16, 16, 0, 0]


def load(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    m = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(m)
    return m


def validate(text):
    closed, facts = load('check-closed-startup-evidence'), load('check-policy-facts-evidence')
    chunks = closed.chunks(text)
    if re.findall(r'(?m)^POLICY_STARTUP_COST_PROFILE ([^\n]+)$', text) != ['cpu_count=1']:
        raise ValueError('actual serial diagnostic kernel required')
    if [n for n, _, _, _ in chunks] != ['TestNativePolicyStartupCosts'] or re.findall(r'(?m)^=== RUN   (\S+)$', text) != ['TestNativePolicyStartupCosts'] or re.findall(r'(?m)^PASS$', text) != ['PASS']:
        raise ValueError('one normally completed diagnostic parent required')
    _, body, _, _ = chunks[0]
    if text.index('POLICY_STARTUP_COST_PROFILE ') > text.index('=== RUN'):
        raise ValueError('CPU profile must precede the parent')
    rows = re.findall(r'(?m)^POLICY_STARTUP_COST stage=([^ ]+) reads=([0-9]+) elapsed_ns=([0-9]+) read_ns=([0-9]+)$', body)
    if [r[0] for r in rows] != STAGES or text.count('POLICY_STARTUP_COST stage=') != 9:
        raise ValueError('complete ordered diagnostic stages required')
    for (_, reads, elapsed, read_ns), expected in zip(rows, READS):
        if int(reads) != expected or not 0 < int(elapsed) <= 480000000000 or not 0 <= int(read_ns) <= int(elapsed) or (expected == 0) != (int(read_ns) == 0):
            raise ValueError('bounded elapsed/read metrics and exact counts required')
    if re.findall(r'(?m)^POLICY_STARTUP_COST_CLEANUP ([^\n]+)$', body) != ['ufs filesystems=pass directories=pass mounts=pass', 'zfs filesystems=pass directories=pass mounts=pass'] or text.count('POLICY_STARTUP_COST_CLEANUP ') != 2:
        raise ValueError('both independent metadata cleanups required')
    for marker, controls in [('POLICY_FACTS_CHILD_RESULT', facts.CHILD), ('POLICY_FACTS_PARENT_RESULT', facts.PARENT)]:
        closed.controls(body, marker, controls)
        if text.count(marker + ' ') != 1:
            raise ValueError('extra/missing retained controls')


if __name__ == '__main__':
    validate(Path(sys.argv[1]).read_text())
