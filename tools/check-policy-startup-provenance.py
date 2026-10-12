#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Strict scalar transcript consumer; never establishes native acceptance."""
import argparse
from decimal import Decimal
import importlib.util
import json
import re
from pathlib import Path


def load(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


envelope = load('check-policy-startup-envelope')
facts = load('check-policy-facts-evidence')
CHILD_EXTRA = set('trace_enabled error_preserved snapshots_before_cancel trace_inventory original_contexts'.split())
PARENT_EXTRA = set('trace_extracted trace_sanitized source_isolated'.split())
CHILD = envelope.CHILD | CHILD_EXTRA
ROOT = envelope.ROOT | {'child_' + name for name in CHILD_EXTRA} | PARENT_EXTRA


FRAME_FIELDS = set('site operation phase branch allowance_ns child_present child_deadline child_deadline_ns parent_deadline parent_deadline_ns child_before child_after parent_before parent_after begin_ns end_ns constraint'.split())
PROFILE = 'cpu_count=1 child_seconds=210 parent_seconds=480 phase_minutes=18'
PREFIX = 'POLICY_STARTUP_PROVENANCE_'
PARENT = 'TestNativeMultiplePolicyStartupProvenanceUFS'


def shape(row, fields):
    if type(row) is not dict or set(row) != fields:
        raise ValueError('exact known scalar fields required')


def integer(value, low, high):
    if type(value) is not int or not low <= value <= high:
        raise ValueError('bounded scalar integer required')


def digest(value, length):
    if type(value) is not str or re.fullmatch('[a-f0-9]{' + str(length) + '}', value) is None:
        raise ValueError('exact candidate checksum required')


def pairs(items):
    result = {}
    for key, value in items:
        if key in result:
            raise ValueError('duplicate scalar key')
        result[key] = value
    return result


def nonfinite(_):
    raise ValueError('nonfinite scalar')


def scalar(raw, cap):
    if len(raw.encode('utf-8')) > cap:
        raise ValueError('bounded scalar record required')
    try:
        return json.loads(raw, object_pairs_hook=pairs, parse_constant=nonfinite)
    except (json.JSONDecodeError, RecursionError) as error:
        raise ValueError('valid bounded scalar JSON required') from error


def constraint(f):
    if f['child_before'] != f['child_after'] or f['parent_before'] != f['parent_after']:
        return 'overlap'
    if not f['child_present']:
        return 'parent-only'
    if 'canceled' in (f['child_after'], f['parent_after']):
        return 'canceled'
    if f['child_after'] == f['parent_after'] == 'deadline':
        return 'both-expired'
    if f['child_deadline'] and f['parent_deadline'] and f['child_deadline_ns'] == f['parent_deadline_ns']:
        return 'inherited'
    if f['child_after'] == 'deadline' and f['parent_after'] == 'none' and f['child_deadline'] and (not f['parent_deadline'] or f['child_deadline_ns'] < f['parent_deadline_ns']):
        return 'local-expiry'
    if f['parent_after'] == 'deadline':
        return 'parent-expired'
    return 'unresolved'


def validate_frame(f, returned):
    shape(f, FRAME_FIELDS)
    for key in ('child_present', 'child_deadline', 'parent_deadline'):
        if type(f[key]) is not bool:
            raise ValueError('exact context-presence boolean required')
    for key in ('child_deadline_ns', 'parent_deadline_ns'):
        integer(f[key], 0, 180000000000)
    integer(f['begin_ns'], 0, returned)
    integer(f['end_ns'], f['begin_ns'], returned)
    integer(f['allowance_ns'], 0, 90000000000)
    if f['operation'] != 'deadline' or type(f['branch']) is not str or len(f['branch']) > 144 or re.fullmatch(r'0(?:\.(?:[0-9]|1[01])){0,47}', f['branch']) is None:
        raise ValueError('fixed deadline operation and bounded branch required')
    scopes = {'review': (90000000000, {'inspect'}),
              'activation': (90000000000, {'first-inspection', 'second-inspection', 'publication', 'final-inspection'}),
              'backing-collector': (45000000000, {'collection'}), 'pool-collector': (45000000000, {'collection'}),
              'provider-collector': (45000000000, {'collection'}), 'system-collector': (45000000000, {'collection'}),
              'backing-command': (10000000000, {'capture'}), 'pool-command': (10000000000, {'capture'}),
              'system-command': (10000000000, {'capture'})}
    if type(f['site']) is not str or f['site'] not in scopes or type(f['phase']) is not str:
        raise ValueError('known original source site required')
    if f['child_present']:
        allowance, phases = scopes[f['site']]
        if f['allowance_ns'] != allowance or f['phase'] not in phases or not f['child_deadline']:
            raise ValueError('original child allowance and phase required')
        if f['child_deadline_ns'] > f['parent_deadline_ns'] or f['child_deadline_ns'] > f['end_ns'] + allowance:
            raise ValueError('original effective child deadline required')
    elif f['phase'] != 'guard' or f['allowance_ns'] != 0 or f['child_deadline'] or f['child_deadline_ns'] != 0 or f['child_before'] != 'absent' or f['child_after'] != 'absent':
        raise ValueError('parent-only guard cannot invent a child')
    if not f['parent_deadline']:
        raise ValueError('original bounded caller ancestry required')
    classes = {'none', 'deadline', 'canceled', 'other'}
    for context in ('child', 'parent'):
        if context == 'child' and not f['child_present']:
            continue
        before, after = f[context + '_before'], f[context + '_after']
        if type(before) is not str or type(after) is not str or before not in classes or after not in classes or before != 'none' and before != after:
            raise ValueError('known monotonic context error classes required')
        if before == 'deadline' and f[context + '_deadline_ns'] > f['begin_ns'] or after == 'deadline' and f[context + '_deadline_ns'] > f['end_ns']:
            raise ValueError('expired context must have its actual expired deadline')
    if f['constraint'] != constraint(f):
        raise ValueError('constraint must match original observations')


def validate_trace(t, summary):
    shape(t, {'status', 'frames'})
    if type(t['status']) is not str or t['status'] not in {'success', 'not-deadline', 'unattributed', 'complete', 'ambiguous', 'invalid', 'overflow'} or type(t['frames']) is not list or len(t['frames']) > 12:
        raise ValueError('bounded known trace required')
    status, frames, operation = t['status'], t['frames'], summary['startup_error']
    if operation == 'none' and status != 'success' or status == 'success' and operation != 'none' or status in {'complete', 'unattributed', 'ambiguous'} and operation != 'deadline' or status == 'not-deadline' and operation in {'none', 'deadline'}:
        raise ValueError('trace and unchanged original operation disagree')
    if status in {'success', 'not-deadline', 'unattributed'} and frames or status == 'complete' and not frames:
        raise ValueError('explicit empty success/nondeadline/unattributed trace required')
    seen = set()
    for f in frames:
        validate_frame(f, summary['return_ns'])
        key = (f['branch'], f['site'], f['phase'])
        if key in seen:
            raise ValueError('duplicate original return annotation')
        seen.add(key)
    if status != 'complete':
        return status in {'success', 'not-deadline'}
    if any(f['branch'] != '0' or f['constraint'] == 'overlap' or 'other' in (f['child_before'], f['child_after'], f['parent_before'], f['parent_after']) for f in frames):
        raise ValueError('complete trace cannot conceal unknown or competing paths')
    outer = frames[-1]
    if outer['site'] not in {'review', 'activation'} or outer['parent_deadline_ns'] != 180000000000:
        raise ValueError('complete original factory return ancestry required')
    if outer['parent_after'] == 'deadline' and summary['caller_error'] != 'deadline' or outer['parent_after'] == 'canceled' and summary['caller_error'] not in {'deadline', 'cancelled'}:
        raise ValueError('original caller error cannot regress after capture')
    allowed = {'backing-command': {'backing-collector'}, 'pool-command': {'pool-collector'},
               'system-command': {'system-collector', 'provider-collector'},
               'backing-collector': {'review', 'activation'}, 'pool-collector': {'review', 'activation'},
               'system-collector': {'review', 'activation'}, 'provider-collector': {'review', 'activation'}}
    for inner, parent in zip(frames, frames[1:]):
        if parent['site'] not in allowed.get(inner['site'], set()) or inner['end_ns'] > parent['begin_ns'] or not parent['child_present'] or inner['parent_deadline_ns'] != parent['child_deadline_ns']:
            raise ValueError('actual original inner-to-outer scope/deadline ordering required')
    return True


def validate(text, *, source_commit, compiler_sha256, artifact_sha256):
    for value, length in [(source_commit, 40), (compiler_sha256, 64), (artifact_sha256, 64)]:
        digest(value, length)
    try:
        if type(text) is not str or len(text.encode('utf-8')) > 128 << 10:
            raise ValueError('bounded sanitized parent transcript required')
    except UnicodeError as error:
        raise ValueError('valid scalar encoding required') from error
    closed = load('check-closed-startup-evidence')
    markers = {PREFIX + k for k in ('PROFILE', 'PREBUILT_SHA256', 'SOURCE', 'TRACE', 'SUMMARY', 'STACK', 'JOURNAL', 'DOMAIN_TEARDOWN', 'UFS_RESULT')} | {'POLICY_FACTS_CHILD_RESULT', 'POLICY_FACTS_PARENT_RESULT'}
    normalized = []
    rows = {}
    for line in text.splitlines():
        # Go's t.Log source prefix is permitted only on fixed control rows.
        match = re.fullmatch(r'    [A-Za-z0-9_]+_test.go:[1-9][0-9]*: ((?:POLICY_FACTS_(?:CHILD|PARENT)_RESULT|POLICY_STARTUP_PROVENANCE_UFS_RESULT) .+)', line)
        if match:
            line = match[1]
        if line in ('=== RUN   ' + PARENT, 'PASS') or re.fullmatch(r'--- PASS: ' + PARENT + r' \([0-9]+(?:\.[0-9]+)?s\)', line):
            normalized.append(line)
            continue
        marker, separator, raw = line.partition(' ')
        if not separator or marker not in markers or not raw:
            raise ValueError('only declared sanitized transcript records allowed')
        rows.setdefault(marker, []).append(raw)
        normalized.append(line)
    for marker in markers:
        if len(rows.get(marker, [])) != (3 if marker == PREFIX + 'STACK' else 1):
            raise ValueError('exact original scalar/control record inventory required')
    canonical = '\n'.join(normalized) + '\n'
    if rows[PREFIX + 'PROFILE'] != [PROFILE] or rows[PREFIX + 'PREBUILT_SHA256'] != [artifact_sha256]:
        raise ValueError('original adopted profile and expected artifact required')
    header = canonical.find('=== RUN')
    if header < 0 or any(canonical.index(PREFIX + k + ' ') > header for k in ('PROFILE', 'PREBUILT_SHA256', 'SOURCE')):
        raise ValueError('candidate binding before original parent required')
    identity = scalar(rows[PREFIX + 'SOURCE'][0], 1024)
    shape(identity, {'schema', 'profile', 'source_commit', 'compiler_sha256', 'artifact_sha256'})
    if type(identity['schema']) is not int or identity != dict(schema=1, profile='deadline-provenance-v1', source_commit=source_commit, compiler_sha256=compiler_sha256, artifact_sha256=artifact_sha256):
        raise ValueError('external expected source/compiler/artifact identity required')
    parents = closed.chunks(canonical)
    if len(parents) != 1 or parents[0][0] != PARENT:
        raise ValueError('one normally waited complete original parent required')
    body = parents[0][1]
    closed.controls(body, PREFIX + 'UFS_RESULT', ROOT)
    # New rows must occur inside the same normally completed parent.
    for k in ('TRACE', 'SUMMARY', 'STACK', 'JOURNAL', 'DOMAIN_TEARDOWN', 'UFS_RESULT'):
        if body.count(PREFIX + k + ' ') != len(rows[PREFIX + k]):
            raise ValueError('scalar evidence outside its original parent')
    record = scalar(rows[PREFIX + 'TRACE'][0], 8192)
    for kind, cap in [('SUMMARY', 8192), ('STACK', 40 << 10), ('JOURNAL', 1024)]:
        for raw in rows[PREFIX + kind]:
            scalar(raw, cap)
    # Only after strict new inventory/identity checks, project the original
    # unchanged 42 controls and scalar records into their authoritative parser.
    projected = []
    for line in normalized:
        if line.startswith(tuple(PREFIX + k + ' ' for k in ('SOURCE', 'TRACE', 'PREBUILT_SHA256'))):
            continue
        if line.startswith(PREFIX + 'UFS_RESULT '):
            line = PREFIX + 'UFS_RESULT ' + ' '.join(c + '=pass' for c in sorted(envelope.ROOT))
        projected.append(line.replace(PARENT, 'TestNativeMultiplePolicyStartupEnvelopeUFS').replace(PREFIX, 'POLICY_STARTUP_ENVELOPE_'))
    result = envelope.validate('\n'.join(projected) + '\n')
    # The older parser permits Python bool/int equality for these journal fields;
    # reject it independently here without altering its historical consumer.
    for key in ('generation_before', 'generation_after'):
        integer(result['journal'][key], 1 if key.endswith('before') else 0, (1 << 64) - 1)
    elapsed = Decimal(parents[0][3])
    if not 0 < elapsed <= 1080 or elapsed * 1000000000 < result['summary']['return_ns']:
        raise ValueError('normally completed parent must contain the original operation')
    complete = validate_trace(record, result['summary'])
    return dict(result, trace=record, identity=identity, diagnostic_complete=complete, native_acceptance=False)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('transcript', type=Path)
    parser.add_argument('--source-commit', required=True)
    parser.add_argument('--compiler-sha256', required=True)
    parser.add_argument('--artifact-sha256', required=True)
    args = parser.parse_args()
    try:
        with args.transcript.open(encoding='utf-8') as stream:
            text = stream.read((128 << 10) + 1)
        result = validate(text, source_commit=args.source_commit, compiler_sha256=args.compiler_sha256, artifact_sha256=args.artifact_sha256)
        print(json.dumps(result, separators=(',', ':')))
        if not result['diagnostic_complete']:
            raise SystemExit(1)
    except (ValueError, OSError) as error:
        parser.exit(2, 'Invalid provenance transcript: ' + str(error) + '\n')
