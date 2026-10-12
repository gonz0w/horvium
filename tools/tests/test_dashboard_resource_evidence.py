# SPDX-License-Identifier: BSD-3-Clause
"""Synthetic transcript tests only: none of these records is native evidence."""
import copy
import hashlib
import importlib.util
import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
CASES = ('overview', 'pools-default', 'pools-maximum', 'datasets-default', 'datasets-maximum')
MODES = ('synthetic-100', 'synthetic-10000', 'native-small', 'projection-10000')
RELEASES = ('15.1', '14.5')


def digest(value):
    return hashlib.sha256(value).hexdigest()


def quantiles(values):
    if not values:
        return None
    ordered = sorted(values)
    return {f'p{p}_ns': ordered[(len(values) * p + 99) // 100 - 1] for p in (50, 95, 99)}


def summary(samples):
    groups = {'all': samples}
    groups.update({case: [x for x in samples if x['case'] == case] for case in CASES})
    return {name: {'scheduled': len(rows), 'success': sum(x['outcome'] == 'success' for x in rows),
                   'expiry': sum(x['outcome'] == 'expiry' for x in rows),
                   'success_latency': quantiles([x['duration_ns'] for x in rows if x['outcome'] == 'success']),
                   'expiry_latency': quantiles([x['duration_ns'] for x in rows if x['outcome'] == 'expiry'])}
            for name, rows in groups.items()}


def fixture():
    candidate = {'source_commit': '1' * 40, 'base_commit': '2' * 40,
                 'pr_title': 'test(dashboards): wire current resource workload',
                 'go_version': 'go1.27.1', 'compiler_sha256': '3' * 64,
                 'archive_sha256': '4' * 64, 'review_verified': True,
                 'artifacts': {r: {'management': '5' * 64, 'helper': '6' * 64,
                                   'fixture': '7' * 64} for r in RELEASES}}
    runs = []
    for release in RELEASES:
        pins = {name: digest((release + name).encode()) for name in ('base', 'qemu', 'firmware_code', 'firmware_vars')}
        provenance = {'candidate': candidate, 'release': release, 'artifacts': candidate['artifacts'][release],
                      'kernel': release + '-RELEASE', 'userland': release + '-RELEASE',
                      'zfs': 'fixture-only', 'guest_go': 'go1.27.1', 'host_os': 'Darwin',
                      'host_arch': 'arm64', 'qemu_version': 'fixture-only', 'arch': 'amd64',
                      'cpu_model': 'qemu64', 'cpu_count': 2, 'memory_mib': 6144,
                      'accelerator': 'tcg', 'physical_bytes': 6387654656}
        rows, legacy = [], []
        for mode in MODES:
            small = mode in ('synthetic-100', 'native-small')
            count = 103 if mode == 'native-small' else (100 if small else 10000)
            resources = {'sampled_combined_peak_rss_bytes': 42 << 20,
                         'conservative_combined_high_water_rss_bytes': 40 << 20,
                         'management_high_water_bytes': 20 << 20, 'helper_high_water_bytes': 20 << 20,
                         'budget_peak_rss_bytes': 42 << 20, 'rss_samples': 600,
                         'max_sample_gap_ns': 1210000000, 'nominal_sample_ns': 100000000,
                         'idle_user_ns': 200000000 if small else 0,
                         'idle_system_ns': 100000000 if small else 0,
                         'idle_wall_ns': 300000000000 if small else 0,
                         'idle_cpu_percent_one_core': 0.1 if small else 0}
            phases = {}
            for phase in ('initial', 'replacement'):
                samples = [{'index': i, 'case': CASES[i % 5], 'outcome': 'success', 'status': 200,
                            'duration_ns': (10 + i % 20) * 1000000, 'body_bytes': 1200,
                            'predicate': 'original-inventory'} for i in range(1000)]
                if phase == 'replacement':
                    samples[0].update(outcome='expiry', status=409, duration_ns=1000000,
                                      predicate='safe-expiry')
                phases[phase] = {'samples': samples, 'summary': summary(samples)}
            procedure = {'clients': 10, 'session_count': 1, 'requests': 1000,
                         'replacement_requests': 1000, 'replacements': 3, 'retries': 0,
                         'hidden_api_requests': 0, 'warmup_seconds': 60,
                         'warmup_wall_ns': 60000000000, 'idle_seconds': 300 if small else 0,
                         'scheduled_collect_seconds': 30, 'default_limit': 100, 'maximum_limit': 500,
                         'comparison_rows': 20, 'response_ceiling_bytes': 32 << 20,
                         'predicate_controller_outside_pair': True}
            row = {'mode': mode, 'datasets': count, 'owned_datasets': 100 if mode == 'native-small' else 0,
                   'inventory_sha256': digest(mode.encode()), 'origin': 'native-original' if mode == 'native-small' else 'generated',
                   'procedure': procedure, 'resources': resources, 'phases': phases}
            rows.append(row)
            old = {key: procedure[key] for key in ('clients', 'requests', 'replacement_requests', 'replacements', 'warmup_seconds', 'idle_seconds')}
            old.update({key: resources[key] for key in ('sampled_combined_peak_rss_bytes', 'conservative_combined_high_water_rss_bytes', 'budget_peak_rss_bytes', 'rss_samples', 'idle_cpu_percent_one_core')})
            old.update(datasets=count, p50_ms=19, p95_ms=28, p99_ms=29)
            if mode.startswith('synthetic'):
                legacy.append('PERFORMANCE_RESULT ' + json.dumps(old))
            else:
                old.update(mode=mode, owned_datasets=row['owned_datasets'])
                legacy.append('NATIVE_READER_RESULT ' + json.dumps(old))
        controls = {'predicate_assertions': 8000, 'native_owned_count': 100,
                    'native_guid_assertions': 100, 'native_boot_state': 'unchanged',
                    'boundary_sampled_rss_bytes': 360 << 20, 'boundary_high_water_bytes': 350 << 20,
                    'boundary_samples': 100, 'boundary_max_sample_gap_ns': 1300000000,
                    'boundary_outside_normal_budget': True, 'guest_cleanup': True,
                    'pair_descendants_removed': True, 'sockets_removed': True}
        lines = ['DASHBOARD_PROVENANCE ' + json.dumps(provenance)]
        lines += ['DASHBOARD_RESOURCE ' + json.dumps(row) for row in rows]
        lines += ['DASHBOARD_CONTROLS ' + json.dumps(controls)] + legacy
        lines += ['PERFORMANCE_LIMITS accepted_model_records=100000 record_exhaustion=pass byte_exhaustion=pass recovery=pass sampled_rss_bytes=377487360 high_water_sum_bytes=367001600 samples=100 max_gap_ms=1300.0',
                  'NATIVE_READER_CRASH captured_native_command=pass subtree_exit=pass stale_retention=pass recovery=pass boot_state=unchanged']
        for name in ('TestNativePerformanceBudgets', 'TestNativeReaderResourceBudgets'):
            lines += ['=== RUN   ' + name, '--- PASS: ' + name + ' (430.00s)']
        lines.append('PASS')
        record = {'release': release, 'candidate': candidate, 'transcript_sha256': '',
                  'provenance': provenance, 'preservation': {k: {'before': v, 'after': v} for k, v in pins.items()},
                  'guest_assertions': True, 'host_teardown': True, 'fixture_key_removed': True,
                  'image_removed': True, 'variables_removed': True, 'listener_removed': True,
                  'qemu_exited': True, 'qualified': True}
        runs.append({'release': release, 'pins': pins, 'fixtures': {row['mode']: {k: row[k] for k in ('datasets', 'owned_datasets', 'inventory_sha256', 'origin')} for row in rows}, 'record': record, 'transcript': '\n'.join(lines).encode(),
                     'supervisor': {'exit_code': 0, 'group_cleaned': True, 'stopped_verified': True,
                                    'source_commit': candidate['source_commit'], 'release': release, 'transcript_sha256': ''}})
    bind(runs)
    return candidate, runs


def bind(runs):
    for run in runs:
        run['record']['transcript_sha256'] = digest(run['transcript'])
        run['supervisor']['transcript_sha256'] = digest(run['transcript'])


def change_record(runs, marker, modify, occurrence=0):
    lines = runs[0]['transcript'].decode().splitlines()
    indices = [i for i, line in enumerate(lines) if line.startswith(marker + ' ')]
    i = indices[occurrence]
    row = json.loads(lines[i].split(' ', 1)[1])
    modify(row)
    lines[i] = marker + ' ' + json.dumps(row)
    runs[0]['transcript'] = '\n'.join(lines).encode()
    bind(runs)


class DashboardResourceEvidenceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        spec = importlib.util.spec_from_file_location('dashboard_receipts', ROOT / 'tools/dashboard-resource-evidence.py')
        cls.module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(cls.module)

    def reject(self, candidate, runs):
        with self.assertRaises(ValueError):
            self.module.validate(candidate, runs)

    def test_complete_two_release_receipt_is_offline_validation_only(self):
        candidate, runs = fixture()
        before = copy.deepcopy((candidate, runs))
        result = self.module.validate(candidate, runs)
        self.assertEqual(result, {'releases': ['15.1', '14.5'], 'modes': 8, 'scheduled': 16000, 'expiry': 8})
        self.assertEqual((candidate, runs), before)
        # No real expiry timing race is required; deterministic eviction is a separate fixture.
        for run in runs:
            lines = run['transcript'].decode().splitlines()
            for i, line in enumerate(lines):
                if line.startswith('DASHBOARD_RESOURCE '):
                    row = json.loads(line.split(' ', 1)[1])
                    p = row['phases']['replacement']
                    p['samples'][0].update(outcome='success', status=200, predicate='original-inventory')
                    p['summary'] = summary(p['samples'])
                    lines[i] = 'DASHBOARD_RESOURCE ' + json.dumps(row)
            run['transcript'] = '\n'.join(lines).encode()
        bind(runs)
        self.assertEqual(self.module.validate(candidate, runs)['expiry'], 0)

    def test_missing_duplicate_unknown_releases_modes_records_and_execution(self):
        candidate, original = fixture()
        for runs in (original[:1], original + [original[0]], [original[0], dict(original[1], release='13.5')]):
            self.reject(candidate, runs)
        for marker in ('DASHBOARD_PROVENANCE', 'DASHBOARD_RESOURCE', 'DASHBOARD_CONTROLS', 'PERFORMANCE_RESULT',
                       'NATIVE_READER_RESULT', 'PERFORMANCE_LIMITS', 'NATIVE_READER_CRASH',
                       '=== RUN   TestNativeReaderResourceBudgets', '--- PASS: TestNativePerformanceBudgets', 'PASS'):
            for action in ('remove', 'duplicate'):
                with self.subTest(marker=marker, action=action):
                    runs = copy.deepcopy(original)
                    lines = runs[0]['transcript'].decode().splitlines()
                    matches = [x for x in lines if x == marker or x.startswith(marker + ' ')]
                    self.assertTrue(matches)
                    if action == 'remove':
                        lines.remove(matches[0])
                    else:
                        lines.append(matches[0])
                    runs[0]['transcript'] = '\n'.join(lines).encode()
                    bind(runs)
                    self.reject(candidate, runs)
        for extra in ('DASHBOARD_FUTURE {}', '--- SKIP: TestAny (0s)', '--- FAIL: TestAny (0s)', 'FAIL', 'VM booted'):
            runs = copy.deepcopy(original)
            runs[0]['transcript'] = (extra if extra == 'VM booted' else runs[0]['transcript'].decode() + '\n' + extra).encode()
            bind(runs)
            self.reject(candidate, runs)

    def test_procedure_counts_bounds_origins_and_resource_accounting(self):
        candidate, original = fixture()
        mutations = [(('procedure', k), v) for k, v in
                     [('clients', 9), ('session_count', 2), ('requests', 999), ('replacement_requests', 999),
                      ('replacements', 2), ('retries', 1), ('hidden_api_requests', 1), ('warmup_seconds', 59),
                      ('warmup_wall_ns', 59999999999), ('idle_seconds', 299), ('scheduled_collect_seconds', 29),
                      ('default_limit', 99), ('maximum_limit', 501), ('comparison_rows', 21),
                      ('response_ceiling_bytes', (32 << 20) + 1), ('predicate_controller_outside_pair', False)]]
        mutations += [(('resources', k), v) for k, v in
                      [('sampled_combined_peak_rss_bytes', 0), ('budget_peak_rss_bytes', 40 << 20),
                       ('management_high_water_bytes', 1), ('helper_high_water_bytes', 0),
                       ('rss_samples', 0), ('nominal_sample_ns', 1), ('max_sample_gap_ns', 0),
                       ('idle_wall_ns', 299999999999), ('idle_cpu_percent_one_core', 0.2),
                       ('idle_user_ns', -1), ('idle_system_ns', True)]]
        mutations += [(('datasets',), 101), (('owned_datasets',), 100), (('origin',), 'native-original'),
                      (('inventory_sha256',), 'bad')]
        for path, value in mutations:
            with self.subTest(path=path, value=value):
                runs = copy.deepcopy(original)
                def mutate(row):
                    target = row
                    for key in path[:-1]:
                        target = target[key]
                    target[path[-1]] = value
                change_record(runs, 'DASHBOARD_RESOURCE', mutate)
                self.reject(candidate, runs)
        for value in (True, '600', -1, float('nan'), float('inf'), 1.5):
            runs = copy.deepcopy(original)
            change_record(runs, 'DASHBOARD_RESOURCE', lambda r: r['resources'].update(rss_samples=value))
            self.reject(candidate, runs)

    def test_raw_sample_identity_denials_predicates_and_summaries(self):
        candidate, original = fixture()
        for phase in ('initial', 'replacement'):
            for key, value in [('index', 1), ('case', 'wrong'), ('duration_ns', 0), ('duration_ns', True),
                               ('duration_ns', '10'), ('duration_ns', 1.5), ('body_bytes', 0),
                               ('body_bytes', (32 << 20) + 1), ('predicate', 'failed'),
                               ('predicate', 'original-inventory'), ('status', 403), ('outcome', 'failure')]:
                if phase == 'initial' and key == 'predicate' and value == 'original-inventory':
                    continue
                with self.subTest(phase=phase, key=key, value=value):
                    runs = copy.deepcopy(original)
                    change_record(runs, 'DASHBOARD_RESOURCE', lambda r: r['phases'][phase]['samples'][0].update({key: value}))
                    self.reject(candidate, runs)
        for mutate in (lambda p: p.pop('summary'), lambda p: p['summary'].pop('pools-default'), lambda p: p['samples'].pop(), lambda p: p['samples'].append(p['samples'][0]),
                       lambda p: p['summary']['all'].update(success=999),
                       lambda p: p['summary']['overview']['success_latency'].update(p95_ns=1),
                       lambda p: p.update(unknown=True)):
            runs = copy.deepcopy(original)
            change_record(runs, 'DASHBOARD_RESOURCE', lambda r: mutate(r['phases']['initial']))
            self.reject(candidate, runs)
        # Recompute summaries to prove a well-formed denial cannot hide a forbidden case/phase.
        for phase, case in [('initial', 'overview'), ('replacement', 'pools-default')]:
            runs = copy.deepcopy(original)
            def denied(row):
                p = row['phases'][phase]
                sample = next(x for x in p['samples'] if x['case'] == case)
                sample.update(outcome='expiry', status=409, predicate='safe-expiry')
                p['summary'] = summary(p['samples'])
            change_record(runs, 'DASHBOARD_RESOURCE', denied)
            self.reject(candidate, runs)
        runs = copy.deepcopy(original)
        def all_expiry(row):
            p = row['phases']['replacement']
            for sample in p['samples']:
                if sample['case'] == 'overview':
                    sample.update(outcome='expiry', status=409, predicate='safe-expiry')
            p['summary'] = summary(p['samples'])
        change_record(runs, 'DASHBOARD_RESOURCE', all_expiry)
        self.reject(candidate, runs)

    def test_each_budget_and_nearest_rank_cannot_be_hidden(self):
        candidate, original = fixture()
        for mode in range(4):
            runs = copy.deepcopy(original)
            def oversized(row):
                row['resources'].update(sampled_combined_peak_rss_bytes=257 << 20, budget_peak_rss_bytes=257 << 20)
            change_record(runs, 'DASHBOARD_RESOURCE', oversized, mode)
            self.reject(candidate, runs)
        for case in CASES:
            runs = copy.deepcopy(original)
            def slow(row):
                p = row['phases']['initial']
                # Eleven of 200 cause the case p95 to fail; aggregate p95 still passes.
                chosen = [x for x in p['samples'] if x['case'] == case][-11:]
                for sample in chosen:
                    sample['duration_ns'] = 250000000
                p['summary'] = summary(p['samples'])
                self.assertLess(p['summary']['all']['success_latency']['p95_ns'], 250000000)
            change_record(runs, 'DASHBOARD_RESOURCE', slow, 1)
            self.reject(candidate, runs)
        # Rank 190, rather than index 190, is required for a 200-sample case.
        runs = copy.deepcopy(original)
        change_record(runs, 'DASHBOARD_RESOURCE', lambda r: r['phases']['initial']['summary']['overview']['success_latency'].update(p95_ns=30000000))
        self.reject(candidate, runs)

    def test_candidate_hash_versions_preservation_and_cleanup_are_independent(self):
        candidate, original = fixture()
        for key in ('source_commit', 'base_commit', 'pr_title', 'compiler_sha256', 'archive_sha256', 'go_version'):
            runs = copy.deepcopy(original)
            bad = copy.deepcopy(candidate)
            bad[key] = 'different'
            self.reject(bad, runs)
        for key, value in [('review_verified', False), ('review_verified', 1)]:
            bad = copy.deepcopy(candidate)
            bad[key] = value
            self.reject(bad, original)
        for mutate in (lambda run: run['record'].update(transcript_sha256='0' * 64),
                       lambda run: run['pins'].update(base='0' * 64),
                       lambda run: run['fixtures']['native-small'].update(datasets=104),
                       lambda run: run['fixtures']['native-small'].update(inventory_sha256='0' * 64),
                       lambda run: run['record']['preservation']['qemu'].update(after='0' * 64),
                       lambda run: run['supervisor'].update(exit_code=1),
                       lambda run: run['supervisor'].update(source_commit='0' * 40),
                       lambda run: run['supervisor'].update(transcript_sha256='0' * 64),
                       lambda run: run['supervisor'].update(group_cleaned=False),
                       lambda run: run['supervisor'].update(stopped_verified=False)):
            runs = copy.deepcopy(original)
            mutate(runs[0])
            self.reject(candidate, runs)
        for key in ('guest_assertions', 'host_teardown', 'fixture_key_removed', 'image_removed',
                    'variables_removed', 'listener_removed', 'qemu_exited', 'qualified'):
            runs = copy.deepcopy(original)
            runs[0]['record'][key] = False
            self.reject(candidate, runs)
        for key, value in [('guest_go', 'go1.26.0'), ('kernel', '14.5-RELEASE'), ('arch', 'arm64'),
                           ('cpu_count', 1), ('memory_mib', 4096), ('accelerator', 'hvf'), ('zfs', '')]:
            runs = copy.deepcopy(original)
            change_record(runs, 'DASHBOARD_PROVENANCE', lambda r: r.update({key: value}))
            self.reject(candidate, runs)
        for key in ('predicate_assertions', 'native_guid_assertions', 'native_owned_count', 'boundary_samples'):
            runs = copy.deepcopy(original)
            change_record(runs, 'DASHBOARD_CONTROLS', lambda r: r.update({key: 0}))
            self.reject(candidate, runs)
        for key in ('guest_cleanup', 'pair_descendants_removed', 'sockets_removed', 'boundary_outside_normal_budget'):
            runs = copy.deepcopy(original)
            change_record(runs, 'DASHBOARD_CONTROLS', lambda r: r.update({key: False}))
            self.reject(candidate, runs)

    def test_fail_closed_json_types_duplicate_keys_unknown_fields_and_limits(self):
        candidate, original = fixture()
        for extra in (',"release":"15.1"}', ',"new_field":1}', ',"physical_bytes":NaN}'):
            runs = copy.deepcopy(original)
            lines = runs[0]['transcript'].decode().splitlines()
            lines[0] = lines[0][:-1] + extra
            runs[0]['transcript'] = '\n'.join(lines).encode()
            bind(runs)
            self.reject(candidate, runs)
        for payload in (b'\xff', b'x' * ((32 << 20) + 1), b'DASHBOARD_RESOURCE {', b'DASHBOARD_RESOURCE []'):
            runs = copy.deepcopy(original)
            runs[0]['transcript'] = payload
            bind(runs)
            self.reject(candidate, runs)
        runs = copy.deepcopy(original)
        runs[0]['record']['unknown'] = True
        self.reject(candidate, runs)
        runs = copy.deepcopy(original)
        change_record(runs, 'DASHBOARD_RESOURCE', lambda r: r['phases']['initial']['summary']['all'].update(scheduled=True))
        self.reject(candidate, runs)
        # A forged old record cannot borrow consistent new samples.
        runs = copy.deepcopy(original)
        change_record(runs, 'PERFORMANCE_RESULT', lambda r: r.update(p95_ms=1))
        self.reject(candidate, runs)

    def test_missing_phase_and_duplicate_modes_with_otherwise_valid_records(self):
        candidate, original = fixture()
        runs = copy.deepcopy(original)
        change_record(runs, 'DASHBOARD_RESOURCE', lambda r: r['phases'].pop('replacement'))
        self.reject(candidate, runs)
        runs = copy.deepcopy(original)
        lines = runs[0]['transcript'].decode().splitlines()
        positions = [i for i, x in enumerate(lines) if x.startswith('DASHBOARD_RESOURCE ')]
        lines[positions[1]] = lines[positions[0]]
        runs[0]['transcript'] = '\n'.join(lines).encode()
        bind(runs)
        self.reject(candidate, runs)

    def test_boundary_evidence_legacy_types_and_predicate_origin_cannot_be_forged(self):
        candidate, original = fixture()
        for marker, mutate in [('PERFORMANCE_RESULT', lambda r: r.update(p95_ms='100')),
                               ('NATIVE_READER_RESULT', lambda r: r.update(mode=[])),
                               ('DASHBOARD_RESOURCE', lambda r: r.update(inventory_sha256='0' * 64)),
                               ('DASHBOARD_RESOURCE', lambda r: r.update(mode='unknown')),
                               ('DASHBOARD_CONTROLS', lambda r: r.update(boundary_sampled_rss_bytes=1))]:
            runs = copy.deepcopy(original)
            change_record(runs, marker, mutate)
            self.reject(candidate, runs)
        runs = copy.deepcopy(original)
        runs[0]['transcript'] = runs[0]['transcript'].replace(b' sampled_rss_bytes=377487360 high_water_sum_bytes=367001600 samples=100 max_gap_ms=1300.0', b'')
        bind(runs)
        self.reject(candidate, runs)


if __name__ == '__main__':
    unittest.main()
