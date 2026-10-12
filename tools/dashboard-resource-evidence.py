#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Inactive offline receipt contract; not a runner or native attestation.

The future reviewed caller must independently verify the clean signed candidate,
compiler/artifacts/pins and owned supervisor, then supply their normalized records.
Never obtain those trusted expectations from the transcript being checked.
Historical performance validation remains a separate, unchanged requirement.
"""
import hashlib
import importlib.util
import json
import math
import re
from pathlib import Path

RELEASES = ('15.1', '14.5')
MODES = ('synthetic-100', 'synthetic-10000', 'native-small', 'projection-10000')
CASES = ('overview', 'pools-default', 'pools-maximum', 'datasets-default', 'datasets-maximum')
HASHES = ('base', 'qemu', 'firmware_code', 'firmware_vars')
PARENTS = ('TestNativePerformanceBudgets', 'TestNativeReaderResourceBudgets')
CEILING = 32 << 20
UINT64 = (1 << 64) - 1


def require(condition, message):
    if not condition:
        raise ValueError(message)


def object_keys(value, keys, context):
    require(type(value) is dict and set(value) == set(keys), context + ': missing/unknown fields')


def integer(value, minimum=0, maximum=UINT64):
    require(type(value) is int and minimum <= value <= maximum, 'invalid integer measurement')
    return value


def number(value):
    require(type(value) in (int, float) and 0 <= value <= UINT64 and math.isfinite(value), 'invalid numeric measurement')
    return value


def text(value):
    require(type(value) is str and 0 < len(value) <= 512 and not any(ord(c) < 32 for c in value), 'invalid identity text')
    return value


def digest(value, length=64):
    require(type(value) is str and re.fullmatch('[0-9a-f]{' + str(length) + '}', value) is not None, 'invalid digest')


def exact(left, right):
    """Equality that cannot mistake bool for int or float for integer counts."""
    if type(left) is not type(right):
        return False
    if type(left) is dict:
        return set(left) == set(right) and all(exact(left[k], right[k]) for k in left)
    if type(left) is list:
        return len(left) == len(right) and all(exact(a, b) for a, b in zip(left, right))
    return left == right


def load_json(payload):
    def pairs(entries):
        result = {}
        for key, value in entries:
            require(key not in result, 'duplicate JSON field')
            result[key] = value
        return result
    def constant(_value):
        raise ValueError('non-finite JSON constant')
    try:
        return json.loads(payload, object_pairs_hook=pairs, parse_constant=constant)
    except (RecursionError, OverflowError) as error:
        raise ValueError('malformed JSON record') from error


def candidate_identity(candidate):
    object_keys(candidate, ('source_commit', 'base_commit', 'pr_title', 'go_version', 'compiler_sha256',
                            'archive_sha256', 'review_verified', 'artifacts'), 'candidate')
    for key in ('source_commit', 'base_commit'):
        digest(candidate[key], 40)
    require(candidate['source_commit'] != candidate['base_commit'], 'empty candidate range')
    require(re.fullmatch(r'(?:feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(?:\([^\n]+\))?!?: .+', text(candidate['pr_title'])) is not None, 'invalid actual PR title')
    require(re.fullmatch(r'go\d+\.\d+\.\d+', text(candidate['go_version'])) is not None, 'invalid compiler version')
    for key in ('compiler_sha256', 'archive_sha256'):
        digest(candidate[key])
    require(candidate['review_verified'] is True, 'independent signed/DCO clean review required')
    object_keys(candidate['artifacts'], RELEASES, 'release artifacts')
    for artifacts in candidate['artifacts'].values():
        object_keys(artifacts, ('management', 'helper', 'fixture'), 'built artifacts')
        for value in artifacts.values():
            digest(value)


def quantiles(values):
    if not values:
        return None
    ordered = sorted(values)
    return {f'p{p}_ns': ordered[(len(values) * p + 99) // 100 - 1] for p in (50, 95, 99)}


def validate_phase(phase, report, large):
    object_keys(report, ('samples', 'summary'), 'phase')
    samples = report['samples']
    require(type(samples) is list and len(samples) == 1000, 'all scheduled observations required')
    seen = set()
    for sample in samples:
        object_keys(sample, ('index', 'case', 'outcome', 'status', 'duration_ns', 'body_bytes', 'predicate'), 'sample')
        index = integer(sample['index'], 0, 999)
        require(index not in seen and sample['case'] == CASES[index % 5], 'duplicate index or wrong case distribution')
        seen.add(index)
        integer(sample['duration_ns'], 1)
        integer(sample['body_bytes'], 1, CEILING)
        integer(sample['status'], 100, 599)
        if sample['outcome'] == 'success':
            require(sample['status'] == 200 and sample['predicate'] == 'original-inventory', 'successful original-inventory predicate missing')
        else:
            require(phase == 'replacement' and sample['case'] == 'overview' and sample['outcome'] == 'expiry'
                    and sample['status'] == 409 and sample['predicate'] == 'safe-expiry', 'unqualified denial or failed predicate')
    groups = {'all': samples}
    groups.update({case: [x for x in samples if x['case'] == case] for case in CASES})
    expected = {}
    for name, rows in groups.items():
        success = [x['duration_ns'] for x in rows if x['outcome'] == 'success']
        expiry = [x['duration_ns'] for x in rows if x['outcome'] == 'expiry']
        require(len(rows) == (1000 if name == 'all' else 200) and bool(success), 'missing successful case')
        q = quantiles(success)
        if phase == 'initial' and large:
            require(q['p95_ns'] < 250000000, 'per-case/aggregate latency budget failed')
        expected[name] = {'scheduled': len(rows), 'success': len(success), 'expiry': len(expiry),
                          'success_latency': q, 'expiry_latency': quantiles(expiry)}
    require(exact(report['summary'], expected), 'counts or nearest-rank quantiles disagree with raw observations')
    return expected


def validate_mode(row):
    object_keys(row, ('mode', 'datasets', 'owned_datasets', 'inventory_sha256', 'origin', 'procedure', 'resources', 'phases'), 'mode')
    mode = row['mode']
    require(type(mode) is str and mode in MODES, 'unknown measured mode')
    small = mode in ('synthetic-100', 'native-small')
    count = integer(row['datasets'], 1, 100000)
    if mode == 'native-small':
        require(count >= 100 and row['origin'] == 'native-original', 'original native count missing')
    else:
        require(count == (100 if small else 10000) and row['origin'] == 'generated', 'generated scope misrepresented')
    require(integer(row['owned_datasets']) == (100 if mode == 'native-small' else 0), 'owned fixture mismatch')
    digest(row['inventory_sha256'])
    procedure = row['procedure']
    expected = {'clients': 10, 'session_count': 1, 'requests': 1000, 'replacement_requests': 1000,
                'replacements': 3, 'retries': 0, 'hidden_api_requests': 0, 'warmup_seconds': 60,
                'idle_seconds': 300 if small else 0, 'scheduled_collect_seconds': 30,
                'default_limit': 100, 'maximum_limit': 500, 'comparison_rows': 20,
                'response_ceiling_bytes': CEILING, 'predicate_controller_outside_pair': True}
    object_keys(procedure, list(expected) + ['warmup_wall_ns'], 'procedure')
    require(exact({k: procedure[k] for k in expected}, expected), 'procedure or comparison bounds changed')
    integer(procedure['warmup_wall_ns'], 60000000000)
    resources = row['resources']
    object_keys(resources, ('sampled_combined_peak_rss_bytes', 'conservative_combined_high_water_rss_bytes',
                            'budget_peak_rss_bytes', 'management_high_water_bytes', 'helper_high_water_bytes',
                            'rss_samples', 'max_sample_gap_ns', 'nominal_sample_ns',
                            'idle_user_ns', 'idle_system_ns', 'idle_wall_ns', 'idle_cpu_percent_one_core'), 'resources')
    sampled = integer(resources['sampled_combined_peak_rss_bytes'], 1)
    high = integer(resources['conservative_combined_high_water_rss_bytes'], 1)
    peak = integer(resources['budget_peak_rss_bytes'], 1)
    require(high == integer(resources['management_high_water_bytes'], 1) + integer(resources['helper_high_water_bytes'], 1), 'owned high-water sum mismatch')
    require(peak == max(sampled, high) and peak <= (128 if small else 256) << 20, 'paired RSS accounting/budget failed')
    integer(resources['rss_samples'], 1)
    integer(resources['max_sample_gap_ns'], 1)
    require(integer(resources['nominal_sample_ns']) == 100000000, 'sample interval changed')
    user = integer(resources['idle_user_ns'])
    system = integer(resources['idle_system_ns'])
    wall = integer(resources['idle_wall_ns'], 300000000000 if small else 0)
    cpu = number(resources['idle_cpu_percent_one_core'])
    if small:
        require(math.isclose(cpu, (user + system) * 100 / wall, rel_tol=1e-12, abs_tol=1e-12) and cpu < 1, 'idle accounting/budget failed')
    else:
        require(user == system == wall == cpu == 0, 'large idle measurement misrepresented')
    object_keys(row['phases'], ('initial', 'replacement'), 'phases')
    return {phase: validate_phase(phase, row['phases'][phase], not small) for phase in ('initial', 'replacement')}


def records(log):
    result = {name: [] for name in ('DASHBOARD_PROVENANCE', 'DASHBOARD_RESOURCE', 'DASHBOARD_CONTROLS',
                                   'PERFORMANCE_RESULT', 'NATIVE_READER_RESULT')}
    for line in log.splitlines():
        if 'DASHBOARD_' in line:
            match = re.fullmatch(r'\s*(?:\S+\.go:\d+: )?(DASHBOARD_[A-Z_]+) (.+)', line)
            require(match is not None and match[1] in result, 'malformed/unknown dashboard coverage record')
            result[match[1]].append(load_json(match[2]))
        else:
            for marker in ('PERFORMANCE_RESULT', 'NATIVE_READER_RESULT'):
                if marker + ' ' in line:
                    result[marker].append(load_json(line.split(marker + ' ', 1)[1]))
    return result


def preserved_controls(log, rows, summaries):
    require(not re.search(r'(?:--- (?:SKIP|FAIL):|^FAIL(?:\s|$))', log, re.MULTILINE), 'failed/skipped native suite')
    run_positions, pass_positions = [], []
    for parent in PARENTS:
        starts = list(re.finditer(r'^=== RUN\s+' + parent + r'\s*$', log, re.MULTILINE))
        ends = list(re.finditer(r'^--- PASS: ' + parent + r' \([0-9.]+s\)$', log, re.MULTILINE))
        require(len(starts) == len(ends) == 1 and starts[0].start() < ends[0].start(), 'native parent execution/pass missing, duplicate or out of order')
        run_positions.append(starts[0].start())
        pass_positions.append(ends[0].end())
    # The existing performance runner first executes discovery suites. Retain
    # their output; their PASS cannot complete the resource suite.
    completions = [m for m in re.finditer(r'^PASS$', log, re.MULTILINE) if m.start() >= min(run_positions)]
    require(len(completions) == 1 and completions[0].start() > max(pass_positions), 'complete native suite pass missing/duplicate/out of order')
    for marker in ('PERFORMANCE_LIMITS ', 'NATIVE_READER_CRASH '):
        require(sum(marker in line for line in log.splitlines()) == 1, 'native control missing/duplicate')
    # Preserve its behavior and old records; add strict new coverage alongside it.
    spec = importlib.util.spec_from_file_location('legacy_performance_evidence', Path(__file__).with_name('check-performance-evidence.py'))
    legacy = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(legacy)
    try:
        legacy.validate(log)
    except (KeyError, TypeError, OverflowError) as error:
        raise ValueError('malformed legacy measurement') from error
    require(len(rows['PERFORMANCE_RESULT']) == 2 and len(rows['NATIVE_READER_RESULT']) == 2, 'legacy measured modes incomplete')
    measured = {row['mode']: row for row in rows['DASHBOARD_RESOURCE']}
    for old in rows['PERFORMANCE_RESULT'] + rows['NATIVE_READER_RESULT']:
        mode = old.get('mode', 'synthetic-' + str(old['datasets']))
        require(type(mode) is str and mode in measured, 'legacy mode mismatch')
        new = measured[mode]
        for key in ('datasets', 'owned_datasets'):
            if key in old:
                require(exact(old[key], new[key]), 'legacy source count mismatch')
        for key in ('clients', 'requests', 'replacement_requests', 'replacements', 'warmup_seconds', 'idle_seconds'):
            require(exact(old[key], new['procedure'][key]), 'legacy procedure mismatch')
        for key in ('sampled_combined_peak_rss_bytes', 'conservative_combined_high_water_rss_bytes', 'budget_peak_rss_bytes', 'rss_samples'):
            require(exact(old[key], new['resources'][key]), 'legacy resource accounting mismatch')
        require(number(old['idle_cpu_percent_one_core']) == new['resources']['idle_cpu_percent_one_core'], 'legacy idle mismatch')
        q = summaries[mode]['initial']['all']['success_latency']
        for p in (50, 95, 99):
            # Existing reports truncate monotonic durations to microseconds.
            require(number(old[f'p{p}_ms']) == (q[f'p{p}_ns'] // 1000) / 1000, 'legacy percentile mismatch')


def validate_run(candidate, run):
    object_keys(run, ('release', 'pins', 'fixtures', 'record', 'transcript', 'supervisor'), 'run')
    object_keys(run['fixtures'], MODES, 'independent original fixtures')
    release = run['release']
    require(type(release) is str and release in RELEASES, 'unknown release')
    payload = run['transcript']
    require(type(payload) is bytes and 0 < len(payload) <= CEILING, 'absent/oversized transcript')
    try:
        log = payload.decode('utf-8')
    except UnicodeError as error:
        raise ValueError('invalid transcript encoding') from error
    record = run['record']
    cleanup = ('guest_assertions', 'host_teardown', 'fixture_key_removed', 'image_removed',
               'variables_removed', 'listener_removed', 'qemu_exited', 'qualified')
    object_keys(record, ('release', 'candidate', 'transcript_sha256', 'provenance', 'preservation') + cleanup, 'host record')
    require(record['release'] == release and exact(record['candidate'], candidate), 'candidate/release mismatch')
    require(record['transcript_sha256'] == hashlib.sha256(payload).hexdigest(), 'transcript hash mismatch')
    require(all(record[key] is True for key in cleanup), 'guest/host cleanup unverified')
    object_keys(run['supervisor'], ('exit_code', 'group_cleaned', 'stopped_verified', 'source_commit', 'release', 'transcript_sha256'), 'supervisor')
    require(exact(run['supervisor'], {'exit_code': 0, 'group_cleaned': True, 'stopped_verified': True,
                                      'source_commit': candidate['source_commit'], 'release': release,
                                      'transcript_sha256': record['transcript_sha256']}), 'owned group not stopped/verified')
    object_keys(run['pins'], HASHES, 'expected independent pins')
    object_keys(record['preservation'], HASHES, 'preservation')
    for key, pin in run['pins'].items():
        digest(pin)
        require(exact(record['preservation'][key], {'before': pin, 'after': pin}), 'base/runtime preservation mismatch')
    rows = records(log)
    require(len(rows['DASHBOARD_PROVENANCE']) == len(rows['DASHBOARD_CONTROLS']) == 1, 'missing/duplicate provenance or controls')
    provenance = rows['DASHBOARD_PROVENANCE'][0]
    object_keys(provenance, ('candidate', 'release', 'artifacts', 'kernel', 'userland', 'zfs', 'guest_go', 'host_os',
                             'host_arch', 'qemu_version', 'arch', 'cpu_model', 'cpu_count', 'memory_mib',
                             'accelerator', 'physical_bytes'), 'provenance')
    require(exact(record['provenance'], provenance) and exact(provenance['candidate'], candidate)
            and exact(provenance['artifacts'], candidate['artifacts'][release]), 'source/compiler/artifact provenance mismatch')
    expected = {'release': release, 'guest_go': candidate['go_version'], 'arch': 'amd64', 'cpu_model': 'qemu64',
                'cpu_count': 2, 'memory_mib': 6144, 'accelerator': 'tcg'}
    require(exact({k: provenance[k] for k in expected}, expected), 'release/compiler/VM procedure mismatch')
    for key in ('kernel', 'userland'):
        require(re.fullmatch(re.escape(release) + r'-RELEASE(?:-p\d+)?', text(provenance[key])) is not None, 'actual guest version mismatch')
    for key in ('zfs', 'host_os', 'host_arch', 'qemu_version'):
        text(provenance[key])
    integer(provenance['physical_bytes'], 1)
    require(len(rows['DASHBOARD_RESOURCE']) == 4, 'four measured modes required')
    summaries = {}
    for row in rows['DASHBOARD_RESOURCE']:
        result = validate_mode(row)
        original = run['fixtures'][row['mode']]
        keys = ('datasets', 'owned_datasets', 'inventory_sha256', 'origin')
        object_keys(original, keys, 'independent fixture observation')
        require(exact(original, {k: row[k] for k in keys}), 'original fixture count/identity digest mismatch')
        require(row['mode'] not in summaries, 'duplicate measured mode')
        summaries[row['mode']] = result
    require(set(summaries) == set(MODES), 'missing measured mode')
    controls = rows['DASHBOARD_CONTROLS'][0]
    expected = {'predicate_assertions': 8000, 'native_owned_count': 100, 'native_guid_assertions': 100,
                'native_boot_state': 'unchanged', 'boundary_outside_normal_budget': True, 'guest_cleanup': True,
                'pair_descendants_removed': True, 'sockets_removed': True}
    boundary = ('boundary_sampled_rss_bytes', 'boundary_high_water_bytes', 'boundary_samples', 'boundary_max_sample_gap_ns')
    object_keys(controls, list(expected) + list(boundary), 'native assertion controls')
    require(exact({k: controls[k] for k in expected}, expected), 'native predicate/crash/cleanup controls incomplete')
    for key in boundary:
        integer(controls[key], 1)
    limits = next(line for line in log.splitlines() if 'PERFORMANCE_LIMITS ' in line) if any('PERFORMANCE_LIMITS ' in line for line in log.splitlines()) else ''
    suffix = re.search(r' sampled_rss_bytes=(\d+) high_water_sum_bytes=(\d+) samples=(\d+) max_gap_ms=(\d+(?:\.\d+)?)$', limits)
    require(suffix is not None, 'original boundary measurements missing')
    require([int(suffix[i]) for i in (1, 2, 3)] == [controls[k] for k in boundary[:3]]
            and float(suffix[4]) == float(format((controls['boundary_max_sample_gap_ns'] // 1000) / 1000, '.1f')), 'original boundary measurements mismatch')
    preserved_controls(log, rows, summaries)
    return sum(result['replacement']['all']['expiry'] for result in summaries.values())


def validate(candidate, runs):
    """Validate all input without modification. Return counts, never native acceptance.

    ``candidate`` and each run's ``pins``/``fixtures``/``record``/``supervisor`` are independent
    caller inputs. Their flags are checked, not cryptographically authenticated.
    Transcript records alone cannot satisfy this contract.
    """
    candidate_identity(candidate)
    require(type(runs) is list and len(runs) == 2, 'both pinned release receipts required')
    seen, expiry = set(), 0
    for run in runs:
        expiry += validate_run(candidate, run)
        require(run['release'] not in seen, 'duplicate release')
        seen.add(run['release'])
    require(seen == set(RELEASES), 'missing release')
    return {'releases': list(RELEASES), 'modes': 8, 'scheduled': 16000, 'expiry': expiry}
