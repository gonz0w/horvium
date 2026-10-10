#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Policy startup evidence requires its new controls AND all retained gates."""
from pathlib import Path
import importlib.util
import re
import sys


def load(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


STORAGE = set('policy_bound leases_held original_observation no_old_release original_close exact_closed exact_ledger finish_retired original_release ordinary_metadata_only ordinary_activation_denied'.split())
for mode in ('collector', 'alias', 'cancel', 'legacy', 'scope', 'member', 'missing-system', 'context-deadline'):
    STORAGE.update(mode + '_' + result for result in ('denied', 'sticky', 'preserved', 'released'))
for stage in ('activate', 'finish'):
    for point in ('syncFile', 'rename', 'syncDirectory', 'post-policy', 'post-directory'):
        STORAGE.update(stage + '-' + point + '_' + result for result in ('denied', 'retired', 'retry_preserved', 'released'))
for format in ('legacy', 'property'):
    for state in ('queued', 'applying', 'observing', 'succeeded', 'failed', 'cancelled', 'reconcile-required'):
        STORAGE.update(format + '-' + state + '_' + result for result in ('denied', 'preserved', 'holds', 'released'))
OWNER = set('data_sentinel_preserved domain_teardown mounts_restored'.split())
for mode in ('clean', 'clean-again', 'legacy', 'member', 'scope', 'missing-pool', 'late', 'uid'):
    if mode in ('clean', 'clean-again'):
        child = 'registered leases-held original-observation copied-denied decoded-denied closed retired released registration-consumed'
    elif mode == 'uid':
        child = 'registered actual-nonroot inherited-root operations-denied active-preserved released'
    elif mode == 'late':
        child = 'denied active-preserved released registration-consumed'
    else:
        child = 'denied preserved released registration-consumed'
    OWNER.update(mode + '_' + result for result in child.split() + ['child_waited', 'inventory', 'parent_leases_released', 'parent_history', 'exact_policy'])


def validate(text, scope):
    original = load('check-closed-startup-evidence')
    found = original.chunks(text)
    if scope not in ('publication', 'owner'):
        raise ValueError('unknown policy scope')
    prefix = 'TestNativePolicyStartup' + ('Owner' if scope == 'owner' else '')
    wanted = {prefix + kind for kind in ('UFS', 'ZFS')}
    new = [(n, b, e, sec) for n, b, e, sec in found if n in wanted]
    if len(new) != 2 or {n for n, _, _, _ in new} != wanted:
        raise ValueError('two distinct actual policy parents required')
    retained = ''.join('=== RUN   ' + n + '\n' + b + '--- PASS: ' + e + ' (' + sec + 's)\n' for n, b, e, sec in found if n not in wanted)
    original.validate(retained, scope)
    for name, body, _, _ in new:
        kind = 'ufs' if name.endswith('UFS') else 'zfs'
        marker = 'POLICY_STARTUP_' + ('OWNER_' if scope == 'owner' else 'STORAGE_') + kind.upper() + '_RESULT'
        original.controls(body, marker, OWNER if scope == 'owner' else STORAGE)
        if scope == 'publication':
            if re.findall(r'STATE_BACKING_RESULT ([^\n]+)', body) != [name + ' filesystem=' + kind + ' softupdates=off teardown=pass']:
                raise ValueError('actual policy backing teardown required')
        else:
            for marker, value in [('DOMAIN', 'filesystems=pass directories=pass'), ('DATA', 'restored=pass')]:
                if re.findall('POLICY_STARTUP_' + marker + r'_TEARDOWN ([^\n]+)', body) != [kind + ' ' + value]:
                    raise ValueError('independent policy fixture teardown required')
    for marker in (['POLICY_STARTUP_(?:STORAGE_UFS|STORAGE_ZFS)_RESULT '] if scope == 'publication' else ['POLICY_STARTUP_OWNER_(?:UFS|ZFS)_RESULT ', 'POLICY_STARTUP_DOMAIN_TEARDOWN ', 'POLICY_STARTUP_DATA_TEARDOWN ']):
        if len(re.findall(marker, text)) != 2:
            raise ValueError('extra/missing policy marker')


if __name__ == '__main__':
    validate(Path(sys.argv[1]).read_text(), sys.argv[2])
