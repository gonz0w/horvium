# SPDX-License-Identifier: BSD-3-Clause
"""Synthetic orchestration/receipt regressions; never native acceptance."""
import copy
import importlib.util
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]


def load(path, name):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class DashboardQualificationTests(unittest.TestCase):
    def setUp(self):
        self.module = load(ROOT / 'tools/dashboard-qualification.py', 'dashboard_qualification')
        fixture = load(Path(__file__).with_name('test_dashboard_resource_evidence.py'), 'original_receipt_fixture')
        self.candidate, self.runs = fixture.fixture()
        self.run = self.runs[0]
        self.provenance = self.run['record']['provenance']
        lines = self.run['transcript'].decode().splitlines()
        self.raw = ('\n'.join(line for line in lines if not line.startswith(('DASHBOARD_PROVENANCE ', 'DASHBOARD_CONTROLS '))) + '\n').encode()
        self.originals = {mode: {'schema': 1, 'mode': mode, 'fixture': original}
                          for mode, original in self.run['fixtures'].items()}
        self.parents = {}
        for index, parent in enumerate(self.module.c.PARENTS):
            modes = self.module.c.MODES[:2] if index == 0 else self.module.c.MODES[2:]
            controls = {'parent': parent, 'modes': {mode: {'fixture': self.run['fixtures'][mode],
                        'predicate_assertions': 2000, 'replacements': 3,
                        'native_guid_assertions': 100 if mode == 'native-small' else 0} for mode in modes},
                        'predicate_assertions': 4000}
            if index == 0:
                controls['boundary'] = {'accepted_model_records': 100000, 'record_exhaustions': 1,
                    'byte_exhaustions': 1, 'recoveries': 1, 'sampled_rss_bytes': 360 << 20,
                    'high_water_sum_bytes': 350 << 20, 'samples': 100,
                    'max_sample_gap_ns': 1300000000, 'outside_normal_budget': True}
            else:
                controls['native'] = {'captured_commands': 1, 'subtree_exits': 1, 'stale_retentions': 1,
                                      'recoveries': 1, 'boot_state': 'unchanged'}
            self.parents[parent] = {'schema': 1, 'controls': controls,
                'artifacts': {'artifacts': self.provenance['artifacts'], 'go_version': self.provenance['guest_go']}, 'cleanup': {
                'descendants_removed': True, 'sockets_removed': True, 'temporary_directories_removed': True,
                'storage_cleanups': 0 if index == 0 else 1, 'pairs': 3 if index == 0 else 2}}

        self.raw += ''.join('    performance_freebsd_test.go:123: NATIVE_DASHBOARD_CONTROLS ' + json.dumps(self.parents[p]['controls']) + '\n' for p in self.module.c.PARENTS).encode()

    def assemble(self):
        return self.module.assemble(self.raw, self.provenance, self.originals, self.parents)

    def test_original_sidecars_assemble_without_modifying_inputs(self):
        before = copy.deepcopy((self.raw, self.provenance, self.originals, self.parents))
        payload, fixtures = self.assemble()
        run = copy.deepcopy(self.run)
        run['transcript'], run['fixtures'] = payload, fixtures
        run['record']['transcript_sha256'] = self.module.sha(payload)
        run['supervisor']['transcript_sha256'] = self.module.sha(payload)
        self.module.c.validate_run(self.candidate, run)
        self.assertEqual(before, (self.raw, self.provenance, self.originals, self.parents))

    def test_resource_record_cannot_repair_missing_or_changed_originals(self):
        for mutation in ('missing', 'digest', 'count', 'origin', 'duplicate-mode'):
            with self.subTest(mutation=mutation):
                saved = copy.deepcopy(self.originals)
                if mutation == 'missing':
                    del self.originals['native-small']
                elif mutation == 'duplicate-mode':
                    self.originals['native-small']['mode'] = 'synthetic-100'
                else:
                    key, value = {'digest': ('inventory_sha256', 'f' * 64), 'count': ('datasets', 999),
                                  'origin': ('origin', 'generated')}[mutation]
                    self.originals['native-small']['fixture'][key] = value
                with self.assertRaises(ValueError):
                    self.assemble()
                self.originals = saved

    def test_partial_parent_controls_and_cleanup_never_qualify(self):
        parent = self.module.c.PARENTS[1]
        for mutation in ('missing', 'predicates', 'guid', 'recovery', 'descendants_removed',
                         'sockets_removed', 'temporary_directories_removed', 'storage_cleanups', 'pairs'):
            with self.subTest(mutation=mutation):
                saved = copy.deepcopy(self.parents)
                if mutation == 'missing':
                    del self.parents[parent]
                elif mutation == 'predicates':
                    self.parents[parent]['controls']['predicate_assertions'] = 3999
                elif mutation == 'guid':
                    self.parents[parent]['controls']['modes']['native-small']['native_guid_assertions'] = 99
                elif mutation == 'recovery':
                    self.parents[parent]['controls']['native']['recoveries'] = 0
                else:
                    self.parents[parent]['cleanup'][mutation] = False
                with self.assertRaises(ValueError):
                    self.assemble()
                self.parents = saved

    def test_raw_failures_unknown_records_and_forged_assembled_records_are_preserved_or_denied(self):
        saved = self.raw
        for extra in ('FAIL\n', '--- SKIP: TestNativePerformanceBudgets (0.00s)\n',
                      'DASHBOARD_UNKNOWN {}\n', 'DASHBOARD_CONTROLS {}\n', 'DASHBOARD_PROVENANCE {}\n'):
            self.raw = saved + extra.encode()
            with self.assertRaises(ValueError):
                payload, fixtures = self.assemble()
                run = copy.deepcopy(self.run)
                run['transcript'], run['fixtures'] = payload, fixtures
                run['record']['transcript_sha256'] = self.module.sha(payload)
                run['supervisor']['transcript_sha256'] = self.module.sha(payload)
                self.module.c.validate_run(self.candidate, run)
        self.raw = saved

    def test_owned_transfer_is_fixed_and_never_uses_supervisor_lease_as_stdin(self):
        with tempfile.TemporaryDirectory() as directory:
            lab = Path(directory)
            for name in ('source.tar', 'horvium', 'reader-fixture.test', 'dashboard-native.test'):
                argv, remote = self.module.transfer_operation(name)
                self.assertEqual(argv, ['transfer', name])
                self.assertNotIn(str(lab), remote)
            for name in ('../../outside', 'fixture-key', 'bad; touch /tmp/bad', ''):
                with self.assertRaises(ValueError):
                    self.module.transfer_operation(name)

    def test_changed_original_launch_artifacts_cannot_be_repaired_by_resource_success(self):
        for parent in self.module.c.PARENTS:
            saved = copy.deepcopy(self.parents)
            self.parents[parent]['artifacts']['artifacts'] = dict(self.provenance['artifacts'], management='f' * 64)
            with self.assertRaises(ValueError):
                self.assemble()
            self.parents = saved

    def test_failed_candidate_build_retains_failure_and_removes_outputs_without_starting_guest(self):
        with tempfile.TemporaryDirectory(dir='/private/tmp' if Path('/private/tmp').is_dir() else None) as temporary:
            root = Path(temporary).resolve()
            lab = root / '.tools/lab-runs/performance-15.1-test'
            lab.mkdir(parents=True, mode=0o700)
            candidate = root / '.tools/dashboard-candidates' / lab.name
            def failed_build(*args, **kwargs):
                candidate.mkdir(parents=True, mode=0o700)
                for name in ('source.tar', *self.module.build.ARTIFACT_FILES.values()):
                    (candidate / name).write_bytes(b'partial build')
                raise ValueError('original build failure')
            identity = {k: self.candidate[k] for k in ('source_commit', 'base_commit', 'pr_title')}
            with patch.object(self.module, 'ROOT', root), patch.object(self.module.build, 'build_candidate', side_effect=failed_build), \
                    patch.object(self.module.subprocess, 'Popen') as guest:
                with self.assertRaisesRegex(ValueError, 'original build failure'):
                    self.module.release_worker('15.1', lab, identity, root / '.tools/review-runs/test')
                guest.assert_not_called()
            record = json.loads((lab / 'result.json').read_text())
            self.assertFalse(record['qualified'])
            self.assertIn('original build failure', record['failure'])
            self.assertFalse(record['host_teardown'])
            self.assertTrue(all(not (candidate / name).exists() for name in ('source.tar', *self.module.build.ARTIFACT_FILES.values())))

    def test_context_selects_exact_review_before_any_guest_work(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary).resolve()
            identity = {'source_commit': self.candidate['source_commit'], 'base_commit': self.candidate['base_commit'],
                        'source_tree': '9' * 40, 'title': self.candidate['pr_title'], 'signed_commits': [self.candidate['source_commit']]}
            for name, record in [('exact', dict(identity, qualified=True)),
                                 ('wrong', dict(identity, qualified=True, source_commit='f' * 40)),
                                 ('failed', dict(identity, qualified=False))]:
                path = root / '.tools/review-runs' / name
                path.mkdir(parents=True, mode=0o700)
                self.module.owned.write_record(path / 'result.json', record)
            with patch.dict('os.environ', {'PR_TITLE': identity['title']}), patch.object(self.module.build.review, 'identity', return_value=identity), \
                    patch.object(self.module.build, 'verify_review') as verify:
                selected, path = self.module.context(root)
            self.assertEqual(path.name, 'exact')
            self.assertEqual(selected, {k: self.candidate[k] for k in ('source_commit', 'base_commit', 'pr_title')})
            verify.assert_called_once_with(root, path, **selected)

    def saved_worker(self, root):
        """Explicitly synthetic opened files and supervision, never a native receipt."""
        lab = root / '.tools/lab-runs/performance-15.1-synthetic'
        lab.mkdir(parents=True, mode=0o700)
        candidate_directory = root / '.tools/dashboard-candidates' / lab.name
        candidate_directory.mkdir(parents=True, mode=0o700)
        runtime = {}
        for name in ('qemu', 'firmware_code', 'firmware_vars'):
            runtime[name] = root / name
            runtime[name].write_bytes(('synthetic-' + name).encode())
        pins = dict(self.run['pins'], **{k: self.module.sha(p.read_bytes()) for k, p in runtime.items()})
        host = {'schema': 1, 'candidate': self.candidate, 'pins': pins, 'port': 12345,
                'runtime_paths': {k: str(p) for k, p in runtime.items()},
                'host_os': 'Darwin', 'host_arch': 'arm64', 'qemu_version': 'fixture-only',
                'candidate_build_sha256': 'a' * 64}
        self.module.owned.write_record(lab / 'host-original.json', host)
        ssh = self.module.runner.ssh_command(lab / 'fixture-key', lab / 'known_hosts', host['port'])
        counter = 0
        def supervision(argv, start, end):
            nonlocal counter
            counter += 1
            directory = root / '.tools/process-runs' / str(counter)
            directory.mkdir(parents=True, mode=0o700)
            state = {'completed': True, 'group_cleaned': True, 'command_sha256': self.module.sha(json.dumps(argv).encode()),
                     'group': 200 + counter, 'started_at': start, 'escalated': False,
                     'worker_returncode': 0, 'reason': 'exit', 'completed_at': end}
            self.module.owned.write_record(directory / 'state.json', state)
            return directory, state
        record = {'suite': 'performance', 'release': '15.1', 'source_commit': self.candidate['source_commit'],
                  'base_commit': self.candidate['base_commit'], 'pr_title': self.candidate['pr_title'],
                  'candidate': self.candidate, 'qualified': False, 'host_teardown': True,
                  'fixture_key_removed': True, 'image_removed': True, 'variables_removed': True,
                  'listener_removed': True, 'qemu_exited': True, 'port': host['port'],
                  'resource_commands_complete': True, 'commands': [],
                  'build': {'artifacts_removed': False, 'supervisors': [{'completed_at': 2}] * 6},
                  'provenance': self.provenance,
                  'preservation': {k: {'before': v, 'after': v} for k, v in pins.items()}}
        host['candidate_build_sha256'] = self.module.sha((json.dumps(record['build'], indent=2) + '\n').encode())
        self.module.owned.write_record(lab / 'host-original.json', host)
        self.module.owned.write_record(candidate_directory / 'result.json', dict(record['build'], artifacts_removed=True))
        hashes = {'source.tar': self.candidate['archive_sha256'], **{self.module.build.ARTIFACT_FILES[k]: v for k, v in self.candidate['artifacts']['15.1'].items()}}
        for name, expected in hashes.items():
            argv = [self.module.sys.executable, str(Path(self.module.__file__).resolve()), 'transfer', name,
                    str(candidate_directory), str(lab), str(host['port']), expected]
            directory, state = supervision(argv, 3, 4)
            record['commands'].append({'operation': 'transfer-' + name, 'directory': str(directory), 'state': state})
        preflight = self.module.preflight.preflight_command(root, lab, host['port'])
        first_directory, _ = supervision(preflight, 5, 6)
        resource_directory, resource_state = supervision(ssh + [self.module.REMOTE], 7, 8)
        after_directory, _ = supervision(preflight, 9, 10)
        observed = {k: self.provenance[k] for k in ('artifacts', 'kernel', 'userland', 'zfs', 'physical_bytes')}
        observed['go_version'] = self.provenance['guest_go']
        payload = ('=== RUN   TestNativeDashboardArtifactProvenance\n    dashboard_artifacts_freebsd_test.go:88: DASHBOARD_GUEST_ARTIFACTS '
                   + json.dumps(observed) + '\n--- PASS: TestNativeDashboardArtifactProvenance (0.10s)\nPASS\n').encode()
        for name in ('dashboard-preflight.log', 'dashboard-preflight-after.log'):
            (lab / name).write_bytes(payload)
        record.update(preflight={'observed': observed, 'transcript_sha256': self.module.sha(payload),
                                 'candidate_build_sha256': host['candidate_build_sha256']},
                      preflight_supervisor_directory=str(first_directory),
                      after_preflight_supervisor_directory=str(after_directory),
                      after_preflight_sha256=self.module.sha(payload),
                      resource_supervisor={'directory': str(resource_directory), 'state': resource_state})
        observations = []
        for name, filename in self.module.ORIGINAL_NAMES.items():
            row = (self.originals if name in self.module.c.MODES else self.parents)[name]
            self.module.owned.write_record(lab / filename, row)
            directory, state = supervision(ssh + ['cat /root/work/.tools/' + filename], 11, 12)
            observations.append({'name': filename, 'sha256': self.module.sha((lab / filename).read_bytes()),
                                 'directory': str(directory), 'state': state})
        payload, fixtures = self.assemble()
        (lab / 'resource-raw.log').write_bytes(self.raw)
        (lab / 'qualification.log').write_bytes(payload)
        record.update(originals=observations, transcript_sha256=self.module.sha(payload),
                      raw_transcript_sha256=self.module.sha(self.raw), fixtures=fixtures)
        self.module.owned.write_record(lab / 'result.json', record)
        identity = {k: record[k] for k in ('source_commit', 'base_commit', 'pr_title')}
        worker = [self.module.sys.executable, str(root / 'tools/local-qualification.py'), 'performance', '15.1', '--run', '--owned-directory', str(lab)]
        directory, _ = supervision(worker, 13, 14)
        return lab, identity, directory, worker, record

    def test_finalization_reopens_original_transfers_launches_sidecars_and_teardown(self):
        with tempfile.TemporaryDirectory(dir='/private/tmp' if Path('/private/tmp').is_dir() else None) as temporary:
            root = Path(temporary).resolve()
            lab, identity, directory, worker, _ = self.saved_worker(root)
            with patch.object(self.module, 'ROOT', root), patch.object(self.module, 'listener_absent', return_value=True), \
                    patch.object(self.module.build, 'source_identity'):
                run = self.module.finalize(lab, identity, '15.1', directory, worker)
            self.assertEqual(run['release'], '15.1')
            self.assertTrue(json.loads((lab / 'result.json').read_text())['qualified'])
            self.assertNotIn('native_acceptance', run['record'])

    def test_late_original_mutations_and_incomplete_host_cleanup_deny_finalization(self):
        for mutation in ('transfer', 'resource', 'sidecar', 'runtime', 'listener', 'key', 'builder', 'raw', 'host-pins'):
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory(dir='/private/tmp' if Path('/private/tmp').is_dir() else None) as temporary:
                root = Path(temporary).resolve()
                lab, identity, directory, worker, record = self.saved_worker(root)
                if mutation in ('transfer', 'resource'):
                    path = Path(record['commands'][0]['directory'] if mutation == 'transfer' else record['resource_supervisor']['directory']) / 'state.json'
                    row = json.loads(path.read_text())
                    row['command_sha256'] = 'f' * 64
                    self.module.owned.write_record(path, row)
                elif mutation == 'sidecar':
                    (lab / self.module.ORIGINAL_NAMES['native-small']).write_text('{}')
                elif mutation == 'runtime':
                    (root / 'qemu').write_bytes(b'changed')
                elif mutation == 'key':
                    (lab / 'fixture-key').write_text('synthetic leftover key')
                elif mutation == 'builder':
                    (root / '.tools/dashboard-candidates' / lab.name / 'result.json').write_text('{}')
                elif mutation == 'raw':
                    with (lab / 'resource-raw.log').open('ab') as log:
                        log.write(b'FAIL\n')
                elif mutation == 'host-pins':
                    row = json.loads((lab / 'host-original.json').read_text())
                    row['pins']['base'] = 'f' * 64
                    self.module.owned.write_record(lab / 'host-original.json', row)
                with patch.object(self.module, 'ROOT', root), patch.object(self.module, 'listener_absent', return_value=mutation != 'listener'), \
                        patch.object(self.module.build, 'source_identity'):
                    with self.assertRaises(ValueError):
                        self.module.finalize(lab, identity, '15.1', directory, worker)
                self.assertFalse(json.loads((lab / 'result.json').read_text())['qualified'])


if __name__ == '__main__':
    unittest.main()
