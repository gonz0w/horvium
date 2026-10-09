#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Research-only native evidence, including real reboot and fixture teardown."""
import json
import re
from pathlib import Path

BEFORE = {'fixture-preflight', 'process-observer', 'owner-death', 'stopped-guard',
          'guard-first', 'queued-cut', 'applying-cut', 'effect-cut', 'normal-completion',
          'retained-for-reboot'}
AFTER = {'new-boot', 'exact-fixture-import', 'unchanged-ledger', 'holds-and-no-authority',
         'independent-properties', 'fixture-preservation', 'fixture-teardown'}


def validate(text):
    if len(text) > 1 << 20 or re.search(r'(?m)^(?:FAIL|\s*--- (?:FAIL|SKIP):)', text):
        raise ValueError('failed, skipped or oversized executor research')
    for phase, cases in [('Before', BEFORE), ('After', AFTER)]:
        parent = 'TestNativeExecutorRecovery' + phase
        names = [parent] + [parent + '/' + name for name in cases]
        for name in names:
            if len(re.findall(r'(?m)^\s*--- PASS: ' + re.escape(name) + r' \([0-9.]+s\)$', text)) != 1:
                raise ValueError('complete once-only executor research required')
        children = re.findall(r'(?m)^\s*--- PASS: ' + re.escape(parent) + r'/([^ /]+) \([0-9.]+s\)$', text)
        if len(children) != len(cases) or set(children) != cases:
            raise ValueError('unknown or duplicate research cases')
    records = re.findall(r'(?m)^EXECUTOR_REBOOT (\{[^\n]+\})$', text)
    if len(records) != 1:
        raise ValueError('one independently observed reboot required')
    reboot = json.loads(records[0])
    if set(reboot) != {'before', 'after'}:
        raise ValueError('unknown reboot fields')
    for boot in reboot.values():
        if (not isinstance(boot, list) or len(boot) != 2 or
                any(type(v) is not int for v in boot) or boot[0] < 1 or not 0 <= boot[1] < 1000000):
            raise ValueError('invalid boot timeval')
    if reboot['before'] == reboot['after']:
        raise ValueError('unchanged boot is not reboot evidence')
    observations = re.findall(r'(?m)^EXECUTOR_CONTAINMENT (\{[^\n]+\})$', text)
    if len(observations) != 3:
        raise ValueError('all independently observed containment scenarios required')
    modes = set()
    keys = {'mode', 'singleton_reacquired', 'original_probe_live', 'original_child_live', 'cleanup_verified'}
    for raw in observations:
        observation = json.loads(raw)
        if (set(observation) != keys or any(type(observation[k]) is not bool for k in keys - {'mode'}) or
                observation['mode'] not in {'owner-death', 'stopped-guard', 'guard-first'} or observation['mode'] in modes or
                not observation['singleton_reacquired'] or not observation['cleanup_verified']):
            raise ValueError('invalid or duplicated containment observation')
        modes.add(observation['mode'])
        if observation['mode'] == 'guard-first' and not (observation['original_probe_live'] or observation['original_child_live']):
            raise ValueError('guard-first observation does not establish counterexample')
    if re.findall(r'(?m)^EXECUTOR_RECOVERY_RESULT ([^\n]+)$', text) != ['before=pass', 'after=pass']:
        raise ValueError('complete native phase results required')


if __name__ == '__main__':
    import sys
    validate(Path(sys.argv[1]).read_text())
