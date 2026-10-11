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
for relation in ('same-pool', 'other-pool', 'mixed'):
    STORAGE.update('history-' + relation + '_' + result for result in ('catalog', 'leases-held', 'denied', 'preserved', 'holds', 'released', 'reopened', 'retry'))
STORAGE.update('multi_' + result for result in ('catalog', 'reviewed', 'activated', 'observed', 'closed', 'lineage', 'ledger', 'all-policies', 'leases', 'shared-deadline', 'retired', 'reopened'))
for stage in ('review', 'activate', 'observe', 'finish', 'post-activate', 'post-finish'):
    STORAGE.update('last-' + stage + '_' + result for result in ('denied', 'last-policy', 'leases', 'deadline', 'preserved', 'sticky', 'released', 'diagnostic'))
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

OWNER_GROUPS = {
    group: {c for c in OWNER if c in ('data_sentinel_preserved', 'domain_teardown', 'mounts_restored') or any(c.startswith(mode + '_') for mode in modes)}
    for group, modes in [('Lifecycle', ('clean', 'clean-again')), ('Revocation', ('late', 'uid')), ('Denial', ('legacy', 'member', 'scope', 'missing-pool'))]
}


def validate(text, scope):
    original = load('check-closed-startup-evidence')
    found = original.chunks(text)
    if scope not in ('publication', 'owner'):
        raise ValueError('unknown policy scope')
    if scope == 'owner' and re.findall(r'(?m)^POLICY_STARTUP_OWNER_PHASE ([^\n]+)$', text) != ['policy begin timeout=34m', 'policy done', 'retained begin timeout=34m', 'retained done']:
        raise ValueError('two normally completed bounded owner phases required')
    wanted = ({'TestNativePolicyStartup' + group + 'Owner' + kind for group in OWNER_GROUPS for kind in ('UFS', 'ZFS')}
              if scope == 'owner' else {'TestNativePolicyStartup' + kind for kind in ('UFS', 'ZFS')})
    if scope == 'owner':
        policy_body = text.split('POLICY_STARTUP_OWNER_PHASE policy begin timeout=34m\n', 1)[1].split('POLICY_STARTUP_OWNER_PHASE policy done\n', 1)[0]
        phase_parents = original.chunks(policy_body)
        if len(phase_parents) != len(wanted) or {name for name, _, _, _ in phase_parents} != wanted:
            raise ValueError('policy parents must complete within their own bounded phase')
        retained_body = text.split('POLICY_STARTUP_OWNER_PHASE retained begin timeout=34m\n', 1)[1].split('POLICY_STARTUP_OWNER_PHASE retained done\n', 1)[0]
        original.validate(retained_body, 'owner')
    new = [(n, b, e, sec) for n, b, e, sec in found if n in wanted]
    if len(new) != len(wanted) or {n for n, _, _, _ in new} != wanted:
        raise ValueError('all distinct actual policy parents required')
    retained = ''.join('=== RUN   ' + n + '\n' + b + '--- PASS: ' + e + ' (' + sec + 's)\n' for n, b, e, sec in found if n not in wanted)
    original.validate(retained, scope)
    for name, body, _, _ in new:
        kind = 'ufs' if name.endswith('UFS') else 'zfs'
        group = next((g for g in OWNER_GROUPS if name.startswith('TestNativePolicyStartup' + g + 'Owner')), '')
        marker = 'POLICY_STARTUP_' + (group.upper() + '_OWNER_' if scope == 'owner' else 'STORAGE_') + kind.upper() + '_RESULT'
        original.controls(body, marker, OWNER_GROUPS[group] if scope == 'owner' else STORAGE)
        if scope == 'publication':
            if re.findall(r'STATE_BACKING_RESULT ([^\n]+)', body) != [name + ' filesystem=' + kind + ' softupdates=off teardown=pass']:
                raise ValueError('actual policy backing teardown required')
        else:
            for marker, value in [('DOMAIN', 'filesystems=pass directories=pass'), ('DATA', 'restored=pass')]:
                if re.findall('POLICY_STARTUP_' + marker + r'_TEARDOWN ([^\n]+)', body) != [group.lower() + ' ' + kind + ' ' + value]:
                    raise ValueError('independent policy fixture teardown required')
    for marker in (['POLICY_STARTUP_(?:STORAGE_UFS|STORAGE_ZFS)_RESULT '] if scope == 'publication' else ['POLICY_STARTUP_(?:LIFECYCLE|REVOCATION|DENIAL)_OWNER_(?:UFS|ZFS)_RESULT ', 'POLICY_STARTUP_DOMAIN_TEARDOWN ', 'POLICY_STARTUP_DATA_TEARDOWN ']):
        if len(re.findall(marker, text)) != len(wanted):
            raise ValueError('extra/missing policy marker')


if __name__ == '__main__':
    validate(Path(sys.argv[1]).read_text(), sys.argv[2])
